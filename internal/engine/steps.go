package engine

import (
	"context"

	"github.com/hrodrig/kzero/internal/config"
)

// RunSteps executes pipeline steps without phase hooks (used by infra probe mini-pipelines).
func (e *Engine) RunSteps(ctx context.Context, cfg *config.Config, phase Phase, steps []config.PipelineStep) error {
	for i, step := range steps {
		if err := e.runPipelineStepWithRetry(ctx, cfg, phase, i, step); err != nil {
			return &PipelineError{Phase: string(phase), Index: i, Ref: step.Ref, Err: err}
		}
	}
	return nil
}
