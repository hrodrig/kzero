package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDryRunner_CustomScriptIsPlannedOnly(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run"}}
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
		Run:    config.RunConfig{Mode: "dry-run"},
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
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run"}}

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
	cfg := &config.Config{Run: config.RunConfig{Mode: "dry-run"}}
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
