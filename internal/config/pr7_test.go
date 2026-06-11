package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_verifyPodsSchedulable(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
pipelines:
  down: []
  up: []
verify:
  checks:
    - pods_schedulable
run:
  mode: "dry-run"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Verify.Checks) != 1 || cfg.Verify.Checks[0] != "pods_schedulable" {
		t.Fatalf("checks: %#v", cfg.Verify.Checks)
	}
}

func TestLoadConfig_helmRegistries(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
pipelines:
  down: []
  up: []
helm:
  registries:
    - host: ghcr.io
      username: bot
      password_env: GHCR_TOKEN
run:
  mode: "dry-run"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Helm.Registries) != 1 || cfg.Helm.Registries[0].Host != "ghcr.io" {
		t.Fatalf("registries: %#v", cfg.Helm.Registries)
	}
}

func TestLoadConfig_helmRegistriesValidation(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
pipelines:
  down: []
  up: []
helm:
  registries:
    - host: ghcr.io
      username: bot
run:
  mode: "dry-run"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadConfig_releaseScriptOption(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
helm:
  workspace: ./charts
pipelines:
  down: []
  up:
    - release.mon/prom:
        script: monitoring/prom.sh
run:
  mode: "dry-run"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Pipelines.Up) != 1 || cfg.Pipelines.Up[0].Script != "monitoring/prom.sh" {
		t.Fatalf("step: %#v", cfg.Pipelines.Up)
	}
}

func TestLoadConfig_helmRegistriesDuplicateHost(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
pipelines:
  down: []
  up: []
helm:
  registries:
    - host: ghcr.io
      username: a
      password: x
    - host: GHCR.IO
      username: b
      password: y
run:
  mode: "dry-run"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadConfig_infraProbePodsSchedulable(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
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
    - pods_schedulable: true
run:
  mode: dry-run
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InfraProbe.Checks[0].PodsSchedulable {
		t.Fatalf("checks: %#v", cfg.InfraProbe.Checks)
	}
}
