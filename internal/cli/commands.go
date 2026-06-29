package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/notify"
	"github.com/hrodrig/kzero/internal/probe"
	"github.com/hrodrig/kzero/internal/redact"
	"github.com/spf13/cobra"
)

func writeDeferredFeatureWarnings(w io.Writer, cfg *config.Config) {
	for _, msg := range config.DeferredFeatureWarnings(cfg) {
		_ = log.WriteLine(w, log.LevelWarn, "warning: "+msg)
	}
}

func applyCLIRunOverrides(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if noEnvPassthrough {
		cfg.Run.NoEnvPassthrough = true
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
	format, err := resolvedLogFormat()
	if err != nil {
		return err
	}
	if err := applyLogLevel(); err != nil {
		return err
	}
	return runTimed(cmd.ErrOrStderr(), command, cfg.Run.Color, format, func() error {
		started := time.Now()
		if err := writeKubernetesTarget(cmd.OutOrStdout(), cfg); err != nil {
			return err
		}
		ctx := cmd.Context()
		if notify.AnyEnabled(cfg) {
			meta := notify.MetaFromConfig(cfg, command, started, 0)
			if err := notify.Dispatch(ctx, cfg, notify.EventStart, meta, nil); err != nil {
				_ = log.WriteLine(cmd.ErrOrStderr(), log.LevelError,
					"notify dispatch failed ("+notify.EventStart+"): "+redact.String(err.Error()))
			}
		}
		emit := log.New(cmd.OutOrStdout(), format)
		emit.SetCommand(command)
		eng := engine.New(cfg, emit)
		eng.Command = command
		eng.Started = started
		if probe.ShouldGate(cfg, command) {
			if err := runInfraProbeGate(cmd, cfg, eng, command); err != nil {
				return err
			}
		}
		if err := run(eng, cfg); err != nil {
			return err
		}
		if shouldAutoVerify(cfg, command) {
			if err := runVerify(cmd, cfg, format, false); err != nil {
				return fmt.Errorf("post-up verify: %w", err)
			}
		}
		if notify.AnyEnabled(cfg) {
			meta := notify.MetaFromConfig(cfg, command, started, time.Since(started))
			if err := notify.Dispatch(ctx, cfg, notify.EventSuccess, meta, nil); err != nil {
				_ = log.WriteLine(cmd.ErrOrStderr(), log.LevelError,
					"notify dispatch failed ("+notify.EventSuccess+"): "+redact.String(err.Error()))
			}
		}
		return nil
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
			format, err := resolvedLogFormat()
			if err != nil {
				return err
			}
			if err := applyLogLevel(); err != nil {
				return err
			}
			return runTimed(cmd.ErrOrStderr(), "analyze", cfg.Run.Color, format, func() error {
				return printAnalyzePlan(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg, configPath)
			})
		},
	}
}

type pipelineRunFunc func(ctx context.Context, eng *engine.Engine, cfg *config.Config) error

func buildPipelineCmd(use, short, label string, run pipelineRunFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			applyCLIRunOverrides(cfg)
			writeDeferredFeatureWarnings(cmd.ErrOrStderr(), cfg)
			return runPipelineCommand(cmd, label, cfg, func(eng *engine.Engine, cfg *config.Config) error {
				return run(cmd.Context(), eng, cfg)
			})
		},
	}
}

func newDownCmd() *cobra.Command {
	return buildPipelineCmd("down", "Run the configured shutdown pipeline", "down",
		func(ctx context.Context, eng *engine.Engine, cfg *config.Config) error {
			return eng.RunDown(ctx, cfg)
		})
}

func newUpCmd() *cobra.Command {
	return buildPipelineCmd("up", "Run the configured startup pipeline", "up",
		func(ctx context.Context, eng *engine.Engine, cfg *config.Config) error {
			return eng.RunUp(ctx, cfg)
		})
}

func newResetCmd() *cobra.Command {
	return buildPipelineCmd("reset", "Run down then up", "reset",
		func(ctx context.Context, eng *engine.Engine, cfg *config.Config) error {
			return eng.RunReset(ctx, cfg)
		})
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
			format, err := resolvedLogFormat()
			if err != nil {
				return err
			}
			if err := applyLogLevel(); err != nil {
				return err
			}
			return runTimed(cmd.ErrOrStderr(), "target", cfg.Run.Color, format, func() error {
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
