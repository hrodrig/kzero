package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunMain_version(t *testing.T) {
	orig := os.Args
	t.Cleanup(func() { os.Args = orig })
	os.Args = []string{"kzero", "version"}
	if got := runMain(); got != 0 {
		t.Fatalf("runMain: exit code %d, want 0", got)
	}
}

func TestRunMain_analyzeInvalidConfig(t *testing.T) {
	orig := os.Args
	t.Cleanup(func() { os.Args = orig })

	dir := t.TempDir()
	cfg := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(cfg, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - "not-a-valid-step"
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"kzero", "analyze", "--config", cfg}
	if got := runMain(); got != 1 {
		t.Fatalf("runMain: exit code %d, want 1", got)
	}
}
