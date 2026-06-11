package executor

import (
	"context"

	"github.com/hrodrig/kzero/internal/config"
)

// HelmReleases performs live release down/up (shell helm or Helm SDK).
type HelmReleases interface {
	Uninstall(ctx context.Context, step config.PipelineStep) error
	UpgradeInstall(ctx context.Context, step config.PipelineStep) error
	UsesSDK() bool
}

// HelmDeps groups subprocess dependencies for the shell Helm backend.
type HelmDeps struct {
	Cfg      *config.Config
	Run      RunFunc
	WriteOut func([]byte)
}

// NewHelmReleases picks shell or SDK from run.execution (native/auto → SDK).
func NewHelmReleases(cfg *config.Config, deps HelmDeps) (HelmReleases, error) {
	if WantHelmSDK(cfg) {
		return NewSDKHelm(cfg)
	}
	return NewShellHelm(deps), nil
}

// HelmPath returns the helm binary configured on cfg.
func HelmPath(cfg *config.Config) string {
	if cfg != nil {
		if p := cfg.Command.Helm; p != "" {
			return p
		}
	}
	return "helm"
}
