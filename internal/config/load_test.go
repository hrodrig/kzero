package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"os"
)

func TestLoadConfig_MinimalValid(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.SchemaVersion != "1.0" {
		t.Fatalf("expected schema_version 1.0, got %q", cfg.SchemaVersion)
	}
	if len(cfg.Pipelines.Down) != 1 {
		t.Fatalf("expected 1 down step, got %d", len(cfg.Pipelines.Down))
	}
	step := cfg.Pipelines.Down[0]
	if step.Type != "deployment" || step.Namespace != "argocd" || step.Name != "argocd-server" {
		t.Fatalf("unexpected parsed step: %#v", step)
	}
}

func TestLoadConfig_UnsupportedSchemaVersion(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "2.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertContains(t, err.Error(), "unsupported schema_version")
}

func TestLoadConfig_InvalidPipelineStep(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - "invalid-step-format"
run:
  mode: "dry-run"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertContains(t, err.Error(), "invalid step reference")
}

func TestLoadConfig_StatefulSetUpOptions(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  up:
    - statefulset.database/postgresql:
        replicas: 3
        wait_for_ready: true
        timeout: 10m
run:
  mode: "live"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(cfg.Pipelines.Up) != 1 {
		t.Fatalf("expected 1 up step, got %d", len(cfg.Pipelines.Up))
	}
	step := cfg.Pipelines.Up[0]
	if step.Type != "statefulset" || step.Namespace != "database" || step.Name != "postgresql" {
		t.Fatalf("unexpected up step identity: %#v", step)
	}
	if step.Replicas == nil || *step.Replicas != 3 {
		t.Fatalf("expected replicas=3, got %#v", step.Replicas)
	}
	if !step.WaitForReady {
		t.Fatal("expected wait_for_ready=true")
	}
	if step.Timeout != 10*time.Minute {
		t.Fatalf("expected timeout=10m, got %s", step.Timeout)
	}
}

func TestLoadConfig_PipelineStepPrePost(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.apps/widget:
        pre: ./hooks/before-widget.sh
        post: ./hooks/after-widget.sh
run:
  mode: "dry-run"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	step := cfg.Pipelines.Down[0]
	if step.PreStep != "./hooks/before-widget.sh" || step.PostStep != "./hooks/after-widget.sh" {
		t.Fatalf("unexpected pre/post: %#v", step)
	}
}

func TestLoadConfig_CustomStepWithPrePost(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - custom: ./hooks/main.sh
      pre: ./hooks/pre.sh
      post: ./hooks/post.sh
run:
  mode: "dry-run"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	step := cfg.Pipelines.Down[0]
	if step.Custom != "./hooks/main.sh" || step.PreStep != "./hooks/pre.sh" || step.PostStep != "./hooks/post.sh" {
		t.Fatalf("unexpected step: %#v", step)
	}
}

func TestLoadConfig_CustomStepInvalidExtraKey(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - custom: ./hooks/main.sh
      replicas: 1
run:
  mode: "dry-run"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "unsupported key")
}

func TestLoadConfig_ReleaseRequiresHelmWorkspace(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - release.monitoring/prom
run:
  mode: "live"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for release without helm.workspace")
	}
	assertContains(t, err.Error(), "helm.workspace")
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("failed writing temp config: %v", err)
	}
	return path
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}

func TestLoadConfig_readMissingFile(t *testing.T) {
	t.Parallel()
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "read config")
}

func TestLoadConfig_runModeInvalid(t *testing.T) {
	t.Parallel()
	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: "staging"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "run.mode")
}

func TestLoadConfig_invalidRunColorRejected(t *testing.T) {
	t.Parallel()
	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: "dry-run"
  color: "rainbow"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "run.color")
}

func TestLoadConfig_runColorDefaultsToAuto(t *testing.T) {
	t.Parallel()
	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: "dry-run"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Run.Color != "auto" {
		t.Fatalf("got run.color %q", cfg.Run.Color)
	}
}

func TestLoadConfig_noPipelineStepsRejected(t *testing.T) {
	t.Parallel()
	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines: {}
run:
  mode: "dry-run"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "pipelines")
}

func TestLoadConfig_DaemonSetKindRejected(t *testing.T) {
	t.Parallel()
	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - daemonset.kube-system/some-agent
run:
  mode: "dry-run"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for daemonset step kind")
	}
	assertContains(t, err.Error(), `unsupported step kind "daemonset"`)
	assertContains(t, err.Error(), "nodeSelector")
}

func TestLoadConfig_UnsupportedKindRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ref  string
		want string
	}{
		{name: "cronjob", ref: "cronjob.batch/nightly", want: `unsupported step kind "cronjob"`},
		{name: "job", ref: "job.batch/migrate", want: `unsupported step kind "job"`},
		{name: "service", ref: "service.default/api", want: `unsupported step kind "service"`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - `+tc.ref+`
run:
  mode: "dry-run"
`)
			_, err := Load(path)
			if err == nil {
				t.Fatalf("expected error for %s", tc.ref)
			}
			assertContains(t, err.Error(), tc.want)
			assertContains(t, err.Error(), "supported: deployment, statefulset, release")
		})
	}
}

func TestLoadConfig_RunExecutionDefaultAndValidation(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: "dry-run"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Run.Execution != "shell" {
		t.Fatalf("expected default execution shell, got %q", cfg.Run.Execution)
	}

	bad := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: "dry-run"
  execution: "kubectl"
`)
	if _, err := Load(bad); err == nil || !strings.Contains(err.Error(), "run.execution") {
		t.Fatalf("expected run.execution validation error, got %v", err)
	}
}

func TestLoadConfig_InfraProbe(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
helm:
  workspace: ./helm
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  before: ["down", "reset"]
  cache_ttl: 15m
  pipeline:
    up:
      - release.probe-ns/probe-storage
    down:
      - release.probe-ns/probe-storage
  checks:
    - pvc_bound: probe-ns/probe-pvc
    - release_ready: true
run:
  mode: "dry-run"
  probe_cache_dir: /tmp/kzero-probe
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.InfraProbe.Enabled {
		t.Fatal("expected enabled")
	}
	if !cfg.InfraProbe.FailFast {
		t.Fatal("expected default fail_fast true")
	}
	if cfg.InfraProbe.CacheTTL != 15*time.Minute {
		t.Fatalf("cache_ttl: got %v", cfg.InfraProbe.CacheTTL)
	}
	if len(cfg.InfraProbe.Pipeline.Up) != 1 || cfg.InfraProbe.Pipeline.Up[0].Name != "probe-storage" {
		t.Fatalf("pipeline.up: %#v", cfg.InfraProbe.Pipeline.Up)
	}
	if len(cfg.InfraProbe.Checks) != 2 {
		t.Fatalf("checks: %#v", cfg.InfraProbe.Checks)
	}
	if cfg.Run.ProbeCacheDir != "/tmp/kzero-probe" {
		t.Fatalf("probe_cache_dir: %q", cfg.Run.ProbeCacheDir)
	}
}

func TestLoadConfig_InfraProbeValidationErrors(t *testing.T) {
	t.Parallel()

	base := `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
run:
  mode: dry-run
`
	if _, err := Load(writeTempConfig(t, base)); err == nil || !strings.Contains(err.Error(), "pipeline.up") {
		t.Fatalf("expected pipeline.up required, got %v", err)
	}

	withBefore := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  before: ["up"]
  pipeline:
    up:
      - deployment.ns/app
run:
  mode: dry-run
`)
	if _, err := Load(withBefore); err == nil || !strings.Contains(err.Error(), "before") {
		t.Fatalf("expected before validation error, got %v", err)
	}

	badPVC := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  pipeline:
    up:
      - deployment.ns/app
  checks:
    - pvc_bound: invalid
run:
  mode: dry-run
`)
	if _, err := Load(badPVC); err == nil || !strings.Contains(err.Error(), "pvc_bound") {
		t.Fatalf("expected pvc_bound validation error, got %v", err)
	}

	releaseNoHelm := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  pipeline:
    up:
      - release.ns/probe
run:
  mode: dry-run
`)
	if _, err := Load(releaseNoHelm); err == nil || !strings.Contains(err.Error(), "helm.workspace") {
		t.Fatalf("expected helm.workspace error, got %v", err)
	}
}

func TestLoadConfig_InfraProbeDefaultBefore(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  pipeline:
    up:
      - deployment.ns/probe
run:
  mode: dry-run
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.InfraProbe.Before) != 1 || cfg.InfraProbe.Before[0] != "reset" {
		t.Fatalf("default before: %#v", cfg.InfraProbe.Before)
	}
}

func TestLoadConfig_InfraProbeFailFastFalse(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  fail_fast: false
  pipeline:
    up:
      - deployment.ns/probe
run:
  mode: dry-run
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InfraProbe.FailFast {
		t.Fatal("expected explicit fail_fast false")
	}
}
