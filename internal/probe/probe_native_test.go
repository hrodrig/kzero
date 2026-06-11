package probe

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
)

func nativeProbeCfg() *config.Config {
	return &config.Config{
		Cluster: config.ClusterConfig{Name: "test"},
		Run:     config.RunConfig{Mode: "dry-run", Execution: "native"},
		Helm:    config.HelmConfig{Workspace: "./helm-assets"},
		InfraProbe: config.InfraProbeConfig{
			Enabled:  true,
			FailFast: true,
			Before:   []string{"reset"},
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{
					Ref: "release.probe-ns/probe-redis", Type: "release",
					Namespace: "probe-ns", Name: "probe-redis",
				}},
				Down: []config.PipelineStep{
					{Ref: "release.probe-ns/probe-redis", Type: "release", Namespace: "probe-ns", Name: "probe-redis"},
					{Ref: "pvc.probe-ns/redis-data-probe-redis-master-0", Type: "pvc", Namespace: "probe-ns", Name: "redis-data-probe-redis-master-0"},
				},
			},
			Checks: []config.ProbeCheck{
				{PVCBound: "probe-ns/redis-data-probe-redis-master-0"},
				{ReleaseReady: true},
			},
		},
	}
}

func TestRun_nativeProbePipelineSteps(t *testing.T) {
	t.Parallel()

	rec := &engine.RecordingRunner{}
	cfg := nativeProbeCfg()
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	if err := Run(context.Background(), cfg, eng, nil, emit); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.Calls) != 3 {
		t.Fatalf("expected up + down(release) + down(pvc), got %d calls", len(rec.Calls))
	}
	if rec.Calls[0].Step.Type != "release" || rec.Calls[0].Phase != engine.PhaseUp {
		t.Fatalf("call 0: %#v", rec.Calls[0])
	}
	if rec.Calls[1].Step.Type != "release" || rec.Calls[2].Step.Type != "pvc" {
		t.Fatalf("down calls: %#v, %#v", rec.Calls[1].Step, rec.Calls[2].Step)
	}
}

func TestLoadConfig_nativeInfraProbeFragment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fragment := filepath.Join("..", "..", "docs", "examples", "infra-probe", "kzero-infra-probe-native.sample.yaml")
	raw, err := os.ReadFile(fragment)
	if err != nil {
		t.Skipf("fragment not found: %v", err)
	}
	path := filepath.Join(dir, "kzero.yaml")
	full := "pipelines:\n  down:\n    - deployment.default/placeholder\n" + string(raw)
	if err := os.WriteFile(path, []byte(full), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Run.Execution != "native" {
		t.Fatalf("execution: %q", cfg.Run.Execution)
	}
	if len(cfg.InfraProbe.Pipeline.Up) != 1 || cfg.InfraProbe.Pipeline.Up[0].Type != "release" {
		t.Fatalf("probe up: %#v", cfg.InfraProbe.Pipeline.Up)
	}
}

func TestNativeProbeDryRunPlan(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	cfg := nativeProbeCfg()
	emit := log.New(&buf, log.FormatText)
	eng := &engine.Engine{Runner: engine.NewDryRunner(cfg, emit), Log: emit}
	if err := Run(context.Background(), cfg, eng, nil, emit); err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "helm sdk upgrade --install") {
		t.Fatalf("missing sdk up plan: %q", out)
	}
	if !strings.Contains(out, "helm sdk uninstall") {
		t.Fatalf("missing sdk down plan: %q", out)
	}
	if !strings.Contains(out, "delete pvc probe-ns/redis-data-probe-redis-master-0") {
		t.Fatalf("missing pvc down plan: %q", out)
	}
}
