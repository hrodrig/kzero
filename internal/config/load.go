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
	"pvc":         {},
	"exec":        {},
}

type rawConfig struct {
	SchemaVersion string                 `mapstructure:"schema_version"`
	Cluster       ClusterConfig          `mapstructure:"cluster"`
	Helm          HelmConfig             `mapstructure:"helm"`
	Client        ClientConfig           `mapstructure:"client"`
	Command       CommandConfig          `mapstructure:"command"`
	Hooks         HooksConfig            `mapstructure:"hooks"`
	Notify        NotifyConfig           `mapstructure:"notify"`
	Verify        VerifyConfig           `mapstructure:"verify"`
	InfraProbe    map[string]interface{} `mapstructure:"infra_probe"`
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
		Verify:        raw.Verify,
		Retry:         raw.Retry,
		Run:           raw.Run,
	}

	if err := parsePipelines(cfg, raw.Pipelines); err != nil {
		return nil, err
	}
	if err := parseInfraProbe(cfg, raw.InfraProbe); err != nil {
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
		"run.operation_timeout",
		"helm.workspace",
		"client.id",
		"command.kubectl",
		"command.helm",
		"command.shell",
		"run.verify",
		"verify.enabled",
		"verify.format",
		"infra_probe.enabled",
		"run.probe_cache_dir",
		"run.no_env_passthrough",
		"run.api_watchdog.enabled",
		"run.api_watchdog.interval",
		"run.api_watchdog.fail_after",
		"notify.on_error",
		"notify.require_delivery",
		"notify.slack.enabled",
		"notify.slack.webhook_url",
		"notify.discord.enabled",
		"notify.discord.webhook_url",
		"notify.teams.enabled",
		"notify.teams.webhook_url",
		"notify.pagerduty.enabled",
		"notify.pagerduty.routing_key",
		"notify.webhook.enabled",
		"notify.webhook.url",
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
		return PipelineStep{}, fmt.Errorf("unsupported step kind %q in %q (supported: deployment, statefulset, release, pvc, exec)", kind, ref)
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
		if err := applyOneStepOption(step, k, v); err != nil {
			return err
		}
	}
	return nil
}

func applyOneStepOption(step *PipelineStep, k string, v interface{}) error {
	switch k {
	case "replicas", "wait_for_ready", "timeout", "pre", "post":
		return applyCommonStepOption(step, k, v)
	case "script":
		return applyReleaseScriptOption(step, v)
	case "chart", "version", "values_files", "create_namespace":
		return applyReleaseStepOption(step, k, v)
	case "container", "command", "stdin":
		return applyExecStepOption(step, k, v)
	default:
		return fmt.Errorf("unsupported option %q", k)
	}
}

func applyCommonStepOption(step *PipelineStep, k string, v interface{}) error {
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
		return fmt.Errorf("unsupported common option %q", k)
	}
	return nil
}

func applyReleaseScriptOption(step *PipelineStep, v interface{}) error {
	if step.Type != "release" {
		return errors.New(`option "script" is only valid on release steps`)
	}
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return errors.New("script must be a non-empty string")
	}
	step.Script = strings.TrimSpace(s)
	return nil
}

func applyReleaseStepOption(step *PipelineStep, k string, v interface{}) error {
	if step.Type != "release" {
		return fmt.Errorf("option %q is only valid on release steps", k)
	}
	switch k {
	case "chart":
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return errors.New("chart must be a non-empty string")
		}
		step.Chart = strings.TrimSpace(s)
	case "version":
		s, ok := v.(string)
		if !ok {
			return errors.New("version must be a string")
		}
		step.Version = strings.TrimSpace(s)
	case "values_files":
		files, err := parseStringList(v)
		if err != nil {
			return fmt.Errorf("values_files: %w", err)
		}
		step.ValuesFiles = files
	case "create_namespace":
		b, ok := v.(bool)
		if !ok {
			return errors.New("create_namespace must be boolean")
		}
		step.CreateNamespace = &b
	default:
		return fmt.Errorf("unsupported release option %q", k)
	}
	return nil
}

func applyExecStepOption(step *PipelineStep, k string, v interface{}) error {
	if step.Type != "exec" {
		return fmt.Errorf("option %q is only valid on exec steps", k)
	}
	switch k {
	case "container":
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return errors.New("container must be a non-empty string")
		}
		step.Container = strings.TrimSpace(s)
	case "command":
		cmd, err := parseStringList(v)
		if err != nil {
			return fmt.Errorf("command: %w", err)
		}
		step.Command = cmd
	case "stdin":
		s, ok := v.(string)
		if !ok {
			return errors.New("stdin must be a string")
		}
		step.Stdin = s
	default:
		return fmt.Errorf("unsupported exec option %q", k)
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
	if err := validateHelmRegistries(cfg); err != nil {
		return err
	}
	if err := validateExecSteps(cfg); err != nil {
		return err
	}
	if err := validateVerify(cfg); err != nil {
		return err
	}
	return validateInfraProbe(cfg)
}

func parseInfraProbe(cfg *Config, raw map[string]interface{}) error {
	if raw == nil {
		return nil
	}
	if err := parseInfraProbeScalars(cfg, raw); err != nil {
		return err
	}
	if err := parseInfraProbePipeline(cfg, raw); err != nil {
		return err
	}
	if v, ok := raw["checks"]; ok {
		checks, err := parseProbeChecks(v)
		if err != nil {
			return fmt.Errorf("parse infra_probe.checks: %w", err)
		}
		cfg.InfraProbe.Checks = checks
	}
	return nil
}

func parseInfraProbeScalars(cfg *Config, raw map[string]interface{}) error {
	if v, ok := raw["enabled"]; ok {
		enabled, err := parseBool(v)
		if err != nil {
			return fmt.Errorf("parse infra_probe.enabled: %w", err)
		}
		cfg.InfraProbe.Enabled = enabled
	}
	if v, ok := raw["before"]; ok {
		before, err := parseStringList(v)
		if err != nil {
			return fmt.Errorf("parse infra_probe.before: %w", err)
		}
		cfg.InfraProbe.Before = before
	}
	if v, ok := raw["fail_fast"]; ok {
		ff, err := parseBool(v)
		if err != nil {
			return fmt.Errorf("parse infra_probe.fail_fast: %w", err)
		}
		cfg.InfraProbe.FailFast = ff
		cfg.infraProbeFailFastSet = true
	}
	if v, ok := raw["cache_ttl"]; ok {
		ttl, err := parseDuration(v)
		if err != nil {
			return fmt.Errorf("parse infra_probe.cache_ttl: %w", err)
		}
		cfg.InfraProbe.CacheTTL = ttl
	}
	return nil
}

func parseInfraProbePipeline(cfg *Config, raw map[string]interface{}) error {
	v, ok := raw["pipeline"]
	if !ok {
		return nil
	}
	pipeMap, ok := v.(map[string]interface{})
	if !ok {
		return errors.New("parse infra_probe.pipeline: must be an object")
	}
	if up, ok := pipeMap["up"]; ok {
		steps, err := parsePipelineList(up)
		if err != nil {
			return fmt.Errorf("parse infra_probe.pipeline.up: %w", err)
		}
		cfg.InfraProbe.Pipeline.Up = steps
	}
	if down, ok := pipeMap["down"]; ok {
		steps, err := parsePipelineList(down)
		if err != nil {
			return fmt.Errorf("parse infra_probe.pipeline.down: %w", err)
		}
		cfg.InfraProbe.Pipeline.Down = steps
	}
	return nil
}

func parseBool(v interface{}) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	default:
		return false, fmt.Errorf("must be boolean")
	}
}

func parseStringList(v interface{}) ([]string, error) {
	items, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("must be a list")
	}
	out := make([]string, 0, len(items))
	for i, item := range items {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("item %d: must be a non-empty string", i)
		}
		out = append(out, strings.TrimSpace(s))
	}
	return out, nil
}

func parseDuration(v interface{}) (time.Duration, error) {
	switch d := v.(type) {
	case string:
		return time.ParseDuration(d)
	case int:
		return time.Duration(d), nil
	case int64:
		return time.Duration(d), nil
	case float64:
		return time.Duration(d), nil
	default:
		return 0, fmt.Errorf("must be duration string")
	}
}

func parseProbeChecks(v interface{}) ([]ProbeCheck, error) {
	items, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("must be a list")
	}
	out := make([]ProbeCheck, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("item %d: must be an object", i)
		}
		if len(m) != 1 {
			return nil, fmt.Errorf("item %d: must have exactly one key", i)
		}
		for k, val := range m {
			check, err := parseProbeCheckItem(i, k, val)
			if err != nil {
				return nil, err
			}
			out = append(out, check)
		}
	}
	return out, nil
}

func parseProbeCheckItem(i int, k string, val interface{}) (ProbeCheck, error) {
	switch k {
	case "pvc_bound":
		s, ok := val.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return ProbeCheck{}, fmt.Errorf("item %d: pvc_bound must be a non-empty string", i)
		}
		return ProbeCheck{PVCBound: strings.TrimSpace(s)}, nil
	case "release_ready":
		b, err := parseBool(val)
		if err != nil {
			return ProbeCheck{}, fmt.Errorf("item %d: release_ready %w", i, err)
		}
		if !b {
			return ProbeCheck{}, fmt.Errorf("item %d: release_ready must be true when set", i)
		}
		return ProbeCheck{ReleaseReady: true}, nil
	case "pods_schedulable":
		b, err := parseBool(val)
		if err != nil {
			return ProbeCheck{}, fmt.Errorf("item %d: pods_schedulable %w", i, err)
		}
		if !b {
			return ProbeCheck{}, fmt.Errorf("item %d: pods_schedulable must be true when set", i)
		}
		return ProbeCheck{PodsSchedulable: true}, nil
	default:
		return ProbeCheck{}, fmt.Errorf("item %d: unknown check %q (want pvc_bound, release_ready, or pods_schedulable)", i, k)
	}
}

func validateInfraProbe(cfg *Config) error {
	p := &cfg.InfraProbe
	if !p.Enabled {
		return nil
	}
	if !cfg.infraProbeFailFastSet {
		p.FailFast = true
	}
	if len(p.Before) == 0 {
		p.Before = []string{"reset"}
	}
	for _, c := range p.Before {
		switch c {
		case "reset", "down":
		default:
			return fmt.Errorf("infra_probe.before: unknown command %q (want reset, down)", c)
		}
	}
	if len(p.Pipeline.Up) == 0 {
		return errors.New("infra_probe.enabled requires pipeline.up")
	}
	if pipelinesContainRelease(p.Pipeline.Up) || pipelinesContainRelease(p.Pipeline.Down) {
		if strings.TrimSpace(cfg.Helm.Workspace) == "" {
			return errors.New("helm.workspace is required when infra_probe pipeline includes release steps")
		}
	}
	for _, c := range p.Checks {
		if c.PVCBound != "" {
			if _, _, err := splitPVCRef(c.PVCBound); err != nil {
				return fmt.Errorf("infra_probe.checks: %w", err)
			}
		}
	}
	return nil
}

func splitPVCRef(ref string) (string, string, error) {
	ref = strings.TrimSpace(ref)
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("pvc_bound must be namespace/name, got %q", ref)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func validateVerify(cfg *Config) error {
	f := strings.TrimSpace(cfg.Verify.Format)
	if f == "" {
		cfg.Verify.Format = "text"
	} else if f != "text" && f != "json" {
		return fmt.Errorf("verify.format must be text or json")
	}
	for _, c := range cfg.Verify.Checks {
		c = strings.TrimSpace(c)
		if c == "" {
			return errors.New("verify.checks: empty check name")
		}
		switch c {
		case "workloads_ready", "nodes_ready", "pods_schedulable":
		default:
			return fmt.Errorf("verify.checks: unknown check %q (want workloads_ready, nodes_ready, pods_schedulable)", c)
		}
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
		// 1.0.0 #32: omitted → native (shell is explicit opt-in).
		cfg.Run.Execution = "native"
		return nil
	}
	switch e {
	case "shell", "native", "auto":
		return nil
	default:
		return fmt.Errorf("run.execution must be one of: shell, native, auto")
	}
}

func validateHelmRegistries(cfg *Config) error {
	seen := make(map[string]struct{})
	for i, reg := range cfg.Helm.Registries {
		host := strings.TrimSpace(reg.Host)
		if host == "" {
			return fmt.Errorf("helm.registries[%d]: host is required", i)
		}
		key := strings.ToLower(host)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("helm.registries[%d]: duplicate host %q", i, host)
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(reg.Username) == "" {
			return fmt.Errorf("helm.registries[%d]: username is required for host %q", i, host)
		}
		if strings.TrimSpace(reg.Password) == "" && strings.TrimSpace(reg.PasswordEnv) == "" {
			return fmt.Errorf("helm.registries[%d]: password or password_env is required for host %q", i, host)
		}
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

func validateExecSteps(cfg *Config) error {
	check := func(label string, steps []PipelineStep) error {
		for i, s := range steps {
			if s.Type != "exec" {
				continue
			}
			if strings.TrimSpace(s.Container) == "" {
				return fmt.Errorf("%s item %d: exec step %q requires container", label, i, s.Ref)
			}
			if len(s.Command) == 0 {
				return fmt.Errorf("%s item %d: exec step %q requires command", label, i, s.Ref)
			}
		}
		return nil
	}
	if err := check("pipelines.down", cfg.Pipelines.Down); err != nil {
		return err
	}
	if err := check("pipelines.up", cfg.Pipelines.Up); err != nil {
		return err
	}
	if err := check("infra_probe.pipeline.down", cfg.InfraProbe.Pipeline.Down); err != nil {
		return err
	}
	return check("infra_probe.pipeline.up", cfg.InfraProbe.Pipeline.Up)
}

func pipelinesContainRelease(steps []PipelineStep) bool {
	for _, s := range steps {
		if s.Type == "release" {
			return true
		}
	}
	return false
}
