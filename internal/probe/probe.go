// Package probe runs the infra_probe mini-pipeline and optional pre-destructive gates.
package probe

import (
	"context"
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/validate"
)

// ShouldGate reports whether command should run infra probe before the main pipeline (live only).
func ShouldGate(cfg *config.Config, command string) bool {
	if cfg == nil || !cfg.InfraProbe.Enabled || cfg.Run.Mode != "live" {
		return false
	}
	for _, c := range cfg.InfraProbe.Before {
		if c == command {
			return true
		}
	}
	return false
}

// Run executes probe up → checks → probe down. Writes cache on success in live mode.
func Run(ctx context.Context, cfg *config.Config, eng *engine.Engine, factory validate.ClientFactory, emit *log.Emitter) error {
	if cfg == nil {
		return fmt.Errorf("infra probe: no config")
	}
	if len(cfg.InfraProbe.Pipeline.Up) == 0 {
		return fmt.Errorf("infra probe: pipeline.up is required")
	}
	dryRun := cfg.Run.Mode == "dry-run"
	if !dryRun && cfg.InfraProbe.CacheTTL > 0 {
		fresh, err := IsFresh(cfg)
		if err == nil && fresh {
			logProbe(emit, cfg, "skipped (cache fresh)")
			return nil
		}
	}

	logProbe(emit, cfg, "starting")
	upErr := eng.RunSteps(ctx, cfg, engine.PhaseUp, cfg.InfraProbe.Pipeline.Up)
	upOK := upErr == nil
	if upErr != nil {
		logProbe(emit, cfg, fmt.Sprintf("up failed: %v", upErr))
		return fmt.Errorf("infra probe: up: %w", upErr)
	}

	if err := RunChecks(ctx, cfg, factory, dryRun, upOK); err != nil {
		logProbe(emit, cfg, fmt.Sprintf("checks failed: %v", err))
		downErr := runProbeDown(ctx, cfg, eng, emit)
		if downErr != nil {
			return fmt.Errorf("infra probe: %w (teardown: %v)", err, downErr)
		}
		return fmt.Errorf("infra probe: %w", err)
	}

	if err := runProbeDown(ctx, cfg, eng, emit); err != nil {
		return err
	}

	if !dryRun {
		if err := WriteOK(cfg); err != nil {
			logProbe(emit, cfg, fmt.Sprintf("cache write failed: %v", err))
		}
	}
	logProbe(emit, cfg, "ok")
	return nil
}

// RunGate runs the probe before a destructive command. Respects fail_fast.
func RunGate(ctx context.Context, cfg *config.Config, eng *engine.Engine, factory validate.ClientFactory, emit *log.Emitter, command string) error {
	logProbe(emit, cfg, fmt.Sprintf("gate before %s", command))
	err := Run(ctx, cfg, eng, factory, emit)
	if err == nil {
		return nil
	}
	if cfg.InfraProbe.FailFast {
		return err
	}
	logProbe(emit, cfg, fmt.Sprintf("failed (fail_fast=false, continuing %s): %v", command, err))
	return nil
}

func runProbeDown(ctx context.Context, cfg *config.Config, eng *engine.Engine, emit *log.Emitter) error {
	if len(cfg.InfraProbe.Pipeline.Down) == 0 {
		return nil
	}
	logProbe(emit, cfg, "teardown down")
	if err := eng.RunSteps(ctx, cfg, engine.PhaseDown, cfg.InfraProbe.Pipeline.Down); err != nil {
		return fmt.Errorf("infra probe: down: %w", err)
	}
	return nil
}

func logProbe(emit *log.Emitter, cfg *config.Config, msg string) {
	if emit == nil || cfg == nil {
		return
	}
	full := "infra probe: " + msg
	if cfg.Run.Mode == "dry-run" {
		emit.DryRun(cfg, full)
	} else {
		emit.Live(full)
	}
}
