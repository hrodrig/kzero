package probe

import (
	"context"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunChecks_pvcBound(t *testing.T) {
	t.Parallel()
	client := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc", Namespace: "ns"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	})
	cfg := &config.Config{
		InfraProbe: config.InfraProbeConfig{
			Checks: []config.ProbeCheck{{PVCBound: "ns/pvc"}},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return client, nil }
	if err := RunChecks(context.Background(), cfg, factory, false, true); err != nil {
		t.Fatalf("RunChecks: %v", err)
	}
}

func TestRunChecks_releaseReadyRequiresUp(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		InfraProbe: config.InfraProbeConfig{
			Checks: []config.ProbeCheck{{ReleaseReady: true}},
		},
	}
	err := RunChecks(context.Background(), cfg, nil, false, false)
	if err == nil {
		t.Fatal("expected release_ready error when up failed")
	}
}

func TestRunChecks_dryRunSkipsAPI(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		InfraProbe: config.InfraProbeConfig{
			Checks: []config.ProbeCheck{{PVCBound: "ns/missing"}},
		},
	}
	if err := RunChecks(context.Background(), cfg, nil, true, true); err != nil {
		t.Fatalf("dry-run should not call API: %v", err)
	}
}
