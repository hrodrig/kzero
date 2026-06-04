package validate

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/config"
)

func TestCheckPipelineWorkloads_foundAndMissing(t *testing.T) {
	t.Parallel()

	rep := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	})
	factory := func(string) (kubernetes.Interface, error) { return client, nil }

	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"},
				{Ref: "deployment.ns/missing", Type: "deployment", Namespace: "ns", Name: "missing"},
			},
		},
	}

	lines, skipped, err := CheckPipelineWorkloads(context.Background(), cfg, factory)
	if skipped != "" {
		t.Fatalf("expected no skip, got %q", skipped)
	}
	if err == nil {
		t.Fatal("expected error for missing deployment")
	}
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if !lines[0].OK || lines[1].OK {
		t.Fatalf("expected first OK second FAIL, got %+v", lines)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckPipelineWorkloads_skipsWhenNoClient(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}
	_, skipped, err := CheckPipelineWorkloads(context.Background(), cfg, func(string) (kubernetes.Interface, error) {
		return nil, errors.New("no kubeconfig")
	})
	if err != nil {
		t.Fatal(err)
	}
	if skipped == "" {
		t.Fatal("expected skip reason")
	}
}

func TestCheckPipelineWorkloads_dedupesRefs(t *testing.T) {
	t.Parallel()

	rep := int32(2)
	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "cache", Namespace: "db"},
		Spec:       appsv1.StatefulSetSpec{Replicas: &rep},
	})
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Type: "statefulset", Namespace: "db", Name: "cache"}},
			Up:   []config.PipelineStep{{Type: "statefulset", Namespace: "db", Name: "cache"}},
		},
	}
	lines, skipped, err := CheckPipelineWorkloads(context.Background(), cfg, func(string) (kubernetes.Interface, error) {
		return client, nil
	})
	if skipped != "" || err != nil {
		t.Fatalf("skip=%q err=%v", skipped, err)
	}
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1 deduped ref", len(lines))
	}
}

func TestPrintClusterValidation_okAndSkip(t *testing.T) {
	t.Parallel()

	rep := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	})
	factory := func(string) (kubernetes.Interface, error) { return client, nil }
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Type: "deployment", Namespace: "ns", Name: "app"}},
		},
	}

	var out, errOut strings.Builder
	if err := PrintClusterValidation(&out, &errOut, cfg, factory); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Cluster validation:") || !strings.Contains(out.String(), "OK") {
		t.Fatalf("stdout: %q", out.String())
	}

	cfg2 := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Type: "deployment", Namespace: "ns", Name: "app"}},
		},
	}
	errOut.Reset()
	if err := PrintClusterValidation(&out, &errOut, cfg2, func(string) (kubernetes.Interface, error) {
		return nil, errors.New("no client")
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "cluster validation skipped") {
		t.Fatalf("stderr: %q", errOut.String())
	}
}

func TestCheckPipelineWorkloads_statefulSetNoReplicas(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "sts", Namespace: "ns"},
		Spec:       appsv1.StatefulSetSpec{},
	})
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Type: "statefulset", Namespace: "ns", Name: "sts"}},
		},
	}
	_, _, err := CheckPipelineWorkloads(context.Background(), cfg, func(string) (kubernetes.Interface, error) {
		return client, nil
	})
	if err == nil || !strings.Contains(err.Error(), "replicas unset") {
		t.Fatalf("got %v", err)
	}
}
