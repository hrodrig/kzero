package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
)

// DescribeStep returns a short normalized label for a pipeline step (ref or custom path).
func DescribeStep(step config.PipelineStep) string {
	if step.Custom != "" {
		return "custom: " + step.Custom
	}
	if step.Ref != "" {
		return step.Ref
	}
	return step.Type + "/" + step.Namespace + "/" + step.Name
}

// FormatStepPlanLine formats a step for analyze output, including optional metadata.
func FormatStepPlanLine(step config.PipelineStep, helmWorkspace string) string {
	base := DescribeStep(step)
	var extras []string

	if step.Type == "release" && strings.TrimSpace(helmWorkspace) != "" {
		script := filepath.Join(strings.TrimSpace(helmWorkspace), step.Name+".sh")
		extras = append(extras, fmt.Sprintf("script: %s", script))
	}

	if step.PreStep != "" {
		extras = append(extras, "pre: "+step.PreStep)
	}
	if step.PostStep != "" {
		extras = append(extras, "post: "+step.PostStep)
	}
	if step.Replicas != nil {
		extras = append(extras, fmt.Sprintf("replicas: %d", *step.Replicas))
	}
	if step.WaitForReady {
		extras = append(extras, "wait_for_ready: true")
	}
	if step.Timeout > 0 {
		extras = append(extras, "timeout: "+step.Timeout.String())
	}

	if len(extras) == 0 {
		return base
	}
	return base + " (" + strings.Join(extras, ", ") + ")"
}
