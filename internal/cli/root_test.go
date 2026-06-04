package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/validate"
)

func stubClusterValidationSkipped(t *testing.T) {
	t.Helper()
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	old := validate.DefaultClientFactory
	validate.DefaultClientFactory = func(string) (kubernetes.Interface, error) {
		return nil, errors.New("test: skip cluster validation")
	}
	t.Cleanup(func() { validate.DefaultClientFactory = old })
}

func TestRootCommand_HasExpectedSubcommands(t *testing.T) {
	// Do not use t.Parallel: newRootCmd binds package-level cfgFile and
	// registers cobra.OnInitialize, which races under go test -race.
	cmd := newRootCmd()
	expected := []string{"analyze", "target", "down", "up", "reset", "version"}
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
	stubClusterValidationSkipped(t)
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
	for _, want := range []string{
		"Pipeline steps: down=1 up=0",
		"[down]",
		"0: deployment.argocd/argocd-server",
		"Run mode: dry-run",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q; full output: %q", want, out)
		}
	}
}

func TestAnalyze_sampleStyleListsStepsAndDeferredOnStdout(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
helm:
  workspace: "./helm-assets"
pipelines:
  down:
    - deployment.argocd/argocd-server
    - release.monitoring/kube-prometheus-stack
  up:
    - deployment.argocd/argocd-server
retry:
  attempts: 3
run:
  mode: "dry-run"
  worker_concurrency: 4
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
	out := stdout.String()
	for _, want := range []string{
		"[down]",
		"[up]",
		"release.monitoring/kube-prometheus-stack",
		"helm-assets/kube-prometheus-stack.sh",
		"Deferred (accepted by schema",
		"run.worker_concurrency=4",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q; full output: %q", want, out)
		}
	}
	if strings.Contains(stderr.String(), "Deferred") {
		t.Fatalf("deferred summary should be on stdout, not stderr: %q", stderr.String())
	}
}

func TestAnalyze_deferredFeatureWarningsOnStderr(t *testing.T) {
	stubClusterValidationSkipped(t)
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
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
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
	if !strings.Contains(stdout.String(), "Kubernetes target:") {
		t.Fatalf("expected kubernetes target block, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Fatalf("expected dry-run log lines, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "kzero down finished in") {
		t.Fatalf("expected elapsed summary, got: %q", stdout.String())
	}
}

func TestUp_dryRunCompletes(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
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
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
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
