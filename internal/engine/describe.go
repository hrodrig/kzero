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
	extras := append(stepTypePlanExtras(cfg, step, helmWorkspace, phase), stepOptionPlanExtras(step)...)
	if len(extras) == 0 {
		return base
	}
	return base + " (" + strings.Join(extras, ", ") + ")"
}

func stepTypePlanExtras(cfg *config.Config, step config.PipelineStep, helmWorkspace, phase string) []string {
	switch step.Type {
	case "release":
		return releasePlanExtras(cfg, step, helmWorkspace, phase)
	case "pvc":
		return []string{"delete pvc (background propagation, ignore-not-found)"}
	case "exec":
		return []string{executor.FormatExecPlan(step)}
	default:
		return nil
	}
}

func releasePlanExtras(cfg *config.Config, step config.PipelineStep, helmWorkspace, phase string) []string {
	if phase == string(PhaseDown) {
		if executor.WantHelmSDK(cfg) {
			return []string{"helm sdk uninstall --wait --ignore-not-found"}
		}
		return []string{"helm uninstall --wait --ignore-not-found"}
	}
	if strings.TrimSpace(helmWorkspace) == "" {
		return nil
	}
	if executor.WantHelmSDK(cfg) {
		if spec, err := executor.ResolveChartSpec(cfg, step); err == nil {
			return []string{executor.FormatChartPlan(spec)}
		}
		if strings.TrimSpace(step.Chart) != "" {
			return []string{executor.FormatChartPlan(executor.ChartSpec{Chart: step.Chart, Version: step.Version, Wait: true})}
		}
		manifest := filepath.Join(strings.TrimSpace(helmWorkspace), step.Name+".yaml")
		return []string{fmt.Sprintf("helm upgrade --install (sdk, manifest: %s)", manifest)}
	}
	script, err := executor.ResolveReleaseScriptIn(cfg, step, helmWorkspace)
	if err != nil {
		return nil
	}
	return []string{fmt.Sprintf("script: %s", script)}
}

func stepOptionPlanExtras(step config.PipelineStep) []string {
	var extras []string
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
	return extras
}
