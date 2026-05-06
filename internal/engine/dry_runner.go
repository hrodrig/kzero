package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/config"
)

// DryRunner logs planned invocations without executing scripts or mutating the cluster.
type DryRunner struct {
	Out io.Writer
}

// RunHook implements Runner.
func (r *DryRunner) RunHook(ctx context.Context, cfg *config.Config, label, scriptPath string) error {
	if scriptPath == "" {
		return nil
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("hook %s: %w", label, ctx.Err())
	default:
	}
	_, _ = fmt.Fprintf(r.Out, "[dry-run] hook %s: %s\n", label, scriptPath)
	return nil
}

// RunPipelineStep implements Runner.
func (r *DryRunner) RunPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("step %s[%d]: %w", phase, index, ctx.Err())
	default:
	}
	desc := describeStep(step)
	_, _ = fmt.Fprintf(r.Out, "[dry-run] pipeline %s step %d: %s\n", phase, index, desc)
	return nil
}

func describeStep(step config.PipelineStep) string {
	if step.Custom != "" {
		return "custom " + step.Custom
	}
	if step.Ref != "" {
		return step.Ref
	}
	return step.Type + "/" + step.Namespace + "/" + step.Name
}
