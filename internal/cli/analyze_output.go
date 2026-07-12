package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/preflight"
	"github.com/hrodrig/kzero/internal/validate"
)

func printAnalyzePlan(w, errW io.Writer, cfg *config.Config, configPath string) error {
	if configPath == "" {
		configPath = "kzero.yaml"
	}
	if err := printAnalyzeHeader(w, cfg, configPath); err != nil {
		return err
	}
	if err := cluster.Print(w, cfg); err != nil {
		return fmt.Errorf("kubernetes target: %w", err)
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	printPhaseHooks(w, cfg)
	if err := printAnalyzePipelines(w, cfg); err != nil {
		return err
	}
	if err := printDeferredSummary(w, cfg); err != nil {
		return err
	}
	if err := validate.PrintClusterValidation(w, errW, cfg, validate.ClientFactoryDefault()); err != nil {
		return err
	}
	if w := preflight.AnalyzeWarning(context.Background(), cfg, validate.ClientFactoryDefault()); w != "" {
		_ = log.WriteLine(errW, log.LevelWarn, "warning: "+w)
	}
	return nil
}

func printAnalyzeHeader(w io.Writer, cfg *config.Config, configPath string) error {
	lines := []string{
		fmt.Sprintf("Config: %s", configPath),
		fmt.Sprintf("Schema: %s", cfg.SchemaVersion),
		fmt.Sprintf("Run mode: %s", cfg.Run.Mode),
		fmt.Sprintf("Run color: %s", cfg.Run.Color),
		fmt.Sprintf("Run execution: %s", cfg.Run.Execution),
	}
	if cfg.Client.ID != "" {
		lines = append(lines, "Client id: "+cfg.Client.ID)
	}
	if cfg.Run.Timeout > 0 {
		lines = append(lines, "Run timeout: "+cfg.Run.Timeout.String())
	}
	if cfg.Retry.Attempts > 0 {
		retryLine := fmt.Sprintf("Retry: attempts=%d", cfg.Retry.Attempts)
		if cfg.Retry.Delay > 0 {
			retryLine += ", delay=" + cfg.Retry.Delay.String()
		}
		if cfg.Run.Mode == "live" && cfg.Retry.Attempts > 1 {
			retryLine += " (live: exponential backoff on transient errors)"
		}
		lines = append(lines, retryLine)
	}
	if ws := strings.TrimSpace(cfg.Helm.Workspace); ws != "" {
		lines = append(lines, "Helm workspace: "+ws)
	}
	for _, line := range lines {
		level := log.LevelDebug
		switch {
		case strings.HasPrefix(line, "Client id:"),
			strings.HasPrefix(line, "Config:"),
			strings.HasPrefix(line, "Run mode:"):
			level = log.LevelInfo
		}
		if err := log.WriteLine(w, level, line); err != nil {
			return err
		}
	}
	return nil
}

func printAnalyzePipelines(w io.Writer, cfg *config.Config) error {
	if err := log.WriteLine(w, log.LevelInfo, fmt.Sprintf("Pipeline steps: down=%d up=%d", len(cfg.Pipelines.Down), len(cfg.Pipelines.Up))); err != nil {
		return err
	}
	helmWS := cfg.Helm.Workspace
	if err := printPipelinePhase(w, cfg, "down", cfg.Pipelines.Down, helmWS); err != nil {
		return err
	}
	return printPipelinePhase(w, cfg, "up", cfg.Pipelines.Up, helmWS)
}

func printDeferredSummary(w io.Writer, cfg *config.Config) error {
	warnings := config.DeferredFeatureWarnings(cfg)
	if len(warnings) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if err := log.WriteLine(w, log.LevelWarn, "Deferred (accepted by schema; not implemented by v1 engine):"); err != nil {
		return err
	}
	for _, msg := range warnings {
		if err := log.WriteLine(w, log.LevelWarn, "  - "+msg); err != nil {
			return err
		}
	}
	return nil
}

func printPhaseHooks(w io.Writer, cfg *config.Config) {
	type hookLine struct {
		label string
		path  string
	}
	for _, h := range []hookLine{
		{"pre-down", cfg.Hooks.PreDown},
		{"post-down", cfg.Hooks.PostDown},
		{"pre-up", cfg.Hooks.PreUp},
		{"post-up", cfg.Hooks.PostUp},
		{"on-error", cfg.Hooks.OnError},
	} {
		if strings.TrimSpace(h.path) == "" {
			continue
		}
		_ = log.WriteLine(w, log.LevelInfo, fmt.Sprintf("Hook %s: %s", h.label, h.path))
	}
}

func printPipelinePhase(w io.Writer, cfg *config.Config, phase string, steps []config.PipelineStep, helmWorkspace string) error {
	if len(steps) == 0 {
		return nil
	}
	if err := log.WriteLine(w, log.LevelInfo, fmt.Sprintf("[%s]", phase)); err != nil {
		return err
	}
	for i, step := range steps {
		line := engine.FormatStepPlanLine(cfg, step, helmWorkspace, phase)
		if err := log.WriteLine(w, log.LevelInfo, fmt.Sprintf("  %d: %s", i, line)); err != nil {
			return err
		}
	}
	return nil
}
