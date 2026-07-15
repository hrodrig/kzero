package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/exitcode"
)

func TestRunMain_printSampleConfig(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	errCh := make(chan error, 1)
	var out bytes.Buffer
	go func() {
		_, e := io.Copy(&out, r)
		errCh <- e
	}()

	os.Args = []string{"kzero", "--print-sample-config"}
	if got := runMain(); got != 0 {
		t.Fatalf("runMain: exit code %d, want 0", got)
	}
	w.Close()
	os.Stdout = oldStdout
	if e := <-errCh; e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), `schema_version: "1.0"`) {
		t.Fatalf("expected sample on stdout, got %q", out.String())
	}
}

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
	if got := runMain(); got != exitcode.ConfigError {
		t.Fatalf("runMain: exit code %d, want %d (config)", got, exitcode.ConfigError)
	}
}
