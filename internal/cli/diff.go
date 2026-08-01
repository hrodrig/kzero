package cli

import (
	"fmt"
	"strings"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	kzdiff "github.com/hrodrig/kzero/internal/diff"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/exitcode"
	"github.com/hrodrig/kzero/internal/validate"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var phase string
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare pipeline desired state to the live cluster",
		Long: `Compare the desired state for pipelines.up or pipelines.down against the live cluster (read-only).

Exit 0 when every comparable step matches; exit 2 on drift or Kubernetes errors; exit 1 on config errors.

Examples:
  # After a successful up / before maintenance — expect running replicas, CronJobs resumed, Jobs present
  kzero diff --config ./kzero.yaml
  kzero diff --phase up

  # After a successful down — expect scale 0, CronJobs suspended, PVCs/Jobs/releases gone
  kzero diff --phase down

  # Typical pre-reset check: cluster should match "up" desired state
  kzero diff -c /etc/kzero/kzero.yaml --phase up && kzero reset -c /etc/kzero/kzero.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return exitcode.New(exitcode.ConfigError, fmt.Errorf("diff config: %w", err))
			}
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			p, err := parseDiffPhase(phase)
			if err != nil {
				return exitcode.New(exitcode.ConfigError, err)
			}
			format, err := resolvedLogFormat()
			if err != nil {
				return err
			}
			if err := applyLogLevel(); err != nil {
				return err
			}
			return runTimed(cmd.ErrOrStderr(), "diff", cfg.Run.Color, format, func() error {
				if err := cluster.Print(cmd.OutOrStdout(), cfg); err != nil {
					return exitcode.New(exitcode.KubernetesError, fmt.Errorf("kubernetes target: %w", err))
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
					return err
				}
				if err := kzdiff.Print(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, p, validate.ClientFactoryDefault()); err != nil {
					if strings.Contains(err.Error(), "cannot load kubeconfig") {
						return exitcode.New(exitcode.KubernetesError, err)
					}
					return exitcode.New(exitcode.KubernetesError, err)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&phase, "phase", "up", "pipeline phase to compare: up or down")
	return cmd
}

func parseDiffPhase(s string) (engine.Phase, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "", "up":
		return engine.PhaseUp, nil
	case "down":
		return engine.PhaseDown, nil
	default:
		return "", fmt.Errorf("diff --phase must be up or down (got %q)", s)
	}
}
