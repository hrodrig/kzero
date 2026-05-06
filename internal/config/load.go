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
	v.AutomaticEnv()

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

	return PipelineStep{
		Ref:       ref,
		Type:      matches[1],
		Namespace: matches[2],
		Name:      matches[3],
	}, nil
}

func parseMapStep(stepMap map[string]interface{}) (PipelineStep, error) {
	if len(stepMap) != 1 {
		return PipelineStep{}, errors.New("map step must have exactly one key")
	}

	for key, value := range stepMap {
		if key == "custom" {
			path, ok := value.(string)
			if !ok || strings.TrimSpace(path) == "" {
				return PipelineStep{}, errors.New("custom step requires non-empty string path")
			}
			return PipelineStep{Custom: path}, nil
		}

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
	if cfg.Pipelines.Down == nil && cfg.Pipelines.Up == nil {
		return errors.New("pipelines.down or pipelines.up is required")
	}
	if err := validateHelmWorkspaceForReleases(cfg); err != nil {
		return err
	}
	return nil
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
