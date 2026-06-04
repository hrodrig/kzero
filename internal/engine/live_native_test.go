package engine

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
)

func TestLiveRunner_NativeWorkloadSkipsKubectl(t *testing.T) {
	t.Parallel()

	rep := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	}
	client := fake.NewSimpleClientset(dep)

	var execCalls int
	r := &LiveRunner{
		Workload: executor.NewNative(client),
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			execCalls++
			return nil, nil
		},
	}
	cfg := &config.Config{
		Run:     config.RunConfig{Mode: "live", Execution: "native"},
		Command: config.CommandConfig{Kubectl: "/bin/kubectl"},
	}
	step := config.PipelineStep{
		Ref:       "deployment.ns/app",
		Type:      "deployment",
		Namespace: "ns",
		Name:      "app",
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if execCalls != 0 {
		t.Fatalf("expected no subprocess calls with injected native workload, got %d", execCalls)
	}
	got, err := client.AppsV1().Deployments("ns").Get(context.Background(), "app", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Replicas == nil || *got.Spec.Replicas != 0 {
		t.Fatalf("replicas = %v, want 0", got.Spec.Replicas)
	}
}
