package probe

import (
	"context"
	"strings"
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

func TestRunChecks_podsSchedulable_noNamespaces(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		InfraProbe: config.InfraProbeConfig{
			Checks: []config.ProbeCheck{{PodsSchedulable: true}},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
	err := RunChecks(context.Background(), cfg, factory, false, true)
	if err == nil || !strings.Contains(err.Error(), "no namespaces") {
		t.Fatalf("got %v", err)
	}
}

func TestRunChecks_podsSchedulable_pass(t *testing.T) {
	t.Parallel()
	client := fake.NewSimpleClientset()
	cfg := &config.Config{
		InfraProbe: config.InfraProbeConfig{
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{Ref: "release.ns/app", Type: "release", Namespace: "ns", Name: "app"}},
			},
			Checks: []config.ProbeCheck{{PodsSchedulable: true}},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return client, nil }
	if err := RunChecks(context.Background(), cfg, factory, false, true); err != nil {
		t.Fatalf("RunChecks: %v", err)
	}
}

func TestRunChecks_podsSchedulable_unschedulable(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "stuck", Namespace: "ns"},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: corev1.PodReasonUnschedulable, Message: "no nodes"},
			},
		},
	}
	client := fake.NewSimpleClientset(pod)
	cfg := &config.Config{
		InfraProbe: config.InfraProbeConfig{
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{Ref: "release.ns/app", Type: "release", Namespace: "ns", Name: "app"}},
			},
			Checks: []config.ProbeCheck{{PodsSchedulable: true}},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return client, nil }
	if err := RunChecks(context.Background(), cfg, factory, false, true); err == nil {
		t.Fatal("expected pods_schedulable error")
	}
}
