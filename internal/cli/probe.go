package cli

import (
	"context"
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/exitcode"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/probe"
	"github.com/spf13/cobra"
)

func newProbeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "probe",
		Short: "Run infra probe mini-pipeline (up, checks, down)",
		Long: `Executes infra_probe.pipeline.up, optional checks (PVC Bound,
release_ready), then pipeline.down. Does not run the main pipelines.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return exitcode.New(exitcode.ConfigError, fmt.Errorf("load config: %w", err))
			}
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			format, err := resolvedLogFormat()
			if err != nil {
				return err
			}
			if err := applyLogLevel(); err != nil {
				return err
			}
			return runTimed(cmd.ErrOrStderr(), "probe", cfg.Run.Color, format, func() error {
				if err := writeKubernetesTarget(cmd.OutOrStdout(), cfg); err != nil {
					return err
				}
				ctx, stop := pipelineRunContext(cmd.Context())
				defer stop()
				emit := log.New(cmd.OutOrStdout(), format)
				emit.SetCommand("probe")
				eng := engine.New(cfg, emit)
				return exitcode.Ensure(exitcode.ExecutorAborted, probe.Run(ctx, cfg, eng, nil, emit))
			})
		},
	}
}

func runInfraProbeGate(cmd *cobra.Command, cfg *config.Config, eng *engine.Engine, command string, ctx context.Context) error {
	emit := eng.Log
	if emit == nil {
		emit = log.New(cmd.OutOrStdout(), log.FormatText)
	}
	return exitcode.Ensure(exitcode.ExecutorAborted, probe.RunGate(ctx, cfg, eng, nil, emit, command))
}
