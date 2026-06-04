package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/config"
)

// Engine runs phased pipelines using a Runner (dry-run or live).
type Engine struct {
	Runner Runner
}

// New builds an Engine for cfg.Run.Mode, writing dry-run / log lines to out.
func New(cfg *config.Config, out io.Writer) *Engine {
	var r Runner
	switch cfg.Run.Mode {
	case "dry-run":
		r = NewDryRunner(cfg, out)
	case "live":
		r = &LiveRunner{Out: out}
	default:
		r = NewDryRunner(cfg, out)
	}
	return &Engine{Runner: r}
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
	if err := e.Runner.RunHook(ctx, cfg, "pre-down", cfg.Hooks.PreDown); err != nil {
		return finishWithError(ctx, e.Runner, cfg, err)
	}
	for i, step := range cfg.Pipelines.Down {
		if err := e.Runner.RunPipelineStep(ctx, cfg, PhaseDown, i, step); err != nil {
			return finishWithError(ctx, e.Runner, cfg, err)
		}
	}
	if err := e.Runner.RunHook(ctx, cfg, "post-down", cfg.Hooks.PostDown); err != nil {
		return finishWithError(ctx, e.Runner, cfg, err)
	}
	return nil
}

func (e *Engine) runUp(ctx context.Context, cfg *config.Config) error {
	if err := e.Runner.RunHook(ctx, cfg, "pre-up", cfg.Hooks.PreUp); err != nil {
		return finishWithError(ctx, e.Runner, cfg, err)
	}
	for i, step := range cfg.Pipelines.Up {
		if err := e.Runner.RunPipelineStep(ctx, cfg, PhaseUp, i, step); err != nil {
			return finishWithError(ctx, e.Runner, cfg, err)
		}
	}
	if err := e.Runner.RunHook(ctx, cfg, "post-up", cfg.Hooks.PostUp); err != nil {
		return finishWithError(ctx, e.Runner, cfg, err)
	}
	return nil
}

func withRunTimeout(ctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	if cfg.Run.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, cfg.Run.Timeout)
}

func finishWithError(ctx context.Context, runner Runner, cfg *config.Config, err error) error {
	if cfg.Hooks.OnError == "" {
		return err
	}
	if hookErr := runner.RunHook(ctx, cfg, "on-error", cfg.Hooks.OnError); hookErr != nil {
		return fmt.Errorf("%w: on-error hook: %v", err, hookErr)
	}
	return err
}
