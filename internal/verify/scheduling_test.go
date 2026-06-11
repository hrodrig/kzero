package verify

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/config"
)

func TestRun_podsSchedulable_pass(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
			},
		},
	}
	client := fake.NewSimpleClientset(pod)
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"}},
		},
		Verify: config.VerifyConfig{Checks: []string{CheckPodsSchedulable}},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Failed(report) {
		t.Fatalf("%+v", report)
	}
}

func TestRun_podsSchedulable_noNamespaces(t *testing.T) {
	t.Parallel()
	client := fake.NewSimpleClientset()
	cfg := &config.Config{
		Verify: config.VerifyConfig{Checks: []string{CheckPodsSchedulable}},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Failed(report) {
		t.Fatalf("%+v", report)
	}
}

func TestNamespacesFromSteps_dedupes(t *testing.T) {
	t.Parallel()
	steps := []config.PipelineStep{
		{Namespace: "a"}, {Namespace: "a"}, {Namespace: "b"},
	}
	got := NamespacesFromSteps(steps)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}

func TestRun_podsSchedulable_unschedulable(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "stuck", Namespace: "ns"},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:    corev1.PodScheduled,
					Status:  corev1.ConditionFalse,
					Reason:  corev1.PodReasonUnschedulable,
					Message: "0/1 nodes available: node(s) had untolerated taint",
				},
			},
		},
	}
	client := fake.NewSimpleClientset(pod)
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{{Ref: "release.ns/stuck", Type: "release", Namespace: "ns", Name: "stuck"}},
		},
		Verify: config.VerifyConfig{Checks: []string{CheckPodsSchedulable}},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err == nil {
		t.Fatal("expected error")
	}
	if !Failed(report) {
		t.Fatalf("%+v", report)
	}
}
