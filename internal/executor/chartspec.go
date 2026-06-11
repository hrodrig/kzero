package executor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"gopkg.in/yaml.v3"
)

// ChartSpec describes a Helm upgrade --install for a release step (SDK path).
type ChartSpec struct {
	Chart           string
	Version         string
	ValuesFiles     []string
	CreateNamespace bool
	Wait            bool
	Timeout         time.Duration
}

type chartSpecFile struct {
	Chart           string   `yaml:"chart"`
	Version         string   `yaml:"version"`
	ValuesFiles     []string `yaml:"values_files"`
	CreateNamespace *bool    `yaml:"create_namespace"`
	Wait            *bool    `yaml:"wait"`
	Timeout         string   `yaml:"timeout"`
}

// ResolveChartSpec loads <helm.workspace>/<release>.yaml and merges optional step overrides.
func ResolveChartSpec(cfg *config.Config, step config.PipelineStep) (ChartSpec, error) {
	ws := strings.TrimSpace(cfg.Helm.Workspace)
	if ws == "" {
		return ChartSpec{}, errors.New("helm.workspace is empty")
	}
	if step.Type != "release" || step.Name == "" {
		return ChartSpec{}, fmt.Errorf("chart spec requires release step, got %q", step.Ref)
	}

	spec := ChartSpec{
		Wait:    true,
		Timeout: defaultChartTimeout(cfg, step),
	}

	manifest := filepath.Join(ws, step.Name+".yaml")
	if data, err := os.ReadFile(manifest); err == nil {
		var raw chartSpecFile
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return ChartSpec{}, fmt.Errorf("parse chart manifest %s: %w", manifest, err)
		}
		if err := applyChartFile(&spec, raw, ws); err != nil {
			return ChartSpec{}, fmt.Errorf("chart manifest %s: %w", manifest, err)
		}
	} else if !os.IsNotExist(err) {
		return ChartSpec{}, fmt.Errorf("read chart manifest %s: %w", manifest, err)
	}

	mergeStepChartOverrides(&spec, step, ws)

	if strings.TrimSpace(spec.Chart) == "" {
		return ChartSpec{}, fmt.Errorf("release %s: chart is required (set helm.workspace/%s.yaml or step chart:)", step.Ref, step.Name)
	}
	return spec, nil
}

func defaultChartTimeout(cfg *config.Config, step config.PipelineStep) time.Duration {
	if step.Timeout > 0 {
		return step.Timeout
	}
	if cfg != nil && cfg.Run.OperationTimeout > 0 {
		return cfg.Run.OperationTimeout
	}
	return 5 * time.Minute
}

func applyChartFile(spec *ChartSpec, raw chartSpecFile, workspace string) error {
	spec.Chart = strings.TrimSpace(raw.Chart)
	spec.Version = strings.TrimSpace(raw.Version)
	if len(raw.ValuesFiles) > 0 {
		spec.ValuesFiles = resolveValuesFiles(workspace, raw.ValuesFiles)
	}
	if raw.CreateNamespace != nil {
		spec.CreateNamespace = *raw.CreateNamespace
	}
	if raw.Wait != nil {
		spec.Wait = *raw.Wait
	}
	if strings.TrimSpace(raw.Timeout) != "" {
		d, err := time.ParseDuration(strings.TrimSpace(raw.Timeout))
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		spec.Timeout = d
	}
	return nil
}

func mergeStepChartOverrides(spec *ChartSpec, step config.PipelineStep, workspace string) {
	if c := strings.TrimSpace(step.Chart); c != "" {
		spec.Chart = c
	}
	if v := strings.TrimSpace(step.Version); v != "" {
		spec.Version = v
	}
	if len(step.ValuesFiles) > 0 {
		spec.ValuesFiles = resolveValuesFiles(workspace, step.ValuesFiles)
	}
	if step.CreateNamespace != nil {
		spec.CreateNamespace = *step.CreateNamespace
	}
	if step.Timeout > 0 {
		spec.Timeout = step.Timeout
	}
}

func resolveValuesFiles(workspace string, files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if filepath.IsAbs(f) {
			out = append(out, f)
			continue
		}
		out = append(out, filepath.Join(workspace, f))
	}
	return out
}

// FormatChartPlan returns a short analyze/dry-run label for SDK upgrade --install.
func FormatChartPlan(spec ChartSpec) string {
	line := "helm upgrade --install (sdk)"
	if spec.Chart != "" {
		line += ", chart=" + spec.Chart
	}
	if spec.Version != "" {
		line += ", version=" + spec.Version
	}
	if spec.Wait {
		line += ", wait"
	}
	if spec.CreateNamespace {
		line += ", create-namespace"
	}
	if spec.Timeout > 0 {
		line += ", timeout=" + spec.Timeout.String()
	}
	return line
}
