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

	"github.com/hrodrig/kzero/internal/validate"
)

func TestAnalyze_clusterValidationOK(t *testing.T) {
	rep := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	})
	old := validate.DefaultClientFactory
	validate.DefaultClientFactory = func(string) (kubernetes.Interface, error) { return client, nil }
	t.Cleanup(func() { validate.DefaultClientFactory = old })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: "dry-run"
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
	if !strings.Contains(stdout.String(), "Cluster validation:") {
		t.Fatalf("missing cluster validation section: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "OK  deployment.ns/app") {
		t.Fatalf("missing OK line: %q", stdout.String())
	}
}

func TestAnalyze_clusterValidationFailsExit(t *testing.T) {
	client := fake.NewSimpleClientset()
	old := validate.DefaultClientFactory
	validate.DefaultClientFactory = func(string) (kubernetes.Interface, error) { return client, nil }
	t.Cleanup(func() { validate.DefaultClientFactory = old })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/ghost
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
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected analyze to fail cluster validation")
	}
	if !strings.Contains(stdout.String(), "FAIL  deployment.ns/ghost") {
		t.Fatalf("stdout: %q", stdout.String())
	}
}
