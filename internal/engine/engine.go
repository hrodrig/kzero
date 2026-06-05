package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/notify"
	"github.com/hrodrig/kzero/internal/retry"
)

// Engine runs phased pipelines using a Runner (dry-run or live).
type Engine struct {
	Runner  Runner
	Log     *log.Emitter
	Command string    // CLI command name: down, up, reset (for notify metadata)
	Started time.Time // pipeline start time (for notify metadata)
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
	return e.runDown(ctx, cfg)
}

// RunUp runs pre-up, pipelines.up, then post-up. Fail-fast; on-error hook runs on failure.
func (e *Engine) RunUp(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := withRunTimeout(ctx, cfg)
	defer cancel()
	return e.runUp(ctx, cfg)
}

// RunReset runs a full down then up under a single run.timeout budget. If down fails, up is not executed.
func (e *Engine) RunReset(ctx context.Context, cfg *config.Config) error {
	ctx, cancel := withRunTimeout(ctx, cfg)
	defer cancel()
	if err := e.runDown(ctx, cfg); err != nil {
		return fmt.Errorf("reset down: %w", err)
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
	if err := e.Runner.RunHook(ctx, cfg, "pre-down", cfg.Hooks.PreDown); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "pre-down", Err: err})
	}
	for i, step := range cfg.Pipelines.Down {
		if err := e.runPipelineStepWithRetry(ctx, cfg, PhaseDown, i, step); err != nil {
			return finishWithError(ctx, e, cfg, &PipelineError{Phase: string(PhaseDown), Index: i, Ref: step.Ref, Err: err})
		}
	}
	if err := e.Runner.RunHook(ctx, cfg, "post-down", cfg.Hooks.PostDown); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "post-down", Err: err})
	}
	return nil
}

func (e *Engine) runUp(ctx context.Context, cfg *config.Config) error {
	if err := e.runPreflight(ctx, cfg); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Err: err})
	}
	if err := e.Runner.RunHook(ctx, cfg, "pre-up", cfg.Hooks.PreUp); err != nil {
		return finishWithError(ctx, e, cfg, &PipelineError{Hook: "pre-up", Err: err})
	}
	for i, step := range cfg.Pipelines.Up {
		if err := e.runPipelineStepWithRetry(ctx, cfg, PhaseUp, i, step); err != nil {
			return finishWithError(ctx, e, cfg, &PipelineError{Phase: string(PhaseUp), Index: i, Ref: step.Ref, Err: err})
		}
	}
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
	dispatchPipelineError(ctx, eng, cfg, err)
	if cfg.Hooks.OnError == "" {
		return err
	}
	if hookErr := eng.Runner.RunHook(ctx, cfg, "on-error", cfg.Hooks.OnError); hookErr != nil {
		return fmt.Errorf("%w: on-error hook: %v", err, hookErr)
	}
	return err
}

func dispatchPipelineError(ctx context.Context, eng *Engine, cfg *config.Config, err error) {
	if eng == nil || err == nil {
		return
	}
	started := eng.Started
	if started.IsZero() {
		started = time.Now()
	}
	meta := notify.MetaFromConfig(cfg, eng.Command, started, time.Since(started))
	meta.Error = err.Error()
	var pe *PipelineError
	if errors.As(err, &pe) {
		meta.FailedStep = pe.FailedStep()
	}
	_ = notify.Dispatch(ctx, cfg, notify.EventError, meta, nil)
}
