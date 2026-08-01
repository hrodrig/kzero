package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDescribeStep_customAndRef(t *testing.T) {
	t.Parallel()

	custom := config.PipelineStep{Custom: "./hooks/x.sh"}
	if got := DescribeStep(custom); got != "custom: ./hooks/x.sh" {
		t.Fatalf("custom: got %q", got)
	}

	ref := config.PipelineStep{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"}
	if got := DescribeStep(ref); got != "deployment.ns/app" {
		t.Fatalf("ref: got %q", got)
	}
}

func TestFormatStepPlanLine_releaseAndOptions(t *testing.T) {
	t.Parallel()

	shellCfg := &config.Config{Run: config.RunConfig{Execution: "shell"}}
	nativeCfg := &config.Config{Run: config.RunConfig{Execution: "native"}}

	three := 3
	step := config.PipelineStep{
		Ref:          "statefulset.db/pg",
		Type:         "statefulset",
		Namespace:    "db",
		Name:         "pg",
		PreStep:      "./hooks/pre.sh",
		Replicas:     &three,
		WaitForReady: true,
		Timeout:      2 * time.Minute,
	}
	got := FormatStepPlanLine(shellCfg, step, "./helm", "up")
	if !strings.Contains(got, "statefulset.db/pg") {
		t.Fatalf("missing ref: %q", got)
	}
	if !strings.Contains(got, "pre: ./hooks/pre.sh") || !strings.Contains(got, "replicas: 3") {
		t.Fatalf("missing options: %q", got)
	}

	release := config.PipelineStep{
		Ref:       "release.mon/prom",
		Type:      "release",
		Namespace: "mon",
		Name:      "prom",
	}
	got = FormatStepPlanLine(shellCfg, release, "./helm-assets", "up")
	if !strings.Contains(got, "script: helm-assets/prom.sh") {
		t.Fatalf("missing script path: %q", got)
	}

	releaseSDK := config.PipelineStep{
		Ref:       "release.mon/prom",
		Type:      "release",
		Namespace: "mon",
		Name:      "prom",
		Chart:     "oci://example/prom",
		Version:   "1.0.0",
	}
	got = FormatStepPlanLine(nativeCfg, releaseSDK, "./helm-assets", "up")
	if !strings.Contains(got, "helm upgrade --install (sdk)") || !strings.Contains(got, "oci://example/prom") {
		t.Fatalf("missing sdk plan: %q", got)
	}

	got = FormatStepPlanLine(shellCfg, release, "./helm-assets", "down")
	if !strings.Contains(got, "helm uninstall") {
		t.Fatalf("missing uninstall hint: %q", got)
	}

	got = FormatStepPlanLine(nativeCfg, release, "./helm-assets", "down")
	if !strings.Contains(got, "helm sdk uninstall") {
		t.Fatalf("missing sdk uninstall hint: %q", got)
	}

	releaseNoChart := config.PipelineStep{
		Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom",
	}
	got = FormatStepPlanLine(nativeCfg, releaseNoChart, "./helm-assets", "up")
	if !strings.Contains(got, "manifest: helm-assets/prom.yaml") {
		t.Fatalf("missing manifest hint: %q", got)
	}

	pvc := config.PipelineStep{
		Ref: "pvc.database/data-postgresql-0", Type: "pvc",
		Namespace: "database", Name: "data-postgresql-0",
	}
	got = FormatStepPlanLine(shellCfg, pvc, "", "down")
	if !strings.Contains(got, "delete pvc (background propagation") {
		t.Fatalf("missing pvc delete hint: %q", got)
	}

	execStep := config.PipelineStep{
		Ref: "exec.database/postgresql-0", Type: "exec",
		Namespace: "database", Name: "postgresql-0",
		Container: "postgres", Command: []string{"psql", "-c", "select 1"},
	}
	got = FormatStepPlanLine(shellCfg, execStep, "", "down")
	if !strings.Contains(got, "exec database/postgresql-0 container=postgres") {
		t.Fatalf("missing exec hint: %q", got)
	}

	cj := config.PipelineStep{Ref: "cronjob.batch/nightly", Type: "cronjob", Namespace: "batch", Name: "nightly"}
	got = FormatStepPlanLine(shellCfg, cj, "", "down")
	if !strings.Contains(got, "suspend=true") {
		t.Fatalf("missing cronjob hint: %q", got)
	}
	job := config.PipelineStep{
		Ref: "job.batch/migrate", Type: "job", Namespace: "batch", Name: "migrate",
		Manifest: "./jobs/m.yaml",
	}
	got = FormatStepPlanLine(shellCfg, job, "", "up")
	if !strings.Contains(got, "create job from manifest ./jobs/m.yaml") {
		t.Fatalf("missing job hint: %q", got)
	}
}
