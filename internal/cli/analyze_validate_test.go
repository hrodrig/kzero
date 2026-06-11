package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/validate"
)

func TestAnalyze_clusterValidationOK(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
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
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
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

func TestAnalyze_releaseSDKPlan(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
helm:
  workspace: ./helm
pipelines:
  up:
    - release.mon/prom:
        chart: oci://example/prom
        version: "1.0.0"
run:
  mode: "dry-run"
  execution: native
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"analyze", "--log-level", "debug", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Run execution: native") {
		t.Fatalf("missing execution line: %q", out)
	}
	if !strings.Contains(out, "helm upgrade --install (sdk)") || !strings.Contains(out, "oci://example/prom") {
		t.Fatalf("missing sdk plan: %q", out)
	}
}

func TestAnalyze_pvcPlanAndValidation(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	client := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-0", Namespace: "db"},
	})
	old := validate.DefaultClientFactory
	validate.DefaultClientFactory = func(string) (kubernetes.Interface, error) { return client, nil }
	t.Cleanup(func() { validate.DefaultClientFactory = old })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - pvc.db/data-0
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
	if !strings.Contains(out, "delete pvc (background propagation") {
		t.Fatalf("missing pvc plan: %q", out)
	}
	if !strings.Contains(out, "OK  pvc.db/data-0") {
		t.Fatalf("missing pvc validation OK: %q", out)
	}
}

func TestAnalyze_execPlanAndValidation(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))
	client := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "postgresql-0", Namespace: "database"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "postgres"}}},
	})
	old := validate.DefaultClientFactory
	validate.DefaultClientFactory = func(string) (kubernetes.Interface, error) { return client, nil }
	t.Cleanup(func() { validate.DefaultClientFactory = old })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - exec.database/postgresql-0:
        container: postgres
        command: ["psql", "-c", "select 1"]
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
	if !strings.Contains(out, "exec database/postgresql-0 container=postgres") {
		t.Fatalf("missing exec plan: %q", out)
	}
	if !strings.Contains(out, "OK  exec.database/postgresql-0") {
		t.Fatalf("missing exec validation OK: %q", out)
	}
}
