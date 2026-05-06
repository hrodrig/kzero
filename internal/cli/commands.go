package cli

import (
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/spf13/cobra"
)

func newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Validate config and print a normalized plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("analyze config: %w", err)
			}

			configPath := cfgFile
			if configPath == "" {
				configPath = "kzero.yaml"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", configPath)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schema: %s\n", cfg.SchemaVersion)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pipeline steps: down=%d up=%d\n", len(cfg.Pipelines.Down), len(cfg.Pipelines.Up))
			return nil
		},
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Run the configured shutdown pipeline (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "down: not implemented yet")
			return nil
		},
	}
}

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run the configured startup pipeline (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "up: not implemented yet")
			return nil
		},
	}
}

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Run down then up (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "reset: not implemented yet")
			return nil
		},
	}
}
