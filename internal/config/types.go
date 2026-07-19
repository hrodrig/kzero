package config

import "time"

// Config is the root configuration contract for kzero v1.
type Config struct {
	SchemaVersion string           `mapstructure:"schema_version"`
	Cluster       ClusterConfig    `mapstructure:"cluster"`
	Helm          HelmConfig       `mapstructure:"helm"`
	Client        ClientConfig     `mapstructure:"client"`
	Command       CommandConfig    `mapstructure:"command"`
	Hooks         HooksConfig      `mapstructure:"hooks"`
	Notify        NotifyConfig     `mapstructure:"notify"`
	Verify        VerifyConfig     `mapstructure:"verify"`
	InfraProbe    InfraProbeConfig `mapstructure:"infra_probe"`
	Pipelines     PipelinesConfig  `mapstructure:"pipelines"`
	Retry         RetryConfig      `mapstructure:"retry"`
	Run           RunConfig        `mapstructure:"run"`

	infraProbeFailFastSet bool
}

// VerifyConfig controls post-up readiness checks (`kzero verify`).
type VerifyConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Checks  []string `mapstructure:"checks"`
	Format  string   `mapstructure:"format"`
}

// InfraProbeConfig gates destructive commands with a mini-pipeline + optional checks.
type InfraProbeConfig struct {
	Enabled  bool                `mapstructure:"enabled"`
	Before   []string            `mapstructure:"before"`
	FailFast bool                `mapstructure:"fail_fast"`
	CacheTTL time.Duration       `mapstructure:"cache_ttl"`
	Pipeline ProbePipelineConfig `mapstructure:"pipeline"`
	Checks   []ProbeCheck        `mapstructure:"checks"`
}

// ProbePipelineConfig is the declarative probe up/down step list.
type ProbePipelineConfig struct {
	Up   []PipelineStep
	Down []PipelineStep
}

// ProbeCheck is one post-probe-up validation (pvc_bound, release_ready, pods_schedulable).
type ProbeCheck struct {
	PVCBound        string `mapstructure:"pvc_bound"`
	ReleaseReady    bool   `mapstructure:"release_ready"`
	PodsSchedulable bool   `mapstructure:"pods_schedulable"`
}

type ClusterConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Description string `mapstructure:"description"`
}

type HelmConfig struct {
	Workspace  string               `mapstructure:"workspace"`
	Registries []HelmRegistryConfig `mapstructure:"registries"`
}

// HelmRegistryConfig holds OCI registry credentials for Helm SDK chart pulls.
type HelmRegistryConfig struct {
	Host        string `mapstructure:"host"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	PasswordEnv string `mapstructure:"password_env"`
}

type ClientConfig struct {
	ID string `mapstructure:"id"`
}

type CommandConfig struct {
	Helm    string `mapstructure:"helm"`
	Kubectl string `mapstructure:"kubectl"`
	// Shell is the interpreter for phase hooks, per-step pre/post, custom: scripts,
	// and shell-path release .sh scripts. Empty means /bin/sh (POSIX). Shebang is ignored.
	Shell string `mapstructure:"shell"`
}

type HooksConfig struct {
	PreDown  string `mapstructure:"pre-down"`
	PostDown string `mapstructure:"post-down"`
	PreUp    string `mapstructure:"pre-up"`
	PostUp   string `mapstructure:"post-up"`
	OnError  string `mapstructure:"on-error"`
}

type NotifyConfig struct {
	OnError *bool `mapstructure:"on_error"`
	// RequireDelivery, when true, fails the pipeline if pipeline.error or
	// pipeline.stalled notify POST(s) cannot be sent. Channel fan-out is
	// allowed: at least one channel must succeed; otherwise the pipeline
	// exits non-zero.
	RequireDelivery *bool                `mapstructure:"require_delivery"`
	Slack           ChannelConfig        `mapstructure:"slack"`
	Discord         ChannelConfig        `mapstructure:"discord"`
	Teams           ChannelConfig        `mapstructure:"teams"`
	PagerDuty       PagerDutyConfig      `mapstructure:"pagerduty"`
	Webhook         GenericWebhookConfig `mapstructure:"webhook"`
}

type ChannelConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

type PagerDutyConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	RoutingKey string `mapstructure:"routing_key"`
}

type GenericWebhookConfig struct {
	Enabled bool              `mapstructure:"enabled"`
	URL     string            `mapstructure:"url"`
	Headers map[string]string `mapstructure:"headers"`
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
	// Release options (release steps only).
	Script string `mapstructure:"script"` // shell path relative to helm.workspace (default <name>.sh)
	// SDK options (used when run.execution is native/auto).
	Chart           string   `mapstructure:"chart"`
	Version         string   `mapstructure:"version"`
	ValuesFiles     []string `mapstructure:"values_files"`
	CreateNamespace *bool    `mapstructure:"create_namespace"`
	// Exec options (exec steps only).
	Container string   `mapstructure:"container"`
	Command   []string `mapstructure:"command"`
	Stdin     string   `mapstructure:"stdin"`
}

type RetryConfig struct {
	Attempts int           `mapstructure:"attempts"`
	Delay    time.Duration `mapstructure:"delay"`
}

type RunConfig struct {
	Kubeconfig string `mapstructure:"kubeconfig"`
	Mode       string `mapstructure:"mode"`
	// Color controls ANSI styling on the command timing line: auto, always, or never.
	Color string `mapstructure:"color"`
	// Execution selects workload step backend: native (client-go + Helm SDK; default when omitted),
	// shell (kubectl/helm subprocesses; opt-in), or auto (native with shell fallback).
	Execution        string        `mapstructure:"execution"`
	Timeout          time.Duration `mapstructure:"timeout"`
	OperationTimeout time.Duration `mapstructure:"operation_timeout"`
	// Verify runs kzero verify after a successful up or reset (non-zero exit on failure).
	Verify bool `mapstructure:"verify"`
	// ProbeCacheDir stores infra probe cache files (empty = OS temp dir).
	ProbeCacheDir string `mapstructure:"probe_cache_dir"`
	// NoEnvPassthrough when true omits os.Environ from hook/release/kubectl subprocesses.
	NoEnvPassthrough bool `mapstructure:"no_env_passthrough"`
	// APIWatchdog configures the periodic API reachability check between
	// steps and during long waits (planned: live down/up/reset only).
	// nil means "not configured" (engine default). When present, schema is
	// parsed and the value drives the Deferred summary; the watchdog
	// goroutine itself lands in PR3 #36.
	APIWatchdog *APIWatchdogConfig `mapstructure:"api_watchdog"`
}

// APIWatchdogConfig mirrors run.api_watchdog.* in YAML.
//
//	enabled:    activates the watchdog during live runs.
//	interval:   period between API reachability checks (Go duration, e.g. "60s").
//	fail_after: cumulative deadline after which a run is aborted if the API
//	            has remained unreachable; uses the engine's notify pipeline.error
//	            channel. Implemented in PR3 #36.
type APIWatchdogConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	Interval  time.Duration `mapstructure:"interval"`
	FailAfter time.Duration `mapstructure:"fail_after"`
}
