package executor

import (
	"fmt"
	"io"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
)

// Deps groups dependencies for resolving a Workload backend.
type Deps struct {
	Run      RunFunc
	WriteOut func([]byte)
	// Out receives optional auto-fallback notices (may be nil).
	Out io.Writer
}

// NewWorkload picks shell, native, or auto (native with shell fallback) per cfg.Run.Execution.
func NewWorkload(cfg *config.Config, deps Deps) (Workload, error) {
	shell := ShellFromConfig(cfg, deps.Run, deps.WriteOut)
	mode := ExecutionMode(cfg)
	switch mode {
	case ExecutionShell:
		return shell, nil
	case ExecutionNative:
		return newNativeOrError(cfg)
	case ExecutionAuto:
		native, err := newNativeOrError(cfg)
		if err == nil {
			return native, nil
		}
		if deps.Out != nil {
			_, _ = fmt.Fprintf(deps.Out, "execution auto: using shell (%v)\n", err)
		}
		return shell, nil
	default:
		return nil, fmt.Errorf("unsupported run.execution %q", mode)
	}
}

func newNativeOrError(cfg *config.Config) (Workload, error) {
	client, err := NewKubernetesClient(cfg.Run.Kubeconfig)
	if err != nil {
		return nil, err
	}
	return NewNative(client), nil
}

// KubectlPath returns the kubectl binary configured on cfg.
func KubectlPath(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Command.Kubectl) != "" {
		return cfg.Command.Kubectl
	}
	return "kubectl"
}
