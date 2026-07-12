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
	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/validate"
)

func TestShouldAutoVerify(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{Verify: true, Mode: "live"}}
	if !shouldAutoVerify(cfg, "up") || !shouldAutoVerify(cfg, "reset") {
		t.Fatal("expected auto verify on up/reset in live")
	}
	if shouldAutoVerify(cfg, "down") {
		t.Fatal("down should not auto verify")
	}
	cfg.Run.Mode = "dry-run"
	if shouldAutoVerify(cfg, "up") {
		t.Fatal("dry-run should not auto verify")
	}
}

func TestVerify_readyDeployment(t *testing.T) {
	t.Setenv("KUBECONFIG", cluster.TestKubeconfigPath(t))

	rep := int32(1)
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns", Generation: 1},
			Spec:       appsv1.DeploymentSpec{Replicas: &rep},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 1,
				UpdatedReplicas:    1,
				ReadyReplicas:      1,
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "n1"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			},
		},
	)
	old := validate.SwapDefaultClientFactory(func(string) (kubernetes.Interface, error) { return client, nil })
	t.Cleanup(func() { validate.SwapDefaultClientFactory(old) })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up:
    - deployment.ns/app:
        replicas: 1
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"verify", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "outcome: ready") {
		t.Fatalf("stdout: %q", stdout.String())
	}
}
