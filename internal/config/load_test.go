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
