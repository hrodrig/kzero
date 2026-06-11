package cli

import (
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/verify"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Run post-up readiness checks without mutating the cluster",
		Long: `Checks deployment/statefulset readiness from pipelines.up and optional
node Ready status. Does not scale workloads or run Helm.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			format, err := resolvedLogFormat()
			if err != nil {
				return err
			}
			if err := applyLogLevel(); err != nil {
				return err
			}
			return runTimed(cmd.ErrOrStderr(), "verify", cfg.Run.Color, format, func() error {
				return runVerify(cmd, cfg, format, true)
			})
		},
	}
}

func runVerify(cmd *cobra.Command, cfg *config.Config, format log.Format, printTarget bool) error {
	if printTarget {
		if err := writeKubernetesTarget(cmd.OutOrStdout(), cfg); err != nil {
			return err
		}
	}
	report, err := verify.Run(cmd.Context(), cfg, nil)
	if printErr := verify.Print(cmd.OutOrStdout(), format, report); printErr != nil {
		return printErr
	}
	if verify.Failed(report) {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", verify.ErrorMessage(report))
	}
	return nil
}

func shouldAutoVerify(cfg *config.Config, command string) bool {
	if cfg == nil || !cfg.Run.Verify || cfg.Run.Mode != "live" {
		return false
	}
	return command == "up" || command == "reset"
}
