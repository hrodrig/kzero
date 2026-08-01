package engine

import (
	"context"
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/log"
)

// DryRunner logs planned invocations without executing scripts or mutating the cluster.
// When nativeWL is set (native/auto execution + kubeconfig), deployment/statefulset
// steps are validated with server-side dry-run (DryRun=All) instead of plan-only text.
// CronJob/Job clients (when set) similarly validate suspend/create with DryRun=All.
type DryRunner struct {
	Log      *log.Emitter
	nativeWL executor.Workload
	cronJob  executor.CronJobSuspender
	job      executor.JobRunner
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
	if r.Log != nil {
		r.Log.DryRun(cfg, fmt.Sprintf("hook %s: %s", label, scriptPath))
	}
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
	if err := r.runMainDryRunStep(ctx, cfg, phase, index, step); err != nil {
		return err
	}
	return r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "post"), step.PostStep)
}

func (r *DryRunner) runMainDryRunStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	switch step.Type {
	case "deployment", "statefulset":
		if err := r.runNativeDryRunScale(ctx, cfg, phase, index, step); err != nil {
			return err
		}
		if r.nativeWL != nil {
			return nil
		}
	case "pvc":
		if r.Log != nil {
			r.Log.DryRun(cfg, fmt.Sprintf("delete pvc %s/%s (background propagation, ignore-not-found)", step.Namespace, step.Name))
		}
		return nil
	case "cronjob":
		return r.runCronJobDryRun(ctx, cfg, phase, step)
	case "job":
		return r.runJobDryRun(ctx, cfg, phase, step)
	case "exec":
		if r.Log != nil {
			r.Log.DryRun(cfg, executor.FormatExecPlan(step))
		}
		return nil
	}
	if r.releaseDryRunHandled(cfg, phase, step) {
		return nil
	}
	if r.Log != nil {
		r.Log.DryRun(cfg, fmt.Sprintf("pipeline %s step %d: %s", phase, index, DescribeStep(step)))
	}
	return nil
}

func (r *DryRunner) releaseDryRunHandled(cfg *config.Config, phase Phase, step config.PipelineStep) bool {
	if step.Type != "release" {
		return false
	}
	if phase == PhaseDown {
		if r.Log != nil {
			msg := fmt.Sprintf("helm uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
			if executor.WantHelmSDK(cfg) {
				msg = fmt.Sprintf("helm sdk uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
			}
			r.Log.DryRun(cfg, msg)
		}
		return true
	}
	if phase == PhaseUp && executor.WantHelmSDK(cfg) {
		if r.Log != nil {
			if spec, err := executor.ResolveChartSpec(cfg, step); err == nil {
				r.Log.DryRun(cfg, fmt.Sprintf("helm sdk upgrade --install %s/%s (%s)", step.Namespace, step.Name, executor.FormatChartPlan(spec)))
			} else {
				r.Log.DryRun(cfg, fmt.Sprintf("helm sdk upgrade --install %s/%s", step.Namespace, step.Name))
			}
		}
		return true
	}
	return false
}
