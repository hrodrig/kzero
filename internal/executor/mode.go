package executor

import (
	"strings"

	"github.com/hrodrig/kzero/internal/config"
)

// Execution modes for run.execution.
const (
	ExecutionShell  = "shell"
	ExecutionNative = "native"
	ExecutionAuto   = "auto"
)

// ExecutionMode returns the effective execution backend from cfg.
// Empty or omitted execution matches load-time default (#32): native.
// nil cfg keeps shell (no config — safer for misuse paths that expect kubectl helpers).
func ExecutionMode(cfg *config.Config) string {
	if cfg == nil {
		return ExecutionShell
	}
	e := strings.TrimSpace(cfg.Run.Execution)
	if e == "" {
		return ExecutionNative
	}
	return e
}

// WantNative reports whether workload steps should use client-go instead of kubectl.
func WantNative(cfg *config.Config) bool {
	switch ExecutionMode(cfg) {
	case ExecutionNative:
		return true
	case ExecutionAuto:
		return true
	default:
		return false
	}
}

// WantHelmSDK reports whether release steps should use helm.sh/helm/v3 instead of shell helm/.sh.
func WantHelmSDK(cfg *config.Config) bool {
	return WantNative(cfg)
}
