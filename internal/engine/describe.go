package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
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
// phase is the pipeline phase name ("down" or "up") for release step hints.
func FormatStepPlanLine(cfg *config.Config, step config.PipelineStep, helmWorkspace string, phase string) string {
	base := DescribeStep(step)
	var extras []string

	if step.Type == "release" {
		if phase == string(PhaseDown) {
			if executor.WantHelmSDK(cfg) {
				extras = append(extras, "helm sdk uninstall --wait --ignore-not-found")
			} else {
				extras = append(extras, "helm uninstall --wait --ignore-not-found")
			}
		} else if strings.TrimSpace(helmWorkspace) != "" {
			if executor.WantHelmSDK(cfg) {
				if spec, err := executor.ResolveChartSpec(cfg, step); err == nil {
					extras = append(extras, executor.FormatChartPlan(spec))
				} else if strings.TrimSpace(step.Chart) != "" {
					extras = append(extras, executor.FormatChartPlan(executor.ChartSpec{Chart: step.Chart, Version: step.Version, Wait: true}))
				} else {
					manifest := filepath.Join(strings.TrimSpace(helmWorkspace), step.Name+".yaml")
					extras = append(extras, fmt.Sprintf("helm upgrade --install (sdk, manifest: %s)", manifest))
				}
			} else {
				script := filepath.Join(strings.TrimSpace(helmWorkspace), step.Name+".sh")
				extras = append(extras, fmt.Sprintf("script: %s", script))
			}
		}
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
