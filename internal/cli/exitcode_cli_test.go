package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/exitcode"
	"github.com/hrodrig/kzero/internal/validate"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// CLI exit taxonomy (#42): representative failures through cobra Execute → exitcode.Of.

func TestCLI_exitCode_badConfig_isConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - "not-a-valid-step"
run:
  mode: dry-run
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"analyze", "--config", cfgPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected config error")
	}
	if got := exitcode.Of(err); got != exitcode.ConfigError {
		t.Fatalf("exit=%d want %d (config)", got, exitcode.ConfigError)
	}
}

func TestCLI_exitCode_clusterValidationFail_isKubernetes(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	client := fake.NewSimpleClientset()
	old := validate.SwapDefaultClientFactory(func(string) (kubernetes.Interface, error) { return client, nil })
	t.Cleanup(func() { validate.SwapDefaultClientFactory(old) })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/ghost
run:
  mode: dry-run
  execution: shell
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"analyze", "--config", cfgPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected cluster validation failure")
	}
	if got := exitcode.Of(err); got != exitcode.KubernetesError {
		t.Fatalf("exit=%d want %d (kubernetes)", got, exitcode.KubernetesError)
	}
}

func TestCLI_exitCode_doctorFail_isKubernetes(t *testing.T) {
	client := fake.NewSimpleClientset() // no deployment → workload error
	old := validate.SwapDefaultClientFactory(func(string) (kubernetes.Interface, error) { return client, nil })
	t.Cleanup(func() { validate.SwapDefaultClientFactory(old) })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: dry-run
  execution: native
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--config", cfgPath, "--output", "json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected doctor failure")
	}
	if got := exitcode.Of(err); got != exitcode.KubernetesError {
		t.Fatalf("exit=%d want %d (kubernetes)", got, exitcode.KubernetesError)
	}
}

func TestCLI_exitCode_nativeDryRunAPIFail_isExecutor(t *testing.T) {
	kc := cluster.TestKubeconfigPath(t)
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: dry-run
  execution: native
  kubeconfig: "`+kc+`"
  color: never
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"down", "--config", cfgPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected native dry-run scale to fail against unreachable API")
	}
	if got := exitcode.Of(err); got != exitcode.ExecutorAborted {
		t.Fatalf("exit=%d want %d (executor); err=%v", got, exitcode.ExecutorAborted, err)
	}
}

func TestCLI_exitCode_notifyTestPOSTFail_isNotify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: dry-run
  execution: shell
notify:
  webhook:
    enabled: true
    url: "`+srv.URL+`"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"notify", "test", "--config", cfgPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected notify test POST failure")
	}
	if got := exitcode.Of(err); got != exitcode.NotifyFailed {
		t.Fatalf("exit=%d want %d (notify); err=%v", got, exitcode.NotifyFailed, err)
	}
}

func TestCLI_exitCode_notifyTestNoChannel_isConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
run:
  mode: dry-run
  execution: shell
notify:
  slack:
    enabled: false
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"notify", "test", "--config", cfgPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no channel enabled")
	}
	if got := exitcode.Of(err); got != exitcode.ConfigError {
		t.Fatalf("exit=%d want %d (config)", got, exitcode.ConfigError)
	}
}

func TestCLI_exitCode_completionBadShell_isConfig(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"completion", "csh"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unsupported shell error")
	}
	if got := exitcode.Of(err); got != exitcode.ConfigError {
		t.Fatalf("exit=%d want %d (config)", got, exitcode.ConfigError)
	}
}
