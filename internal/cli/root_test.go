package cli

import (
	"bytes"
	"errors"
	"io"
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

func TestExecute_printSampleConfig(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"kzero", "--print-sample-config"}

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

	if err := Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout
	if e := <-errCh; e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), "pipelines:") {
		t.Fatalf("expected sample on stdout, got %q", out.String())
	}
}

func TestRoot_printSampleConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--print-sample-config"})
	err := cmd.Execute()
	if err != nil && !errors.Is(err, errPrintSampleDone) {
		t.Fatalf("Execute: %v", err)
	}
	if stdout.Len() == 0 || !strings.Contains(stdout.String(), `schema_version: "1.0"`) {
		t.Fatalf("expected sample on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("sample must not go to stderr; stderr=%q", stderr.String())
	}
}

func TestAnalyze_printSampleConfig(t *testing.T) {
	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"analyze", "--print-sample-config"})
	err := cmd.Execute()
	if err != nil && !errors.Is(err, errPrintSampleDone) {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "pipelines:") {
		t.Fatalf("expected sample on stdout, got %q", stdout.String())
	}
}

func TestRootCommand_HasExpectedSubcommands(t *testing.T) {
	// Do not use t.Parallel: newRootCmd binds package-level cfgFile and
	// registers cobra.OnInitialize, which races under go test -race.
	cmd := newRootCmd()
	expected := []string{"analyze", "target", "notify", "verify", "down", "up", "reset", "version"}
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

func TestAnalyze_sampleStyleListsStepsOnStdout(t *testing.T) {
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
    - release.monitoring/kube-prometheus-stack
run:
  mode: "dry-run"
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
	out := stdout.String()
	for _, want := range []string{
		"[down]",
		"[up]",
		"release.monitoring/kube-prometheus-stack",
		"helm uninstall --wait --ignore-not-found",
		"script: helm-assets/kube-prometheus-stack.sh",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q; full output: %q", want, out)
		}
	}
	if strings.Contains(out, "Deferred (accepted by schema") {
		t.Fatalf("notify is implemented; unexpected deferred block: %q", out)
	}
}

func TestAnalyze_notifyEnabledNoDeferredWarning(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
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
	if strings.Contains(stderr.String(), "warning: notify.") {
		t.Fatalf("unexpected notify deferred warning: %q", stderr.String())
	}
}

func TestAnalyze_apiWatchdogEnabledNoDeferredWarning(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
  api_watchdog:
    enabled: true
    interval: 60s
    fail_after: 5m
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
	if strings.Contains(stderr.String(), "api_watchdog") {
		t.Fatalf("unexpected api_watchdog deferred warning after v0.8.0 engine wiring: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "Deferred (accepted by schema") {
		t.Fatalf("unexpected Deferred summary for implemented api_watchdog: %q", stdout.String())
	}
}

func TestAnalyze_notifyRequireDeliveryNoDeferredSummary(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
notify:
  require_delivery: true
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
	if strings.Contains(stdout.String(), "Deferred (accepted by schema") {
		t.Fatalf("unexpected Deferred summary for implemented require_delivery: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "warning: notify.require_delivery") {
		t.Fatalf("unexpected deferred warning on stderr for analyze: %q", stderr.String())
	}
}

func TestDown_dryRunRequireDeliveryNoDeferredWarningOnStderr(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
run:
  mode: "dry-run"
notify:
  require_delivery: true
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"down", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(stderr.String(), "warning: notify.require_delivery") {
		t.Fatalf("unexpected deferred warning on stderr for down: stderr=%q stdout=%q", stderr.String(), stdout.String())
	}
}

// TestClientID_e2eAnalyzeAndDownDryRun verifies client.id appears on analyze stdout
// and in engine dry-run log lines (integration across config load → CLI → engine).
func TestClientID_e2eAnalyzeAndDownDryRun(t *testing.T) {
	stubClusterValidationSkipped(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
client:
  id: "kzero-e2e-client"
pipelines:
  down:
    - deployment.ns/widget
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var analyzeOut bytes.Buffer
	analyzeCmd := newRootCmd()
	analyzeCmd.SetOut(&analyzeOut)
	analyzeCmd.SetErr(&analyzeOut)
	analyzeCmd.SetArgs([]string{"analyze", "--config", cfgPath})
	if err := analyzeCmd.Execute(); err != nil {
		t.Fatalf("analyze Execute: %v", err)
	}
	if !strings.Contains(analyzeOut.String(), "Client id: kzero-e2e-client") {
		t.Fatalf("analyze missing Client id line: %q", analyzeOut.String())
	}

	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	var downOut bytes.Buffer
	downCmd := newRootCmd()
	downCmd.SetOut(&downOut)
	downCmd.SetErr(&downOut)
	downCmd.SetArgs([]string{"down", "--config", cfgPath})
	if err := downCmd.Execute(); err != nil {
		t.Fatalf("down Execute: %v", err)
	}
	out := downOut.String()
	if !strings.Contains(out, `client_id=kzero-e2e-client`) {
		t.Fatalf("down dry-run missing client_id in logs: %q", out)
	}
	if !strings.Contains(out, "[dry-run]") {
		t.Fatalf("down dry-run missing [dry-run] prefix: %q", out)
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

func TestTargetCmd_outputSlug(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"target", "--config", cfgPath, "--output", "slug"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "test-cluster" {
		t.Fatalf("slug output = %q, want test-cluster", got)
	}
	if strings.Contains(stderr.String(), "kzero target finished") {
		t.Fatalf("slug output should not emit command summary, stderr: %q", stderr.String())
	}
}

func TestTargetCmd_printsKubernetesTarget(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"target", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"Kubernetes target:",
		"  context: test-context",
		"  api_server: https://test.kubernetes.local",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q; full output: %q", want, out)
		}
	}
}

func TestPipeline_missingConfigFile(t *testing.T) {
	for _, sub := range []string{"down", "up", "reset"} {
		t.Run(sub, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs([]string{sub, "--config", filepath.Join(t.TempDir(), "missing.yaml")})
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected config load error")
			}
		})
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
