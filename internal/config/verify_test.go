package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_verifyInvalidCheck(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
pipelines:
  down: []
  up: []
verify:
  checks:
    - bogus
run:
  mode: "dry-run"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "verify.checks") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadConfig_verifyFormat(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	content := `
schema_version: "1.0"
pipelines:
  down: []
  up: []
verify:
  format: json
  checks:
    - workloads_ready
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
	if cfg.Verify.Format != "json" {
		t.Fatalf("format %q", cfg.Verify.Format)
	}
}
