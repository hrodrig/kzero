package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
	"github.com/hrodrig/kzero/internal/executor"
)

// DryRunner logs planned invocations without executing scripts or mutating the cluster.
// When nativeWL is set (native/auto execution + kubeconfig), deployment/statefulset
// steps are validated with server-side dry-run (DryRun=All) instead of plan-only text.
type DryRunner struct {
	Out      io.Writer
	nativeWL executor.Workload
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
	_, _ = fmt.Fprintf(r.Out, "[dry-run] %shook %s: %s\n", correlation.LogPrefix(cfg), label, scriptPath)
	return nil
}

// RunPipelineStep implements Runner.
func (r *DryRunner) RunPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	if err := r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "pre"), step.PreStep); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("step %s[%d]: %w", phase, index, ctx.Err())
	default:
	}
	if step.Type == "deployment" || step.Type == "statefulset" {
		if err := r.runNativeDryRunScale(ctx, cfg, phase, index, step); err != nil {
			return err
		}
		if r.nativeWL != nil {
			return r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "post"), step.PostStep)
		}
	}

	if step.Type == "release" && phase == PhaseDown {
		if _, err := fmt.Fprintf(r.Out, "[dry-run] %shelm uninstall %s/%s (--wait --ignore-not-found)\n",
			correlation.LogPrefix(cfg), step.Namespace, step.Name); err != nil {
			return err
		}
		return r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "post"), step.PostStep)
	}

	desc := DescribeStep(step)
	if _, err := fmt.Fprintf(r.Out, "[dry-run] %spipeline %s step %d: %s\n", correlation.LogPrefix(cfg), phase, index, desc); err != nil {
		return err
	}
	return r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "post"), step.PostStep)
}
