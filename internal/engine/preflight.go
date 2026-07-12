package engine

import (
	"context"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/preflight"
)

func (e *Engine) runPreflight(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	switch cfg.Run.Mode {
	case "dry-run":
		if e.Log != nil {
			e.Log.DryRun(cfg, preflight.DryRunLine)
		}
		return nil
	case "live":
		if err := preflight.Check(ctx, cfg, nil); err != nil {
			return err
		}
		return nil
	default:
		return nil
	}
}
