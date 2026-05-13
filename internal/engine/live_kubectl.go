package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
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
	env := os.Environ()
	if k := strings.TrimSpace(cfg.Run.Kubeconfig); k != "" {
		env = append(env, "KUBECONFIG="+k)
	}
	return env
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
		return fmt.Errorf("live: unsupported resource kind %q for kubectl scale", step.Type)
	}

	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	replicas := scaleReplicas(phase, step)
	bin := kubectlPath(cfg)
	args := []string{
		"scale",
		fmt.Sprintf("%s/%s", step.Type, step.Name),
		"-n", step.Namespace,
		"--replicas", strconv.Itoa(replicas),
	}
	out, err := r.runProcess(opCtx, bin, args, r.envFor(cfg), ".")
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("kubectl scale %s/%s in %s: %w", step.Type, step.Name, step.Namespace, err)
	}

	if phase == PhaseUp && step.WaitForReady {
		return r.runRolloutStatus(opCtx, cfg, step)
	}
	return nil
}

func (r *LiveRunner) runRolloutStatus(ctx context.Context, cfg *config.Config, step config.PipelineStep) error {
	timeout := rolloutTimeout(cfg, step)
	bin := kubectlPath(cfg)
	args := []string{
		"rollout", "status",
		fmt.Sprintf("%s/%s", step.Type, step.Name),
		"-n", step.Namespace,
		"--timeout", timeout.String(),
	}
	out, err := r.runProcess(ctx, bin, args, r.envFor(cfg), ".")
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("kubectl rollout status %s/%s in %s: %w", step.Type, step.Name, step.Namespace, err)
	}
	return nil
}

func (r *LiveRunner) runReleaseScript(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	ws := strings.TrimSpace(cfg.Helm.Workspace)
	if ws == "" {
		return fmt.Errorf("helm.workspace is empty (required for release step %s)", step.Ref)
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
	out, err := r.runProcess(opCtx, "/bin/sh", []string{script, string(phase)}, env, ws)
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("release script %s: %w", script, err)
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
