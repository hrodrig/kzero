package executor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
)

// ResolveReleaseScript returns the shell install script path for a release up step.
// Default: <helm.workspace>/<release-name>.sh; override with step script: (relative to workspace).
func ResolveReleaseScript(cfg *config.Config, step config.PipelineStep) (string, error) {
	return resolveReleaseScript(cfg, step, "")
}

// ResolveReleaseScriptIn is like ResolveReleaseScript but falls back to workspaceOverride
// when cfg.Helm.Workspace is empty (analyze/describe paths).
func ResolveReleaseScriptIn(cfg *config.Config, step config.PipelineStep, workspaceOverride string) (string, error) {
	return resolveReleaseScript(cfg, step, workspaceOverride)
}

func resolveReleaseScript(cfg *config.Config, step config.PipelineStep, workspaceOverride string) (string, error) {
	ws := strings.TrimSpace(cfg.Helm.Workspace)
	if ws == "" {
		ws = strings.TrimSpace(workspaceOverride)
	}
	if ws == "" {
		return "", fmt.Errorf("helm.workspace is empty (required for release step %s on up)", step.Ref)
	}
	if script := strings.TrimSpace(step.Script); script != "" {
		if filepath.IsAbs(script) {
			return script, nil
		}
		return filepath.Join(ws, script), nil
	}
	return filepath.Join(ws, step.Name+".sh"), nil
}
