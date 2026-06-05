package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	logFormat string
)

// Execute runs the root command tree.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kzero",
		Short: "Declarative Kubernetes workload shutdown and startup",
		Long: `kzero runs ordered down/up pipelines from configuration (YAML),
so operators can scale workloads and Helm releases in a safe, repeatable way.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: ./kzero.yaml)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log output format: text or json")
	cobra.OnInitialize(initConfig)

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.AddCommand(
		newAnalyzeCmd(),
		newTargetCmd(),
		newNotifyCmd(),
		newDownCmd(),
		newUpCmd(),
		newResetCmd(),
		newVersionCmd(),
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
