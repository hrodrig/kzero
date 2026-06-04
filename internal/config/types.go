package config

import "time"

// Config is the root configuration contract for kzero v1.
type Config struct {
	SchemaVersion string          `mapstructure:"schema_version"`
	Cluster       ClusterConfig   `mapstructure:"cluster"`
	Helm          HelmConfig      `mapstructure:"helm"`
	Client        ClientConfig    `mapstructure:"client"`
	Command       CommandConfig   `mapstructure:"command"`
	Hooks         HooksConfig     `mapstructure:"hooks"`
	Notify        NotifyConfig    `mapstructure:"notify"`
	Pipelines     PipelinesConfig `mapstructure:"pipelines"`
	Retry         RetryConfig     `mapstructure:"retry"`
	Run           RunConfig       `mapstructure:"run"`
}

type ClusterConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Description string `mapstructure:"description"`
}

type HelmConfig struct {
	Workspace string `mapstructure:"workspace"`
}

type ClientConfig struct {
	ID string `mapstructure:"id"`
}

type CommandConfig struct {
	Helm    string `mapstructure:"helm"`
	Kubectl string `mapstructure:"kubectl"`
}

type HooksConfig struct {
	PreDown  string `mapstructure:"pre-down"`
	PostDown string `mapstructure:"post-down"`
	PreUp    string `mapstructure:"pre-up"`
	PostUp   string `mapstructure:"post-up"`
	OnError  string `mapstructure:"on-error"`
}

type NotifyConfig struct {
	Slack   ChannelConfig `mapstructure:"slack"`
	Discord ChannelConfig `mapstructure:"discord"`
}

type ChannelConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

type PipelinesConfig struct {
	Down []PipelineStep `mapstructure:"down"`
	Up   []PipelineStep `mapstructure:"up"`
}

type PipelineStep struct {
	Ref       string
	Type      string
	Namespace string
	Name      string
	Custom    string
	// PreStep and PostStep are optional shell scripts run immediately before and
	// after the main step action (scale, release, or custom). Post runs only if
	// the main action succeeds.
	PreStep      string `mapstructure:"pre"`
	PostStep     string `mapstructure:"post"`
	Replicas     *int
	WaitForReady bool
	Timeout      time.Duration
}

type RetryConfig struct {
	Attempts int           `mapstructure:"attempts"`
	Delay    time.Duration `mapstructure:"delay"`
}

type RunConfig struct {
	Kubeconfig string `mapstructure:"kubeconfig"`
	Mode       string `mapstructure:"mode"`
	// Execution selects workload step backend: shell (kubectl), native (client-go), or auto.
	Execution         string        `mapstructure:"execution"`
	Timeout           time.Duration `mapstructure:"timeout"`
	WorkerConcurrency int           `mapstructure:"worker_concurrency"`
	OperationTimeout  time.Duration `mapstructure:"operation_timeout"`
}
