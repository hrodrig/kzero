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
