package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureNamespace_createsWhenMissing(t *testing.T) {

	client := fake.NewClientset()
	prev := kubeClientForHelm
	kubeClientForHelm = func(string) (kubernetes.Interface, error) {
		return client, nil
	}
	t.Cleanup(func() { kubeClientForHelm = prev })

	if err := ensureNamespace(context.Background(), &config.Config{}, "probe-ns"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "probe-ns", metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureNamespace_existing(t *testing.T) {

	client := fake.NewClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "exists"}})
	prev := kubeClientForHelm
	kubeClientForHelm = func(string) (kubernetes.Interface, error) {
		return client, nil
	}
	t.Cleanup(func() { kubeClientForHelm = prev })

	if err := ensureNamespace(context.Background(), &config.Config{}, "exists"); err != nil {
		t.Fatal(err)
	}
}

func TestResolveChartRef(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Helm: config.HelmConfig{Workspace: "/ws"}}
	if got := resolveChartRef(cfg, "oci://x/c"); got != "oci://x/c" {
		t.Fatalf("oci: %q", got)
	}
	if got := resolveChartRef(cfg, "charts/app"); got != "/ws/charts/app" {
		t.Fatalf("relative: %q", got)
	}
}

func TestNewSDKHelm(t *testing.T) {
	t.Parallel()

	h, err := NewSDKHelm(&config.Config{Run: config.RunConfig{Kubeconfig: "/tmp/k"}})
	if err != nil || !h.UsesSDK() {
		t.Fatalf("sdk helm: %v uses=%v", err, h.UsesSDK())
	}
	if _, err := NewSDKHelm(nil); err == nil {
		t.Fatal("expected nil config error")
	}
}

func TestSDKHelm_opTimeout(t *testing.T) {
	t.Parallel()

	h := &SDKHelm{cfg: &config.Config{Run: config.RunConfig{OperationTimeout: 0}}}
	if got := h.opTimeout(config.PipelineStep{}); got != 5*time.Minute {
		t.Fatalf("timeout: %s", got)
	}
}

func TestSDKHelm_UpgradeInstall_missingChart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h, err := NewSDKHelm(&config.Config{
		Run:  config.RunConfig{Execution: "native"},
		Helm: config.HelmConfig{Workspace: dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	step := config.PipelineStep{Name: "prom", Namespace: "mon", Type: "release", Ref: "release.mon/prom"}
	if err := h.UpgradeInstall(context.Background(), step); err == nil {
		t.Fatal("expected missing chart error")
	}
}

func TestSDKHelm_UpgradeInstall_createNamespaceThenFailsChart(t *testing.T) {

	client := fake.NewClientset()
	prev := kubeClientForHelm
	kubeClientForHelm = func(string) (kubernetes.Interface, error) {
		return client, nil
	}
	t.Cleanup(func() { kubeClientForHelm = prev })

	dir := t.TempDir()
	manifest := `chart: oci://example.invalid/chart
create_namespace: true
`
	if err := os.WriteFile(filepath.Join(dir, "prom.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := NewSDKHelm(&config.Config{
		Run:  config.RunConfig{Execution: "native"},
		Helm: config.HelmConfig{Workspace: dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	step := config.PipelineStep{Name: "prom", Namespace: "probe-ns", Type: "release", Ref: "release.probe-ns/prom"}
	if err := h.UpgradeInstall(context.Background(), step); err == nil {
		t.Fatal("expected chart pull error")
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "probe-ns", metav1.GetOptions{}); err != nil {
		t.Fatalf("namespace should be created before chart error: %v", err)
	}
}

func TestFormatChartPlan_containsFields(t *testing.T) {
	t.Parallel()
	got := FormatChartPlan(ChartSpec{Chart: "oci://x/chart", Version: "1.0", Wait: true, CreateNamespace: true})
	for _, want := range []string{"sdk", "oci://x/chart", "1.0", "wait", "create-namespace"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}
