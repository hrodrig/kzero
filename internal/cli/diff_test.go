package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/exitcode"
	"github.com/hrodrig/kzero/internal/validate"
)

func TestDiffCmd_matchOK(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	rep := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	})
	old := validate.SwapDefaultClientFactory(func(string) (kubernetes.Interface, error) { return client, nil })
	t.Cleanup(func() { validate.SwapDefaultClientFactory(old) })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  up:
    - deployment.ns/app
run:
  mode: dry-run
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"diff", "--config", cfgPath, "--phase", "up"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("diff: %v\n%s\n%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Diff (phase=up)") || !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestDiffCmd_invalidPhase(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "kzero.yaml")
	body := `
schema_version: "1.0"
pipelines:
  up:
    - deployment.default/app
run:
  mode: dry-run
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"diff", "--config", cfgPath, "--phase", "sideways"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid phase")
	}
	if got := exitcode.Of(err); got != exitcode.ConfigError {
		t.Fatalf("exit=%d want ConfigError, err=%v", got, err)
	}
	if !strings.Contains(err.Error(), "up or down") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseDiffPhase(t *testing.T) {
	t.Parallel()

	p, err := parseDiffPhase("UP")
	if err != nil || p != engine.PhaseUp {
		t.Fatalf("up: %v %q", err, p)
	}
	p, err = parseDiffPhase("down")
	if err != nil || p != engine.PhaseDown {
		t.Fatalf("down: %v %q", err, p)
	}
	if _, err := parseDiffPhase("both"); err == nil {
		t.Fatal("expected error")
	}
}
