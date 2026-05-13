package engine

import "fmt"

// Phase identifies a pipeline direction for logging and runner dispatch.
type Phase string

const (
	PhaseDown Phase = "down"
	PhaseUp   Phase = "up"
)

// pipelineStepHookLabel is the hook label for per-step pre/post scripts (dry-run, logs, tests).
func pipelineStepHookLabel(phase Phase, index int, hookKind string) string {
	return fmt.Sprintf("pipeline-%s-%d-%s", phase, index, hookKind)
}
