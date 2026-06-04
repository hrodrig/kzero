package cli

import (
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/spf13/cobra"
)

func writeDeferredFeatureWarnings(w io.Writer, cfg *config.Config) {
	for _, msg := range config.DeferredFeatureWarnings(cfg) {
		_, _ = fmt.Fprintf(w, "warning: %s\n", msg)
	}
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
			return printAnalyzePlan(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, configPath)
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
			eng := engine.New(cfg, cmd.OutOrStdout())
			return eng.RunDown(cmd.Context(), cfg)
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
			eng := engine.New(cfg, cmd.OutOrStdout())
			return eng.RunUp(cmd.Context(), cfg)
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
			eng := engine.New(cfg, cmd.OutOrStdout())
			return eng.RunReset(cmd.Context(), cfg)
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
