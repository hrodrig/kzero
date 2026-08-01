package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOverrideRunMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_RUN_MODE", "live")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Run.Mode != "live" {
		t.Fatalf("got mode %q, want live", cfg.Run.Mode)
	}
}

func TestEnvOverrideClientID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
client:
  id: "from-yaml"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_CLIENT_ID", "from-env")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Client.ID != "from-env" {
		t.Fatalf("got client.id %q, want from-env", cfg.Client.ID)
	}
}

func TestEnvOverrideNotifyOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
notify:
  on_error: false
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_NOTIFY_ON_ERROR", "true")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Notify.OnError == nil || *cfg.Notify.OnError != true {
		t.Fatalf("expected notify.on_error=true from env override, got %v", cfg.Notify.OnError)
	}
}

func TestEnvOverrideNotifyRequireDelivery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_NOTIFY_REQUIRE_DELIVERY", "true")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Notify.RequireDelivery == nil || *cfg.Notify.RequireDelivery != true {
		t.Fatalf("expected notify.require_delivery=true from env, got %v", cfg.Notify.RequireDelivery)
	}
}

func TestEnvOverrideAPIWatchdogEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "live"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_RUN_API_WATCHDOG_ENABLED", "true")
	t.Setenv("KZERO_RUN_API_WATCHDOG_INTERVAL", "30s")
	t.Setenv("KZERO_RUN_API_WATCHDOG_FAIL_AFTER", "3m")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Run.APIWatchdog == nil {
		t.Fatal("expected api_watchdog to be configured from env")
	}
	if !cfg.Run.APIWatchdog.Enabled {
		t.Fatalf("expected api_watchdog.enabled=true, got false")
	}
	if cfg.Run.APIWatchdog.Interval.Seconds() != 30 {
		t.Fatalf("expected interval=30s, got %s", cfg.Run.APIWatchdog.Interval)
	}
	if cfg.Run.APIWatchdog.FailAfter.Minutes() != 3 {
		t.Fatalf("expected fail_after=3m, got %s", cfg.Run.APIWatchdog.FailAfter)
	}
}

func TestEnvOverrideCommandShell(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
command:
  shell: "/bin/sh"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_COMMAND_SHELL", "/bin/bash")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Command.Shell != "/bin/bash" {
		t.Fatalf("got command.shell %q, want /bin/bash", cfg.Command.Shell)
	}
}
