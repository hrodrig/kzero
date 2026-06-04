package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
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
func NewDryRunner(cfg *config.Config, out io.Writer) Runner {
	dr := &DryRunner{Out: out}
	if cfg == nil || !nativeDryRunEnabled(cfg) {
		return dr
	}
	wl, err := executor.NewNativeForDryRun(cfg)
	if err != nil {
		if out != nil {
			_, _ = fmt.Fprintf(out, "[dry-run] server-side dry-run unavailable (%v); planning scale steps only\n", err)
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
	if _, err := fmt.Fprintf(r.Out, "[dry-run] native scale %s -> %d replicas (server-side dry-run ok)\n", step.Ref, replicas); err != nil {
		return err
	}
	if phase == PhaseUp && step.WaitForReady {
		timeout := rolloutTimeout(cfg, step)
		if _, err := fmt.Fprintf(r.Out, "[dry-run] native would wait for rollout %s (timeout %s)\n", step.Ref, timeout); err != nil {
			return err
		}
	}
	return nil
}
