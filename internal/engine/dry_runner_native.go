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
// kubeconfig, deployment/statefulset steps use server-side dry-run (DryRun=All).
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
