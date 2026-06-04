package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
)

func TestDryRunner_NativeServerSideDryRun(t *testing.T) {
	t.Parallel()

	rep := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	}
	client := fake.NewSimpleClientset(dep)

	var buf bytes.Buffer
	r := &DryRunner{
		Out:      &buf,
		nativeWL: executor.NewNative(client, true),
	}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run", Execution: "native"},
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
	out := buf.String()
	if !strings.Contains(out, "server-side dry-run ok") {
		t.Fatalf("expected server-side dry-run message, got: %q", out)
	}
	if strings.Contains(out, "pipeline down step 0:") {
		t.Fatalf("expected native dry-run path, not plan-only line: %q", out)
	}

}

func TestDryRunner_ShellExecutionPlanOnly(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run", Execution: "shell"},
	}
	r := NewDryRunner(cfg, &buf)
	step := config.PipelineStep{
		Ref:       "deployment.ns/app",
		Type:      "deployment",
		Namespace: "ns",
		Name:      "app",
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[dry-run] pipeline down step 0:") {
		t.Fatalf("expected plan-only output, got: %q", out)
	}
	if strings.Contains(out, "server-side dry-run ok") {
		t.Fatalf("unexpected native dry-run: %q", out)
	}
}
