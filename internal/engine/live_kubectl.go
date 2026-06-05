package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
	"github.com/hrodrig/kzero/internal/executor"
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

func helmPath(cfg *config.Config) string {
	if p := strings.TrimSpace(cfg.Command.Helm); p != "" {
		return p
	}
	return "helm"
}

func (r *LiveRunner) envFor(cfg *config.Config) []string {
	env := os.Environ()
	if k := strings.TrimSpace(cfg.Run.Kubeconfig); k != "" {
		env = append(env, "KUBECONFIG="+k)
	}
	return correlation.AppendEnv(cfg, env)
}

func (r *LiveRunner) runProcess(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
	if r.Exec != nil {
		return r.Exec(ctx, argv0, args, env, dir)
	}
	return defaultExecCombined(ctx, argv0, args, env, dir)
}

func (r *LiveRunner) writeOutput(out []byte) {
	if len(out) > 0 && r.Out != nil {
		_, _ = r.Out.Write(out)
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
	r.logLive(cfg, "scale %s -> %d replicas", step.Ref, replicas)
	if err := wl.Scale(opCtx, step.Type, step.Namespace, step.Name, replicas); err != nil {
		return err
	}
	if phase == PhaseUp && step.WaitForReady {
		timeout := rolloutTimeout(cfg, step)
		r.logLive(cfg, "wait rollout %s (timeout %s)", step.Ref, timeout)
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
	wl, err := executor.NewWorkload(cfg, executor.Deps{
		Run:      r.runProcess,
		WriteOut: r.writeOutput,
		Out:      r.Out,
	})
	if err != nil {
		return nil, err
	}
	r.cachedWL = wl
	r.cachedWLKey = key
	return wl, nil
}

func (r *LiveRunner) runReleaseScript(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	if phase == PhaseDown {
		return r.runHelmUninstall(ctx, cfg, step)
	}
	return r.runHelmInstallScript(ctx, cfg, phase, step)
}

func (r *LiveRunner) runHelmUninstall(ctx context.Context, cfg *config.Config, step config.PipelineStep) error {
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	helmBin := helmPath(cfg)
	args := []string{
		"uninstall", step.Name,
		"-n", step.Namespace,
		"--wait",
		"--ignore-not-found",
	}
	env := r.envFor(cfg)
	env = append(env,
		"KZERO_PHASE=down",
		"KZERO_RELEASE_NAME="+step.Name,
		"KZERO_RELEASE_NAMESPACE="+step.Namespace,
	)
	r.logLive(cfg, "helm uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
	out, err := r.runProcess(opCtx, helmBin, args, env, ".")
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("helm uninstall %s/%s: %w", step.Namespace, step.Name, executor.WrapSubprocess(helmBin, args, out, err))
	}
	return nil
}

func (r *LiveRunner) runHelmInstallScript(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	ws := strings.TrimSpace(cfg.Helm.Workspace)
	if ws == "" {
		return fmt.Errorf("helm.workspace is empty (required for release step %s on up)", step.Ref)
	}
	script := filepath.Join(ws, step.Name+".sh")
	st, err := os.Stat(script)
	if err != nil {
		return fmt.Errorf("release script %s: %w", script, err)
	}
	if st.IsDir() {
		return fmt.Errorf("release script path is a directory: %s", script)
	}

	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	env := r.envFor(cfg)
	env = append(env,
		"KZERO_PHASE="+string(phase),
		"KZERO_RELEASE_NAME="+step.Name,
		"KZERO_RELEASE_NAMESPACE="+step.Namespace,
	)
	r.logLive(cfg, "release script %s (%s)", script, phase)
	out, err := r.runProcess(opCtx, "/bin/sh", []string{script, string(phase)}, env, ws)
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("release script %s: %w", script, executor.WrapSubprocess("/bin/sh", []string{script, string(phase)}, out, err))
	}
	return nil
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
