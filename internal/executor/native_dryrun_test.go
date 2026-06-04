package executor

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestNative_ScaleDeployment_serverSideDryRunSendsDryRunAll(t *testing.T) {
	t.Parallel()

	rep := int32(2)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	}
	client := fake.NewSimpleClientset(dep)

	n := NewNative(client, true)
	if err := n.Scale(context.Background(), "deployment", "ns", "app", 0); err != nil {
		t.Fatal(err)
	}
	if !updateActionHasDryRunAll(client.Actions(), "deployments") {
		t.Fatal("expected Update with DryRun=All on deployments")
	}
}

func TestNative_ScaleDeployment_liveUpdateNoDryRun(t *testing.T) {
	t.Parallel()

	rep := int32(2)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	}
	client := fake.NewSimpleClientset(dep)

	n := NewNative(client, false)
	if err := n.Scale(context.Background(), "deployment", "ns", "app", 0); err != nil {
		t.Fatal(err)
	}
	if updateActionHasDryRunAll(client.Actions(), "deployments") {
		t.Fatal("live scale must not send DryRun=All")
	}
	got, err := client.AppsV1().Deployments("ns").Get(context.Background(), "app", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Replicas == nil || *got.Spec.Replicas != 0 {
		t.Fatalf("replicas = %v, want 0", got.Spec.Replicas)
	}
}

func updateActionHasDryRunAll(actions []ktesting.Action, resource string) bool {
	for _, a := range actions {
		if a.GetVerb() != "update" || a.GetResource().Resource != resource {
			continue
		}
		ua, ok := a.(ktesting.UpdateActionImpl)
		if !ok {
			continue
		}
		for _, d := range ua.GetUpdateOptions().DryRun {
			if d == metav1.DryRunAll {
				return true
			}
		}
	}
	return false
}
