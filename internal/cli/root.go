package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/hrodrig/kzero/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// errPrintSampleDone stops command execution after sample YAML was written to stdout.
var errPrintSampleDone = errors.New("sample config printed")

var (
	cfgFile           string
	logFormat         string
	logLevel          string
	noEnvPassthrough  bool
	printSampleConfig bool
)

// Execute runs the root command tree.
func Execute() error {
	err := newRootCmd().Execute()
	if errors.Is(err, errPrintSampleDone) {
		return nil
	}
	return err
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kzero",
		Short: "Declarative Kubernetes workload shutdown and startup",
		Long: `kzero runs ordered down/up pipelines from configuration (YAML),
so operators can scale workloads and Helm releases in a safe, repeatable way.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if !printSampleConfig {
				return nil
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), configs.SampleYAML())
			if err != nil {
				return err
			}
			return errPrintSampleDone
		},
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: ./kzero.yaml)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log output format: text or json")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "minimum log level: debug, info, warn, or error")
	rootCmd.PersistentFlags().BoolVar(&noEnvPassthrough, "no-env-passthrough", false, "omit host environment from hook and subprocess env (KZERO_* and KUBECONFIG only)")
	rootCmd.PersistentFlags().BoolVar(&printSampleConfig, "print-sample-config", false, "print sample kzero.yaml to stdout and exit")
	cobra.OnInitialize(initConfig)

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(
		newAnalyzeCmd(),
		newTargetCmd(),
		newNotifyCmd(),
		newVerifyCmd(),
		newProbeCmd(),
		newDownCmd(),
		newUpCmd(),
		newResetCmd(),
		newVersionCmd(),
		newCompletionCmd(),
	)

	return rootCmd
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("kzero")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("KZERO")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return
		}
		_, _ = fmt.Fprintln(os.Stderr, "config:", err)
	}
}
