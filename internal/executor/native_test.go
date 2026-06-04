package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNative_ScaleDeployment(t *testing.T) {
	t.Parallel()

	rep := int32(2)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	}
	client := fake.NewSimpleClientset(dep)

	n := NewNative(client)
	if err := n.Scale(context.Background(), "deployment", "ns", "app", 0); err != nil {
		t.Fatal(err)
	}
	got, err := client.AppsV1().Deployments("ns").Get(context.Background(), "app", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Replicas == nil || *got.Spec.Replicas != 0 {
		t.Fatalf("replicas = %v, want 0", got.Spec.Replicas)
	}
}

func TestNative_ScaleDeploymentNotFound(t *testing.T) {
	t.Parallel()

	n := NewNative(fake.NewSimpleClientset())
	err := n.Scale(context.Background(), "deployment", "ns", "missing", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNative_WaitRollout_zeroReplicas(t *testing.T) {
	t.Parallel()

	rep := int32(0)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns", Generation: 1},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			UpdatedReplicas:    0,
			ReadyReplicas:      0,
		},
	}
	client := fake.NewSimpleClientset(dep)
	n := NewNative(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := n.WaitRollout(ctx, "deployment", "ns", "app", time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestNative_UnsupportedKind(t *testing.T) {
	t.Parallel()

	n := NewNative(fake.NewSimpleClientset())
	err := n.Scale(context.Background(), "daemonset", "ns", "x", 0)
	if err == nil || !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}
