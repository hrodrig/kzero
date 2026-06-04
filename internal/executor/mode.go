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

// ExecutionMode returns the effective execution backend from cfg (default shell).
func ExecutionMode(cfg *config.Config) string {
	if cfg == nil {
		return ExecutionShell
	}
	e := strings.TrimSpace(cfg.Run.Execution)
	if e == "" {
		return ExecutionShell
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
