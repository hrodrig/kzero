package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/notify"
	"github.com/hrodrig/kzero/internal/redact"
	"github.com/hrodrig/kzero/internal/retry"
	"github.com/hrodrig/kzero/internal/validate"
	"github.com/hrodrig/kzero/internal/watchdog"
)

// Engine runs phased pipelines using a Runner (dry-run or live).
type Engine struct {
	Runner  Runner
	Log     *log.Emitter
	Command string    // CLI command name: down, up, reset (for notify metadata)
	Started time.Time // pipeline start time (for notify metadata)

	// PreflightFactory overrides live preflight client construction when set.
	// Production callers leave it nil (uses validate.ClientFactoryDefault).
	PreflightFactory validate.ClientFactory

	// stalled is set when the API watchdog trips, causing the pipeline
	// to be aborted due to sustained API unreachability. dispatchPipelineError
	// checks this to emit EventStalled instead of EventError.
	stalled bool

	progressMu sync.Mutex
	progress   pipelineProgress
}

type pipelineProgress struct {
	Phase string
	Index int
	Ref   string
	Hook  string
}

// Stalled returns true when the engine aborted a pipeline due to API
// unreachability (watchdog trip).
func (e *Engine) Stalled() bool {
	return e != nil && e.stalled
}

// startAPIObserver returns a derived context and a watchdog that periodically
// probes the API server. When the watchdog trips (cumulative unreachability
// exceeds cfg.Run.APIWatchdog.FailAfter), the derived context is cancelled,
// causing the running pipeline step to fail with context.Canceled.
// Returns the original context and nil watchdog when api_watchdog is
// disabled or when the REST config cannot be loaded (non-fatal).
func (e *Engine) startAPIObserver(ctx context.Context, cfg *config.Config) (context.Context, *watchdog.Watchdog) {
	if cfg == nil || cfg.Run.APIWatchdog == nil || !cfg.Run.APIWatchdog.Enabled {
		return ctx, nil
	}

	client, healthzURL, err := newAPIWatchdogProbe(cfg.Run.Kubeconfig)
	if err != nil {
		e.warnAPIWatchdogDisabled("api_watchdog: watchdog disabled", err)
		return ctx, nil
	}

	interval, failAfter := apiWatchdogTimings(cfg.Run.APIWatchdog)
	stepCtx, stepCancel := context.WithCancel(ctx)

	w := watchdog.New(stepCtx, watchdog.Config{
		Interval:  interval,
		FailAfter: failAfter,
		Healthz: func(probeCtx context.Context) error {
			return probeKubernetesHealthz(probeCtx, client, healthzURL)
		},
		OnTrip: func() {
			e.stalled = true
			stepCancel()
		},
	})

	return stepCtx, w
}

// New builds an Engine for cfg.Run.Mode, writing dry-run / live lines to emit.
func New(cfg *config.Config, emit *log.Emitter) *Engine {
	var r Runner
	switch cfg.Run.Mode {
	case "dry-run":
		r = NewDryRunner(cfg, emit)
	case "live":
		r = &LiveRunner{Log: emit}
	default:
		r = NewDryRunner(cfg, emit)
	}
	return &Engine{Runner: r, Log: emit}
}

// RunDown runs pre-down, pipelines.down, then post-down. Fail-fast; on-error hook runs on failure.
func (e *Engine) RunDown(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := withRunTimeout(ctx, cfg)
	defer cancel()
	ctx, wd := e.startAPIObserver(ctx, cfg)
	if wd != nil {
		defer wd.Stop()
	}
	return e.runDown(ctx, cfg)
}

// RunUp runs pre-up, pipelines.up, then post-up. Fail-fast; on-error hook runs on failure.
func (e *Engine) RunUp(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := withRunTimeout(ctx, cfg)
	defer cancel()
	ctx, wd := e.startAPIObserver(ctx, cfg)
	if wd != nil {
		defer wd.Stop()
	}
	return e.runUp(ctx, cfg)
}

// RunReset runs a full down then up under a single run.timeout budget. If down fails, up is not executed.
func (e *Engine) RunReset(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := withRunTimeout(ctx, cfg)
	defer cancel()
	ctx, wd := e.startAPIObserver(ctx, cfg)
	if wd != nil {
		defer wd.Stop()
	}
	if err := e.runDown(ctx, cfg); err != nil {
		return fmt.Errorf("reset down: %w", err)
	}
	// Phase-boundary preflight (#37): re-check API after down, before up.
	if err := e.runPreflight(ctx, cfg); err != nil {
		return fmt.Errorf("reset phase preflight: %w", err)
	}
	if err := e.runUp(ctx, cfg); err != nil {
		return fmt.Errorf("reset up: %w", err)
	}
	return nil
}

func (e *Engine) runDown(ctx context.Context, cfg *config.Config) error {
	if err := e.runPreflight(ctx, cfg); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Err: err})
	}
	e.setProgressHook("pre-down")
	if err := e.Runner.RunHook(ctx, cfg, "pre-down", cfg.Hooks.PreDown); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "pre-down", Err: err})
	}
	for i, step := range cfg.Pipelines.Down {
		e.setProgressStep(PhaseDown, i, step.Ref)
		if err := e.runPipelineStepWithRetry(ctx, cfg, PhaseDown, i, step); err != nil {
			return finishWithError(ctx, e, cfg, &PipelineError{Phase: string(PhaseDown), Index: i, Ref: step.Ref, Err: err})
		}
	}
	e.setProgressHook("post-down")
	if err := e.Runner.RunHook(ctx, cfg, "post-down", cfg.Hooks.PostDown); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "post-down", Err: err})
	}
	return nil
}

func (e *Engine) runUp(ctx context.Context, cfg *config.Config) error {
	if err := e.runPreflight(ctx, cfg); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Err: err})
	}
	e.setProgressHook("pre-up")
	if err := e.Runner.RunHook(ctx, cfg, "pre-up", cfg.Hooks.PreUp); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "pre-up", Err: err})
	}
	for i, step := range cfg.Pipelines.Up {
		e.setProgressStep(PhaseUp, i, step.Ref)
		if err := e.runPipelineStepWithRetry(ctx, cfg, PhaseUp, i, step); err != nil {
			return finishWithError(ctx, e, cfg, &PipelineError{Phase: string(PhaseUp), Index: i, Ref: step.Ref, Err: err})
		}
	}
	e.setProgressHook("post-up")
	if err := e.Runner.RunHook(ctx, cfg, "post-up", cfg.Hooks.PostUp); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "post-up", Err: err})
	}
	return nil
}

func withRunTimeout(ctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	if cfg.Run.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, cfg.Run.Timeout)
}

func (e *Engine) runPipelineStepWithRetry(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	max := retry.Attempts(cfg)
	if max <= 1 || cfg.Run.Mode != "live" {
		return e.Runner.RunPipelineStep(ctx, cfg, phase, index, step)
	}
	var lastErr error
	for try := 1; try <= max; try++ {
		lastErr = e.Runner.RunPipelineStep(ctx, cfg, phase, index, step)
		if lastErr == nil {
			return nil
		}
		if try == max || !retry.IsRetriable(lastErr) {
			return lastErr
		}
		wait := retry.Backoff(cfg.Retry.Delay, try)
		if e.Log != nil {
			e.Log.Retry(cfg, string(phase), index, step.Ref, try, max, wait, lastErr)
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastErr
		case <-timer.C:
		}
	}
	return lastErr
}

func finishWithError(ctx context.Context, eng *Engine, cfg *config.Config, err error) error {
	if isUserInterrupt(eng, err) {
		eng.logPipelineInterrupted()
	}
	if dispatchErr := dispatchPipelineError(ctx, eng, cfg, err); dispatchErr != nil {
		return fmt.Errorf("%w: %w", err, dispatchErr)
	}
	if cfg.Hooks.OnError == "" {
		return err
	}
	if hookErr := eng.Runner.RunHook(ctx, cfg, "on-error", cfg.Hooks.OnError); hookErr != nil {
		return fmt.Errorf("%w: on-error hook: %v", err, hookErr)
	}
	return err
}

// ErrPipelineStalled is a sentinel error returned by the engine when a
// pipeline is aborted because the API watchdog detected sustained
// unreachability. It triggers EventStalled instead of EventError.
var ErrPipelineStalled = errors.New("pipeline stalled: API unreachable")

func dispatchPipelineError(ctx context.Context, eng *Engine, cfg *config.Config, err error) error {
	if eng == nil || err == nil {
		return nil
	}
	started := eng.Started
	if started.IsZero() {
		started = time.Now()
	}
	meta := notify.MetaFromConfig(cfg, eng.Command, started, time.Since(started))
	meta.Error = redact.String(err.Error())
	var pe *PipelineError
	if errors.As(err, &pe) {
		meta.FailedStep = pe.FailedStep()
	}
	// Use pipeline.stalled when the pipeline was cancelled by the API
	// watchdog (step context cancelled, not user-initiated).
	event := notify.EventError
	if eng.Stalled() {
		event = notify.EventStalled
	}
	if dispatchErr := notify.Dispatch(ctx, cfg, event, meta, nil); dispatchErr != nil {
		// #35: surface dispatch failures. Emit() redacts Msg and Err so
		// webhook URLs / bearer tokens do not leak into the log stream.
		if eng.Log != nil {
			eng.Log.Emit(log.Entry{
				Kind:  log.KindLive,
				Level: log.LevelError,
				Msg:   "notify dispatch failed (" + event + ")",
				Err:   dispatchErr.Error(),
			})
		}
		if config.RequireNotifyDelivery(cfg) {
			return fmt.Errorf("notify delivery required (%s): %w", event, dispatchErr)
		}
	}
	return nil
}

func (e *Engine) setProgressHook(hook string) {
	if e == nil {
		return
	}
	e.progressMu.Lock()
	e.progress = pipelineProgress{Hook: hook}
	e.progressMu.Unlock()
}

func (e *Engine) setProgressStep(phase Phase, index int, ref string) {
	if e == nil {
		return
	}
	e.progressMu.Lock()
	e.progress = pipelineProgress{Phase: string(phase), Index: index, Ref: ref}
	e.progressMu.Unlock()
}

func isUserInterrupt(eng *Engine, err error) bool {
	if err == nil || (eng != nil && eng.Stalled()) {
		return false
	}
	return errors.Is(err, context.Canceled)
}

func (e *Engine) logPipelineInterrupted() {
	if e == nil || e.Log == nil {
		return
	}
	e.progressMu.Lock()
	p := e.progress
	e.progressMu.Unlock()

	msg := "pipeline interrupted (signal received)"
	if p.Hook != "" {
		msg += fmt.Sprintf("; last hook %s", p.Hook)
	} else if p.Ref != "" {
		msg += fmt.Sprintf("; last step %s (%s[%d])", p.Ref, p.Phase, p.Index)
	} else if p.Phase != "" {
		msg += fmt.Sprintf("; last step %s[%d]", p.Phase, p.Index)
	}
	e.Log.Emit(log.Entry{
		Kind:  log.KindLive,
		Level: log.LevelWarn,
		Msg:   msg,
	})
}
