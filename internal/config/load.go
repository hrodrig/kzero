package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const supportedSchemaVersion = "1.0"

var stepRefPattern = regexp.MustCompile(`^([a-z0-9-]+)\.([a-z0-9-]+)/([a-zA-Z0-9._-]+)$`)

// supportedStepKinds are the resource kinds the v1 engine knows how to execute
// in pipelines.down / pipelines.up. Any other kind is rejected at parse time
// so analyze surfaces the problem before live mode would.
var supportedStepKinds = map[string]struct{}{
	"deployment":  {},
	"statefulset": {},
	"release":     {},
}

type rawConfig struct {
	SchemaVersion string                 `mapstructure:"schema_version"`
	Cluster       ClusterConfig          `mapstructure:"cluster"`
	Helm          HelmConfig             `mapstructure:"helm"`
	Client        ClientConfig           `mapstructure:"client"`
	Command       CommandConfig          `mapstructure:"command"`
	Hooks         HooksConfig            `mapstructure:"hooks"`
	Notify        NotifyConfig           `mapstructure:"notify"`
	Pipelines     map[string]interface{} `mapstructure:"pipelines"`
	Retry         RetryConfig            `mapstructure:"retry"`
	Run           RunConfig              `mapstructure:"run"`
}

// Load reads and validates kzero configuration from YAML.
func Load(path string) (*Config, error) {
	configPath := path
	if strings.TrimSpace(configPath) == "" {
		configPath = "kzero.yaml"
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("KZERO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	bindConfigEnv(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw rawConfig
	if err := v.Unmarshal(&raw); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	cfg := &Config{
		SchemaVersion: raw.SchemaVersion,
		Cluster:       raw.Cluster,
		Helm:          raw.Helm,
		Client:        raw.Client,
		Command:       raw.Command,
		Hooks:         raw.Hooks,
		Notify:        raw.Notify,
		Retry:         raw.Retry,
		Run:           raw.Run,
	}

	if err := parsePipelines(cfg, raw.Pipelines); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// bindConfigEnv links nested YAML keys to KZERO_* variables so Unmarshal picks up overrides.
func bindConfigEnv(v *viper.Viper) {
	for _, key := range []string{
		"run.mode",
		"run.kubeconfig",
		"run.color",
		"run.execution",
		"run.timeout",
		"run.worker_concurrency",
		"run.operation_timeout",
		"helm.workspace",
		"command.kubectl",
		"command.helm",
	} {
		_ = v.BindEnv(key)
	}
}

func parsePipelines(cfg *Config, raw map[string]interface{}) error {
	if raw == nil {
		return errors.New("pipelines is required")
	}

	if down, ok := raw["down"]; ok {
		steps, err := parsePipelineList(down)
		if err != nil {
			return fmt.Errorf("parse pipelines.down: %w", err)
		}
		cfg.Pipelines.Down = steps
	}

	if up, ok := raw["up"]; ok {
		steps, err := parsePipelineList(up)
		if err != nil {
			return fmt.Errorf("parse pipelines.up: %w", err)
		}
		cfg.Pipelines.Up = steps
	}

	return nil
}

func parsePipelineList(raw interface{}) ([]PipelineStep, error) {
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("must be a list")
	}

	steps := make([]PipelineStep, 0, len(items))
	for i, item := range items {
		step, err := parsePipelineStep(item)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

func parsePipelineStep(raw interface{}) (PipelineStep, error) {
	switch v := raw.(type) {
	case string:
		return parseReferenceStep(v)
	case map[string]interface{}:
		return parseMapStep(v)
	default:
		return PipelineStep{}, fmt.Errorf("unsupported step type %T", raw)
	}
}

func parseReferenceStep(ref string) (PipelineStep, error) {
	matches := stepRefPattern.FindStringSubmatch(ref)
	if len(matches) != 4 {
		return PipelineStep{}, fmt.Errorf("invalid step reference %q", ref)
	}

	kind := matches[1]
	if _, ok := supportedStepKinds[kind]; !ok {
		if kind == "daemonset" {
			return PipelineStep{}, fmt.Errorf(`unsupported step kind "daemonset" in %q: kubectl scale is not supported for DaemonSet; use a custom: step with kubectl patch to set a nodeSelector that drains the pods`, ref)
		}
		return PipelineStep{}, fmt.Errorf("unsupported step kind %q in %q (supported: deployment, statefulset, release)", kind, ref)
	}

	return PipelineStep{
		Ref:       ref,
		Type:      kind,
		Namespace: matches[2],
		Name:      matches[3],
	}, nil
}

func parseMapStep(stepMap map[string]interface{}) (PipelineStep, error) {
	if _, ok := stepMap["custom"]; ok {
		return parseCustomMapStep(stepMap)
	}

	if len(stepMap) != 1 {
		return PipelineStep{}, errors.New("map step must have exactly one key")
	}

	for key, value := range stepMap {
		step, err := parseReferenceStep(key)
		if err != nil {
			return PipelineStep{}, err
		}
		if value == nil {
			return step, nil
		}

		opts, ok := value.(map[string]interface{})
		if !ok {
			return PipelineStep{}, fmt.Errorf("step options for %q must be an object", key)
		}
		if err := applyStepOptions(&step, opts); err != nil {
			return PipelineStep{}, fmt.Errorf("invalid options for %q: %w", key, err)
		}
		return step, nil
	}

	return PipelineStep{}, errors.New("unreachable map step parser state")
}

func parseCustomMapStep(stepMap map[string]interface{}) (PipelineStep, error) {
	customVal, ok := stepMap["custom"]
	if !ok {
		return PipelineStep{}, errors.New("custom step requires custom key")
	}
	path, ok := customVal.(string)
	if !ok || strings.TrimSpace(path) == "" {
		return PipelineStep{}, errors.New("custom step requires non-empty string path")
	}
	step := PipelineStep{Custom: strings.TrimSpace(path)}
	for k, v := range stepMap {
		if k == "custom" {
			continue
		}
		switch k {
		case "pre", "post":
			s, ok := v.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return PipelineStep{}, fmt.Errorf("custom step field %q must be a non-empty string", k)
			}
			if k == "pre" {
				step.PreStep = strings.TrimSpace(s)
			} else {
				step.PostStep = strings.TrimSpace(s)
			}
		default:
			return PipelineStep{}, fmt.Errorf("unsupported key %q in custom step (allowed: custom, pre, post)", k)
		}
	}
	return step, nil
}

func applyStepOptions(step *PipelineStep, opts map[string]interface{}) error {
	for k, v := range opts {
		switch k {
		case "replicas":
			replicas, err := parseReplicas(v)
			if err != nil {
				return err
			}
			step.Replicas = &replicas
		case "wait_for_ready":
			wait, ok := v.(bool)
			if !ok {
				return errors.New("wait_for_ready must be boolean")
			}
			step.WaitForReady = wait
		case "timeout":
			timeoutRaw, ok := v.(string)
			if !ok {
				return errors.New("timeout must be duration string")
			}
			timeout, err := time.ParseDuration(timeoutRaw)
			if err != nil {
				return fmt.Errorf("invalid timeout duration: %w", err)
			}
			step.Timeout = timeout
		case "pre", "post":
			s, ok := v.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return fmt.Errorf("%s must be a non-empty string", k)
			}
			if k == "pre" {
				step.PreStep = strings.TrimSpace(s)
			} else {
				step.PostStep = strings.TrimSpace(s)
			}
		default:
			return fmt.Errorf("unsupported option %q", k)
		}
	}

	return nil
}

func parseReplicas(v interface{}) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		if n != float64(int(n)) {
			return 0, errors.New("replicas must be an integer")
		}
		return int(n), nil
	default:
		return 0, errors.New("replicas must be numeric")
	}
}

func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.SchemaVersion) == "" {
		return errors.New("schema_version is required")
	}
	if cfg.SchemaVersion != supportedSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", cfg.SchemaVersion)
	}
	if cfg.Run.Mode == "" {
		return errors.New("run.mode is required")
	}
	if cfg.Run.Mode != "dry-run" && cfg.Run.Mode != "live" {
		return fmt.Errorf("run.mode must be one of: dry-run, live")
	}
	if err := validateRunExecution(cfg); err != nil {
		return err
	}
	if err := validateRunColor(cfg); err != nil {
		return err
	}
	if cfg.Pipelines.Down == nil && cfg.Pipelines.Up == nil {
		return errors.New("pipelines.down or pipelines.up is required")
	}
	if err := validateHelmWorkspaceForReleases(cfg); err != nil {
		return err
	}
	return nil
}

func validateRunColor(cfg *Config) error {
	c := strings.TrimSpace(cfg.Run.Color)
	if c == "" {
		cfg.Run.Color = "auto"
		return nil
	}
	switch c {
	case "auto", "always", "never":
		return nil
	default:
		return fmt.Errorf("run.color must be one of: auto, always, never")
	}
}

func validateRunExecution(cfg *Config) error {
	e := strings.TrimSpace(cfg.Run.Execution)
	if e == "" {
		cfg.Run.Execution = "shell"
		return nil
	}
	switch e {
	case "shell", "native", "auto":
		return nil
	default:
		return fmt.Errorf("run.execution must be one of: shell, native, auto")
	}
}

func validateHelmWorkspaceForReleases(cfg *Config) error {
	if !pipelinesContainRelease(cfg.Pipelines.Down) && !pipelinesContainRelease(cfg.Pipelines.Up) {
		return nil
	}
	if strings.TrimSpace(cfg.Helm.Workspace) == "" {
		return errors.New("helm.workspace is required when pipelines include release steps")
	}
	return nil
}

func pipelinesContainRelease(steps []PipelineStep) bool {
	for _, s := range steps {
		if s.Type == "release" {
			return true
		}
	}
	return false
}
