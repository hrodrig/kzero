package cli

import (
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/spf13/cobra"
)

func writeDeferredFeatureWarnings(w io.Writer, cfg *config.Config) {
	for _, msg := range config.DeferredFeatureWarnings(cfg) {
		_, _ = fmt.Fprintf(w, "warning: %s\n", msg)
	}
}

func writeKubernetesTarget(w io.Writer, cfg *config.Config) error {
	if err := cluster.Print(w, cfg); err != nil {
		return fmt.Errorf("kubernetes target: %w", err)
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

func runPipelineCommand(cmd *cobra.Command, command string, cfg *config.Config, run func(*engine.Engine, *config.Config) error) error {
	return runTimed(cmd.ErrOrStderr(), command, cfg.Run.Color, func() error {
		if err := writeKubernetesTarget(cmd.OutOrStdout(), cfg); err != nil {
			return err
		}
		eng := engine.New(cfg, cmd.OutOrStdout())
		return run(eng, cfg)
	})
}

func newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Validate config and print a normalized plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("analyze config: %w", err)
			}
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			configPath := cfgFile
			if configPath == "" {
				configPath = "kzero.yaml"
			}
			return runTimed(cmd.ErrOrStderr(), "analyze", cfg.Run.Color, func() error {
				return printAnalyzePlan(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, configPath)
			})
		},
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Run the configured shutdown pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			return runPipelineCommand(cmd, "down", cfg, func(eng *engine.Engine, cfg *config.Config) error {
				return eng.RunDown(cmd.Context(), cfg)
			})
		},
	}
}

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run the configured startup pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			return runPipelineCommand(cmd, "up", cfg, func(eng *engine.Engine, cfg *config.Config) error {
				return eng.RunUp(cmd.Context(), cfg)
			})
		},
	}
}

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Run down then up",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			return runPipelineCommand(cmd, "reset", cfg, func(eng *engine.Engine, cfg *config.Config) error {
				return eng.RunReset(cmd.Context(), cfg)
			})
		},
	}
}

func newTargetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "target",
		Short: "Print the Kubernetes API target for the loaded config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return runTimed(cmd.ErrOrStderr(), "target", cfg.Run.Color, func() error {
				return cluster.Print(cmd.OutOrStdout(), cfg)
			})
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build metadata",
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "kzero %s\ncommit %s\nbuild %s\nbranch %s\n",
				Version, Commit, BuildDate, Branch)
		},
	}
}
