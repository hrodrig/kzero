package engine

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/redact"
	"github.com/hrodrig/kzero/internal/subprocess"
)

// LiveExec runs argv0 with args; env is the full environment; dir is the working directory.
// Used by tests to stub process execution. If nil on LiveRunner, os/exec is used.
type LiveExec func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error)

var scalableKinds = map[string]struct{}{
	"deployment":  {},
	"statefulset": {},
}

func kubectlPath(cfg *config.Config) string {
	if p := strings.TrimSpace(cfg.Command.Kubectl); p != "" {
		return p
	}
	return "kubectl"
}

func (r *LiveRunner) envFor(cfg *config.Config) []string {
	return subprocess.Env(cfg)
}

func (r *LiveRunner) runProcess(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
	if r.Exec != nil {
		return r.Exec(ctx, argv0, args, env, dir)
	}
	return defaultExecCombined(ctx, argv0, args, env, dir)
}

func (r *LiveRunner) writeOutput(out []byte) {
	if len(out) > 0 && r.Log != nil {
		_, _ = r.Log.Writer().Write([]byte(redact.String(string(out))))
	}
}

func (r *LiveRunner) runScaledWorkload(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	if _, ok := scalableKinds[step.Type]; !ok {
		return fmt.Errorf("live: unsupported resource kind %q for scale", step.Type)
	}

	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	wl, err := r.workloadFor(cfg)
	if err != nil {
		return err
	}

	replicas := int32(scaleReplicas(phase, step))
	r.logLive("scale %s -> %d replicas", step.Ref, replicas)
	if err := wl.Scale(opCtx, step.Type, step.Namespace, step.Name, replicas); err != nil {
		return err
	}
	if phase == PhaseUp && step.WaitForReady {
		timeout := rolloutTimeout(cfg, step)
		r.logLive("wait rollout %s (timeout %s)", step.Ref, timeout)
		return wl.WaitRollout(opCtx, step.Type, step.Namespace, step.Name, timeout)
	}
	return nil
}

func (r *LiveRunner) workloadFor(cfg *config.Config) (executor.Workload, error) {
	if r.Workload != nil {
		return r.Workload, nil
	}
	key := cfg.Run.Kubeconfig + "|" + cfg.Run.Execution + "|" + cfg.Command.Kubectl
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedWL != nil && r.cachedWLKey == key {
		return r.cachedWL, nil
	}
	var rawOut io.Writer
	if r.Log != nil {
		rawOut = r.Log.Writer()
	}
	wl, err := executor.NewWorkload(cfg, executor.Deps{
		Run:      r.runProcess,
		WriteOut: r.writeOutput,
		Out:      rawOut,
	})
	if err != nil {
		return nil, err
	}
	r.cachedWL = wl
	r.cachedWLKey = key
	return wl, nil
}

func (r *LiveRunner) runReleaseScript(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	helm, err := r.helmFor(cfg)
	if err != nil {
		return err
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	if phase == PhaseDown {
		if helm.UsesSDK() {
			r.logLive("helm sdk uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
		} else {
			r.logLive("helm uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
		}
		return helm.Uninstall(opCtx, step)
	}

	if helm.UsesSDK() {
		spec, err := executor.ResolveChartSpec(cfg, step)
		if err != nil {
			return err
		}
		r.logLive("helm sdk upgrade --install %s/%s (%s)", step.Namespace, step.Name, executor.FormatChartPlan(spec))
		return helm.UpgradeInstall(opCtx, step)
	}

	ws := cfg.Helm.Workspace
	if ws == "" {
		return fmt.Errorf("helm.workspace is empty (required for release step %s on up)", step.Ref)
	}
	r.logLive("release script %s/%s.sh (%s)", ws, step.Name, phase)
	return helm.UpgradeInstall(opCtx, step)
}

func (r *LiveRunner) helmFor(cfg *config.Config) (executor.HelmReleases, error) {
	if r.Helm != nil {
		return r.Helm, nil
	}
	key := cfg.Run.Kubeconfig + "|" + cfg.Run.Execution + "|" + cfg.Helm.Workspace + "|" + cfg.Command.Helm
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedHelm != nil && r.cachedHelmKey == key {
		return r.cachedHelm, nil
	}
	var writeOut func([]byte)
	if r.Log != nil {
		writeOut = r.writeOutput
	}
	helm, err := executor.NewHelmReleases(cfg, executor.HelmDeps{
		Cfg:      cfg,
		Run:      r.runProcess,
		WriteOut: writeOut,
	})
	if err != nil {
		return nil, fmt.Errorf("helm releases: %w", err)
	}
	r.cachedHelm = helm
	r.cachedHelmKey = key
	return helm, nil
}

func withOpTimeout(ctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	if cfg.Run.OperationTimeout > 0 {
		return context.WithTimeout(ctx, cfg.Run.OperationTimeout)
	}
	return ctx, func() {}
}

func scaleReplicas(phase Phase, step config.PipelineStep) int {
	if phase == PhaseDown {
		return 0
	}
	if step.Replicas != nil {
		return *step.Replicas
	}
	return 1
}

func rolloutTimeout(cfg *config.Config, step config.PipelineStep) time.Duration {
	if step.Timeout > 0 {
		return step.Timeout
	}
	if cfg.Run.OperationTimeout > 0 {
		return cfg.Run.OperationTimeout
	}
	return 5 * time.Minute
}
