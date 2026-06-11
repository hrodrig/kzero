package executor

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/subprocess"
)

// ShellHelm runs helm CLI and release .sh scripts.
type ShellHelm struct {
	deps HelmDeps
}

// NewShellHelm returns a shell-backed HelmReleases executor.
func NewShellHelm(deps HelmDeps) *ShellHelm {
	return &ShellHelm{deps: deps}
}

func (h *ShellHelm) UsesSDK() bool { return false }

func (h *ShellHelm) Uninstall(ctx context.Context, step config.PipelineStep) error {
	helmBin := HelmPath(h.deps.Cfg)
	args := []string{
		"uninstall", step.Name,
		"-n", step.Namespace,
		"--wait",
		"--ignore-not-found",
	}
	env := subprocess.Env(h.deps.Cfg)
	env = append(env,
		"KZERO_PHASE=down",
		"KZERO_RELEASE_NAME="+step.Name,
		"KZERO_RELEASE_NAMESPACE="+step.Namespace,
	)
	out, err := h.deps.Run(ctx, helmBin, args, env, ".")
	if h.deps.WriteOut != nil && len(out) > 0 {
		h.deps.WriteOut(out)
	}
	if err != nil {
		return fmt.Errorf("helm uninstall %s/%s: %w", step.Namespace, step.Name, WrapSubprocess(helmBin, args, out, err))
	}
	return nil
}

func (h *ShellHelm) UpgradeInstall(ctx context.Context, step config.PipelineStep) error {
	script, err := ResolveReleaseScript(h.deps.Cfg, step)
	if err != nil {
		return err
	}
	st, err := os.Stat(script)
	if err != nil {
		return fmt.Errorf("release script %s: %w", script, err)
	}
	if st.IsDir() {
		return fmt.Errorf("release script path is a directory: %s", script)
	}
	ws := strings.TrimSpace(h.deps.Cfg.Helm.Workspace)

	env := subprocess.Env(h.deps.Cfg)
	env = append(env,
		"KZERO_PHASE=up",
		"KZERO_RELEASE_NAME="+step.Name,
		"KZERO_RELEASE_NAMESPACE="+step.Namespace,
	)
	out, err := h.deps.Run(ctx, "/bin/sh", []string{script, "up"}, env, ws)
	if h.deps.WriteOut != nil && len(out) > 0 {
		h.deps.WriteOut(out)
	}
	if err != nil {
		return fmt.Errorf("release script %s: %w", script, WrapSubprocess("/bin/sh", []string{script, "up"}, out, err))
	}
	return nil
}
