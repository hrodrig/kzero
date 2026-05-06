package engine

import (
	"context"

	"github.com/hrodrig/kzero/internal/config"
)

// Runner performs hooks and pipeline steps. Implementations honor cfg.Run.Mode
// (dry-run vs live) and must respect ctx cancellation.
type Runner interface {
	RunHook(ctx context.Context, cfg *config.Config, label string, scriptPath string) error
	RunPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error
}
