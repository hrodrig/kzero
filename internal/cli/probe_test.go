package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProbeCmd_dryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "kzero.yaml")
	cfg := `
schema_version: "1.0"
helm:
  workspace: ./helm
cluster:
  name: probe-test
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  pipeline:
    up:
      - release.probe-ns/probe-storage
    down:
      - release.probe-ns/probe-storage
run:
  mode: dry-run
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"probe", "-c", cfgPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, errOut.String())
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "infra probe") {
		t.Fatalf("expected infra probe log lines, got:\n%s", combined)
	}
}
