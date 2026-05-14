package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommand_HasExpectedSubcommands(t *testing.T) {
	// Do not use t.Parallel: newRootCmd binds package-level cfgFile and
	// registers cobra.OnInitialize, which races under go test -race.
	cmd := newRootCmd()
	expected := []string{"analyze", "down", "up", "reset", "version"}
	for _, name := range expected {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("expected subcommand %q to exist: %v", name, err)
		}
	}
}

func TestAnalyze_InvalidConfigExitCode(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - "not-a-valid-step"
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"analyze", "--config", cfgPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected analyze to fail with invalid config")
	}
}

func TestVersionCommand_PrintsMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "kzero") || !strings.Contains(out, Version) {
		t.Fatalf("unexpected stdout: %q", out)
	}
}

func TestAnalyze_validConfigPrintsSummary(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"analyze", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Pipeline steps: down=1 up=0") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestAnalyze_deferredFeatureWarningsOnStderr(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
retry:
  attempts: 2
run:
  mode: "dry-run"
  worker_concurrency: 2
notify:
  slack:
    enabled: true
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"analyze", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	errOut := stderr.String()
	for _, want := range []string{
		"warning: run.worker_concurrency=2",
		"warning: retry.attempts=2",
		"warning: notify.slack.enabled",
	} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("stderr missing %q; full stderr: %q", want, errOut)
		}
	}
}

func TestDown_dryRunCompletes(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/widget
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"down", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Fatalf("expected dry-run log lines, got: %q", stdout.String())
	}
}

func TestUp_dryRunCompletes(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  up:
    - deployment.ns/widget
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"up", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Fatalf("expected dry-run log lines, got: %q", stdout.String())
	}
}

func TestReset_dryRunRunsDownThenUp(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/a
  up:
    - deployment.ns/a
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"reset", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "pipeline down") || !strings.Contains(out, "pipeline up") {
		t.Fatalf("expected down and up pipelines in output: %q", out)
	}
}
