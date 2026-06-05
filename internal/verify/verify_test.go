package verify

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
)

func TestRun_workloadsReady_pass(t *testing.T) {
	t.Parallel()
	rep := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns", Generation: 1},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			Replicas:           1,
			UpdatedReplicas:    1,
			ReadyReplicas:      1,
		},
	}
	client := fake.NewSimpleClientset(dep)
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
		Verify: config.VerifyConfig{Checks: []string{CheckWorkloadsReady}},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Failed(report) {
		t.Fatalf("report: %+v", report)
	}
}

func TestRun_workloadsReady_notReady(t *testing.T) {
	t.Parallel()
	rep := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns", Generation: 1},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			ReadyReplicas:      0,
		},
	}
	client := fake.NewSimpleClientset(dep)
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err == nil {
		t.Fatal("expected error")
	}
	if !Failed(report) {
		t.Fatalf("report: %+v", report)
	}
}

func TestRun_nodesReady(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "n1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
	client := fake.NewSimpleClientset(node)
	cfg := &config.Config{Verify: config.VerifyConfig{Checks: []string{CheckNodesReady}}}
	report, err := Run(context.Background(), cfg, factory(client))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Failed(report) {
		t.Fatalf("%+v", report)
	}
}

func TestRun_nodesNotReady(t *testing.T) {
	t.Parallel()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "n1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
			},
		},
	}
	client := fake.NewSimpleClientset(node)
	cfg := &config.Config{Verify: config.VerifyConfig{Checks: []string{CheckNodesReady}}}
	report, err := Run(context.Background(), cfg, factory(client))
	if err == nil {
		t.Fatal("expected error")
	}
	if !Failed(report) {
		t.Fatalf("%+v", report)
	}
}

func TestRun_deploymentNotFound(t *testing.T) {
	t.Parallel()
	client := fake.NewSimpleClientset()
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "deployment.ns/missing", Type: "deployment", Namespace: "ns", Name: "missing"},
			},
		},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err == nil {
		t.Fatal("expected error")
	}
	if !Failed(report) {
		t.Fatalf("%+v", report)
	}
}

func TestPrint_text(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	r := Report{
		Outcome: OutcomeFailed,
		Checks: []CheckResult{{
			Name:  CheckWorkloadsReady,
			OK:    false,
			Items: []Item{{Ref: "deployment.ns/app", OK: false, Detail: "0/1 ready"}},
		}},
	}
	if err := Print(&buf, log.FormatText, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "outcome: failed") {
		t.Fatalf("%q", buf.String())
	}
}

func TestPrint_json(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	r := Report{Outcome: OutcomeReady, Checks: []CheckResult{{Name: CheckNodesReady, OK: true}}}
	if err := Print(&buf, log.FormatJSON, r); err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(strings.TrimSpace(buf.String()))) {
		t.Fatalf("%q", buf.String())
	}
}

func TestRun_statefulSetReady(t *testing.T) {
	t.Parallel()
	three := 3
	rep := int32(3)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "data", Generation: 1},
		Spec:       appsv1.StatefulSetSpec{Replicas: &rep},
		Status: appsv1.StatefulSetStatus{
			ObservedGeneration: 1,
			UpdatedReplicas:    3,
			ReadyReplicas:      3,
		},
	}
	client := fake.NewSimpleClientset(sts)
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "statefulset.data/db", Type: "statefulset", Namespace: "data", Name: "db", Replicas: &three},
			},
		},
		Verify: config.VerifyConfig{Checks: []string{CheckWorkloadsReady}},
	}
	report, err := Run(context.Background(), cfg, factory(client))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if Failed(report) {
		t.Fatalf("%+v", report)
	}
}

func TestRun_unknownCheck(t *testing.T) {
	t.Parallel()
	client := fake.NewSimpleClientset()
	cfg := &config.Config{Verify: config.VerifyConfig{Checks: []string{"unknown"}}}
	_, err := Run(context.Background(), cfg, factory(client))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestErrorMessage(t *testing.T) {
	t.Parallel()
	msg := ErrorMessage(Report{Outcome: OutcomeFailed, Checks: []CheckResult{{Name: CheckNodesReady, OK: false}}})
	if msg == "" {
		t.Fatal("expected message")
	}
}

func factory(c kubernetes.Interface) func(string) (kubernetes.Interface, error) {
	return func(string) (kubernetes.Interface, error) { return c, nil }
}
