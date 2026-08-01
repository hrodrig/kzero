package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDryRunner_CustomScriptIsPlannedOnly(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "shell"}}
	r := &DryRunner{Log: testEmitter(&buf)}
	step := config.PipelineStep{Custom: "./hooks/x.sh"}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[dry-run]") || !strings.Contains(out, "./hooks/x.sh") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestDryRunner_LogIncludesClientID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Config{
		Run:    config.RunConfig{Mode: "dry-run", Execution: "shell"},
		Client: config.ClientConfig{ID: "lab"},
	}
	r := &DryRunner{Log: testEmitter(&buf)}
	step := config.PipelineStep{Custom: "./hooks/x.sh"}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "client_id=lab") {
		t.Fatalf("expected client_id in logs, got %q", buf.String())
	}
}

func TestDryRunner_RunHook_skipsEmptyAndHonoursCancel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "shell"}}

	if err := r.RunHook(context.Background(), cfg, "pre", ""); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("empty hook should not log, got %q", buf.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.RunHook(ctx, cfg, "pre", "./hooks/x.sh"); err == nil {
		t.Fatal("expected cancelled hook error")
	}
}

func TestDryRunner_ReleaseDownPlansHelmUninstall(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "shell"}}
	step := config.PipelineStep{
		Ref:       "release.monitoring/kube-prometheus-stack",
		Type:      "release",
		Namespace: "monitoring",
		Name:      "kube-prometheus-stack",
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "helm uninstall monitoring/kube-prometheus-stack") {
		t.Fatalf("expected helm uninstall plan, got: %q", out)
	}
}

func TestDryRunner_ReleaseUpSDKPlan(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{
		Run:  config.RunConfig{Mode: "dry-run", Execution: "native"},
		Helm: config.HelmConfig{Workspace: t.TempDir()},
	}
	step := config.PipelineStep{
		Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom",
		Chart: "oci://example/prom", Version: "1.0.0",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 0, step); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "helm sdk upgrade --install mon/prom") {
		t.Fatalf("expected sdk plan, got: %q", buf.String())
	}
}

func TestDryRunner_PVCDeletePlan(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "shell"}}
	step := config.PipelineStep{
		Ref: "pvc.database/data-postgresql-0", Type: "pvc",
		Namespace: "database", Name: "data-postgresql-0",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "delete pvc database/data-postgresql-0") {
		t.Fatalf("expected pvc delete plan, got: %q", buf.String())
	}
}

func TestDryRunner_CronJobAndJobPlan(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "shell"}}
	cj := config.PipelineStep{Ref: "cronjob.batch/nightly", Type: "cronjob", Namespace: "batch", Name: "nightly"}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, cj); err != nil {
		t.Fatal(err)
	}
	job := config.PipelineStep{
		Ref: "job.batch/migrate", Type: "job", Namespace: "batch", Name: "migrate",
		Manifest: "./jobs/migrate.yaml",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 1, job); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "spec.suspend=true") {
		t.Fatalf("expected cronjob suspend plan, got: %q", out)
	}
	if !strings.Contains(out, "create job from manifest ./jobs/migrate.yaml") {
		t.Fatalf("expected job create plan, got: %q", out)
	}
}

func TestDryRunner_CronJobAndJobServerSide(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	var suspended *bool
	var created bool
	r := &DryRunner{
		Log: testEmitter(&buf),
		cronJob: &stubCronJob{
			suspendFn: func(_ context.Context, _, _ string, suspend bool) error {
				suspended = &suspend
				return nil
			},
		},
		job: &stubJob{
			deleteFn: func(context.Context, string, string) error { return nil },
			createFn: func(_ context.Context, _, _, _ string) error {
				created = true
				return nil
			},
			waitFn: func(context.Context, string, string, time.Duration) error { return nil },
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "native"}}
	cj := config.PipelineStep{Ref: "cronjob.batch/nightly", Type: "cronjob", Namespace: "batch", Name: "nightly"}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 0, cj); err != nil {
		t.Fatal(err)
	}
	if suspended == nil || *suspended {
		t.Fatalf("expected unsuspend, got %v", suspended)
	}
	job := config.PipelineStep{
		Ref: "job.batch/migrate", Type: "job", Namespace: "batch", Name: "migrate",
		Manifest: "./jobs/migrate.yaml",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 1, job); err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected job create dry-run")
	}
	out := buf.String()
	if !strings.Contains(out, "server-side dry-run ok") {
		t.Fatalf("expected server-side note, got: %q", out)
	}
}

func TestDryRunner_ExecPlan(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "shell"}}
	step := config.PipelineStep{
		Ref: "exec.database/postgresql-0", Type: "exec",
		Namespace: "database", Name: "postgresql-0",
		Container: "postgres", Command: []string{"psql", "-c", "select 1"},
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "exec database/postgresql-0 container=postgres") {
		t.Fatalf("expected exec plan, got: %q", buf.String())
	}
}

func TestDryRunner_ReleaseDownSDKPlan(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := &DryRunner{Log: testEmitter(&buf)}
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run", Execution: "native"}}
	step := config.PipelineStep{
		Ref: "release.monitoring/prom", Type: "release", Namespace: "monitoring", Name: "prom",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "helm sdk uninstall monitoring/prom") {
		t.Fatalf("expected sdk uninstall plan, got: %q", buf.String())
	}
}
