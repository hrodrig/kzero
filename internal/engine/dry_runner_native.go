package engine

import (
	"context"
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/log"
)

func nativeDryRunEnabled(cfg *config.Config) bool {
	switch executor.ExecutionMode(cfg) {
	case executor.ExecutionNative, executor.ExecutionAuto:
		return true
	default:
		return false
	}
}

// NewDryRunner builds a dry-run Runner. For native/auto execution with a reachable
// kubeconfig, deployment/statefulset steps use server-side dry-run (DryRun=All);
// cronjob suspend and job create use DryRun=All when clients can be built.
func NewDryRunner(cfg *config.Config, emit *log.Emitter) Runner {
	dr := &DryRunner{Log: emit}
	if cfg == nil || !nativeDryRunEnabled(cfg) {
		return dr
	}
	wl, err := executor.NewNativeForDryRun(cfg)
	if err != nil {
		if emit != nil {
			emit.DryRun(cfg, fmt.Sprintf("server-side dry-run unavailable (%v); planning scale steps only", err))
		}
		return dr
	}
	dr.nativeWL = wl
	if cj, err := executor.NewCronJobForDryRun(cfg.Run.Kubeconfig); err == nil {
		dr.cronJob = cj
	}
	if j, err := executor.NewJobForDryRun(cfg.Run.Kubeconfig); err == nil {
		dr.job = j
	}
	return dr
}

func (r *DryRunner) runNativeDryRunScale(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	if r.nativeWL == nil {
		return nil
	}
	if _, ok := scalableKinds[step.Type]; !ok {
		return nil
	}

	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	replicas := int32(scaleReplicas(phase, step))
	if err := r.nativeWL.Scale(opCtx, step.Type, step.Namespace, step.Name, replicas); err != nil {
		return fmt.Errorf("dry-run native scale %s: %w", step.Ref, err)
	}
	if r.Log != nil {
		r.Log.DryRun(cfg, fmt.Sprintf("native scale %s -> %d replicas (server-side dry-run ok)", step.Ref, replicas))
	}
	if phase == PhaseUp && step.WaitForReady {
		timeout := rolloutTimeout(cfg, step)
		if r.Log != nil {
			r.Log.DryRun(cfg, fmt.Sprintf("native would wait for rollout %s (timeout %s)", step.Ref, timeout))
		}
	}
	return nil
}

func (r *DryRunner) runCronJobDryRun(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	suspend := phase == PhaseDown
	plan := executor.FormatCronJobPlan(suspend)
	if r.cronJob == nil {
		if r.Log != nil {
			r.Log.DryRun(cfg, fmt.Sprintf("%s %s", plan, step.Ref))
		}
		return nil
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()
	if err := r.cronJob.Suspend(opCtx, step.Namespace, step.Name, suspend); err != nil {
		return fmt.Errorf("dry-run cronjob %s: %w", step.Ref, err)
	}
	if r.Log != nil {
		r.Log.DryRun(cfg, fmt.Sprintf("%s %s (server-side dry-run ok)", plan, step.Ref))
	}
	return nil
}

func (r *DryRunner) runJobDryRun(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	if phase == PhaseDown {
		if r.Log != nil {
			r.Log.DryRun(cfg, fmt.Sprintf("delete job %s/%s (background propagation, ignore-not-found)", step.Namespace, step.Name))
		}
		return nil
	}
	plan := executor.FormatJobPlan(false, step.Manifest, step.JobWaitForComplete())
	if r.job == nil {
		if r.Log != nil {
			r.Log.DryRun(cfg, plan)
		}
		return nil
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()
	if err := r.job.CreateFromManifest(opCtx, step.Namespace, step.Name, step.Manifest); err != nil {
		return fmt.Errorf("dry-run job %s: %w", step.Ref, err)
	}
	if r.Log != nil {
		r.Log.DryRun(cfg, plan+" (server-side dry-run ok)")
		if step.JobWaitForComplete() {
			timeout := rolloutTimeout(cfg, step)
			r.Log.DryRun(cfg, fmt.Sprintf("would wait for job complete %s (timeout %s)", step.Ref, timeout))
		}
	}
	return nil
}
