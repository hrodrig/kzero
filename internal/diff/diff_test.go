package diff

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
)

func TestRun_upReplicasMatchAndDrift(t *testing.T) {
	t.Parallel()

	rep2 := int32(2)
	rep0 := int32(0)
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "app"},
			Spec:       appsv1.DeploymentSpec{Replicas: &rep2},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "pg", Namespace: "db"},
			Spec:       appsv1.StatefulSetSpec{Replicas: &rep0},
		},
	)
	want := 2
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "deployment.app/api", Type: "deployment", Namespace: "app", Name: "api", Replicas: &want},
				{Ref: "statefulset.db/pg", Type: "statefulset", Namespace: "db", Name: "pg"},
			},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return client, nil }

	lines, err := Run(context.Background(), cfg, engine.PhaseUp, factory)
	if err == nil {
		t.Fatal("expected drift error")
	}
	if len(lines) != 2 || !lines[0].OK || lines[1].OK {
		t.Fatalf("lines=%+v", lines)
	}
	if !strings.Contains(lines[1].Detail, "replicas=1 (live=0)") {
		t.Fatalf("sts detail=%q", lines[1].Detail)
	}
}

func TestRun_downScaleZero(t *testing.T) {
	t.Parallel()

	rep0 := int32(0)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "app"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep0},
	})
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.app/api", Type: "deployment", Namespace: "app", Name: "api"},
			},
		},
	}
	lines, err := Run(context.Background(), cfg, engine.PhaseDown, func(string) (kubernetes.Interface, error) {
		return client, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !lines[0].OK {
		t.Fatalf("lines=%+v", lines)
	}
}

func TestRun_pvcAlwaysAbsent(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-0", Namespace: "db"},
	})
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "pvc.db/data-0", Type: "pvc", Namespace: "db", Name: "data-0"},
			},
		},
	}
	lines, err := Run(context.Background(), cfg, engine.PhaseDown, func(string) (kubernetes.Interface, error) {
		return client, nil
	})
	if err == nil {
		t.Fatal("expected drift when pvc still present")
	}
	if lines[0].OK || !strings.Contains(lines[0].Detail, "expected absent") {
		t.Fatalf("line=%+v", lines[0])
	}

	empty := fake.NewSimpleClientset()
	lines, err = Run(context.Background(), cfg, engine.PhaseDown, func(string) (kubernetes.Interface, error) {
		return empty, nil
	})
	if err != nil || !lines[0].OK {
		t.Fatalf("err=%v line=%+v", err, lines)
	}
}

func TestRun_cronjobSuspend(t *testing.T) {
	t.Parallel()

	suspend := true
	client := fake.NewSimpleClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "batch"},
		Spec:       batchv1.CronJobSpec{Suspend: &suspend},
	})
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "cronjob.batch/nightly", Type: "cronjob", Namespace: "batch", Name: "nightly"},
			},
			Up: []config.PipelineStep{
				{Ref: "cronjob.batch/nightly", Type: "cronjob", Namespace: "batch", Name: "nightly"},
			},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return client, nil }

	lines, err := Run(context.Background(), cfg, engine.PhaseDown, factory)
	if err != nil || !lines[0].OK {
		t.Fatalf("down err=%v line=%+v", err, lines)
	}
	lines, err = Run(context.Background(), cfg, engine.PhaseUp, factory)
	if err == nil || lines[0].OK {
		t.Fatalf("up should drift, err=%v line=%+v", err, lines)
	}
}

func TestRun_jobAndReleasePresence(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "migrate", Namespace: "batch"}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sh.helm.release.v1.prom.v1",
				Namespace: "mon",
				Labels:    map[string]string{"owner": "helm", "name": "prom"},
			},
		},
	)
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "job.batch/migrate", Type: "job", Namespace: "batch", Name: "migrate"},
				{Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom"},
			},
			Down: []config.PipelineStep{
				{Ref: "job.batch/migrate", Type: "job", Namespace: "batch", Name: "migrate"},
				{Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom"},
			},
		},
	}
	factory := func(string) (kubernetes.Interface, error) { return client, nil }

	lines, err := Run(context.Background(), cfg, engine.PhaseUp, factory)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || !lines[0].OK || !lines[1].OK {
		t.Fatalf("up lines=%+v", lines)
	}

	lines, err = Run(context.Background(), cfg, engine.PhaseDown, factory)
	if err == nil {
		t.Fatal("expected down drift while resources present")
	}
	if lines[0].OK || lines[1].OK {
		t.Fatalf("down lines=%+v", lines)
	}
}

func TestRun_skipsExecAndCustom(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Custom: "./hooks/x.sh"},
				{Ref: "exec.ns/pod", Type: "exec", Namespace: "ns", Name: "pod", Container: "c", Command: []string{"true"}},
			},
		},
	}
	lines, err := Run(context.Background(), cfg, engine.PhaseUp, func(string) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || !lines[0].Skip || !lines[1].Skip {
		t.Fatalf("lines=%+v", lines)
	}
}

func TestRun_kubeconfigError(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), &config.Config{}, engine.PhaseUp, func(string) (kubernetes.Interface, error) {
		return nil, errors.New("no kube")
	})
	if err == nil || !strings.Contains(err.Error(), "cannot load kubeconfig") {
		t.Fatalf("got %v", err)
	}
}

func TestRun_releaseMissingOnUp(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom"},
			},
		},
	}
	lines, err := Run(context.Background(), cfg, engine.PhaseUp, func(string) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	})
	if err == nil || lines[0].OK {
		t.Fatalf("err=%v line=%+v", err, lines)
	}
	if !strings.Contains(lines[0].Detail, "expected present") {
		t.Fatalf("detail=%q", lines[0].Detail)
	}
}

func TestPrint_okAndDrift(t *testing.T) {
	t.Parallel()

	rep := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "app"},
		Spec:       appsv1.DeploymentSpec{Replicas: &rep},
	})
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "deployment.app/api", Type: "deployment", Namespace: "app", Name: "api"},
			},
		},
	}
	var buf, errBuf strings.Builder
	err := Print(&buf, &errBuf, cfg, engine.PhaseUp, func(string) (kubernetes.Interface, error) {
		return client, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Diff (phase=up)") || !strings.Contains(buf.String(), "OK") {
		t.Fatalf("out=%q", buf.String())
	}

	// Drift path (deployment missing)
	err = Print(&buf, &errBuf, cfg, engine.PhaseUp, func(string) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	})
	if err == nil {
		t.Fatal("expected drift")
	}
	if !strings.Contains(buf.String(), "DRIFT") {
		t.Fatalf("out=%q", buf.String())
	}
}

func TestRun_nilConfig(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), nil, engine.PhaseUp, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestStepsForPhase(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "a"}},
			Up:   []config.PipelineStep{{Ref: "b"}},
		},
	}
	if got := stepsForPhase(cfg, engine.PhaseDown); len(got) != 1 || got[0].Ref != "a" {
		t.Fatalf("%+v", got)
	}
	if got := stepsForPhase(cfg, engine.Phase("")); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestFriendlyGet(t *testing.T) {
	t.Parallel()

	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	if got := friendlyGet("deployment", "ns", "x", apierrors.NewNotFound(gr, "x")); !strings.Contains(got, "not found") {
		t.Fatalf("got %q", got)
	}
	if got := friendlyGet("deployment", "ns", "x", apierrors.NewForbidden(gr, "x", errors.New("no"))); !strings.Contains(got, "forbidden") {
		t.Fatalf("got %q", got)
	}
}
