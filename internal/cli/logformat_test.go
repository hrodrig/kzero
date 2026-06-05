package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDown_jsonLogFormat(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
  up: []
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"down", "--config", cfgPath, "--log-format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var jsonLines int
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		if !json.Valid([]byte(line)) {
			t.Fatalf("invalid JSON line in stdout: %q", line)
		}
		jsonLines++
	}
	if jsonLines < 1 {
		t.Fatalf("expected JSON engine lines, stdout=%q", stdout.String())
	}
	summary := strings.TrimSpace(stderr.String())
	if !json.Valid([]byte(summary)) {
		t.Fatalf("expected JSON command summary on stderr, got %q", summary)
	}
}

func TestDown_invalidLogFormat(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"down", "--config", cfgPath, "--log-format", "xml"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid log-format")
	}
}
