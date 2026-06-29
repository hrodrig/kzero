package cli

import (
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/notify"
	"github.com/spf13/cobra"
)

func newNotifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify",
		Short: "Outbound notification helpers",
	}
	cmd.AddCommand(newNotifyTestCmd())
	return cmd
}

func newNotifyTestCmd() *cobra.Command {
	var event string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Send a test notification without running a pipeline",
		Long: `Loads notify.* from config and POSTs to every enabled channel.
Does not contact the Kubernetes API or run down/up/reset steps.

Default event is notify.test. Use --event to preview pipeline.start,
pipeline.success, or pipeline.error payload formatting.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if err := notify.ValidateTestEvent(event); err != nil {
				return err
			}
			if !notify.AnyEnabled(cfg) {
				return fmt.Errorf("notify test: no notify channel enabled in config")
			}
			format, err := resolvedLogFormat()
			if err != nil {
				return err
			}
			if err := applyLogLevel(); err != nil {
				return err
			}
			return runTimed(cmd.ErrOrStderr(), "notify test", cfg.Run.Color, format, func() error {
				if err := notify.DispatchTest(cmd.Context(), cfg, event, nil); err != nil {
					return err
				}
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "notify test: sent event %q to enabled channel(s)\n", event)
				return err
			})
		},
	}
	cmd.Flags().StringVar(&event, "event", notify.EventTest, "event to send: notify.test, pipeline.start, pipeline.success, pipeline.error, pipeline.stalled")
	return cmd
}
