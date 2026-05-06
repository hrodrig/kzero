package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/config"
)

// LiveRunner executes hooks, custom scripts, kubectl scale/rollout for workloads,
// and release helper scripts under helm.workspace.
type LiveRunner struct {
	Out  io.Writer
	Exec LiveExec
}

// RunHook implements Runner.
func (r *LiveRunner) RunHook(ctx context.Context, cfg *config.Config, label, scriptPath string) error {
	if scriptPath == "" {
		return nil
	}
	return r.execScript(ctx, cfg, label, scriptPath)
}

// RunPipelineStep implements Runner.
func (r *LiveRunner) RunPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	if step.Custom != "" {
		return r.execScript(ctx, cfg, fmt.Sprintf("pipeline-%s-%d", phase, index), step.Custom)
	}
	if step.Ref == "" {
		return fmt.Errorf("live: empty pipeline step at index %d", index)
	}

	switch step.Type {
	case "deployment", "statefulset", "daemonset":
		return r.runScaledWorkload(ctx, cfg, phase, step)
	case "release":
		return r.runReleaseScript(ctx, cfg, phase, step)
	default:
		return fmt.Errorf("live: unsupported pipeline resource type %q", step.Type)
	}
}

func (r *LiveRunner) execScript(ctx context.Context, cfg *config.Config, label, scriptPath string) error {
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	out, err := r.runProcess(opCtx, "/bin/sh", []string{scriptPath}, r.envFor(cfg), ".")
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}
