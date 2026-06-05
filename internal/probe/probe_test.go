package probe

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func testEmitter(w io.Writer) *log.Emitter {
	return log.New(w, log.FormatText)
}

func probeCfg() *config.Config {
	return &config.Config{
		Cluster: config.ClusterConfig{Name: "test"},
		Run:     config.RunConfig{Mode: "dry-run"},
		InfraProbe: config.InfraProbeConfig{
			Enabled:  true,
			FailFast: true,
			Before:   []string{"reset", "down"},
			Pipeline: config.ProbePipelineConfig{
				Up:   []config.PipelineStep{{Ref: "release.probe-ns/probe-storage", Type: "release", Namespace: "probe-ns", Name: "probe-storage"}},
				Down: []config.PipelineStep{{Ref: "release.probe-ns/probe-storage", Type: "release", Namespace: "probe-ns", Name: "probe-storage"}},
			},
			Checks: []config.ProbeCheck{{ReleaseReady: true}},
		},
		Helm: config.HelmConfig{Workspace: "./helm"},
	}
}

func TestShouldGate_liveAndCommands(t *testing.T) {
	t.Parallel()
	cfg := probeCfg()
	cfg.Run.Mode = "live"
	if !ShouldGate(cfg, "reset") || !ShouldGate(cfg, "down") {
		t.Fatal("expected gate for reset/down in live")
	}
	if ShouldGate(cfg, "up") {
		t.Fatal("up should not gate")
	}
	cfg.Run.Mode = "dry-run"
	if ShouldGate(cfg, "reset") {
		t.Fatal("dry-run should not gate")
	}
	cfg.InfraProbe.Enabled = false
	cfg.Run.Mode = "live"
	if ShouldGate(cfg, "reset") {
		t.Fatal("disabled should not gate")
	}
}

func TestRun_dryRunUpDown(t *testing.T) {
	t.Parallel()
	rec := &engine.RecordingRunner{}
	cfg := probeCfg()
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	if err := Run(context.Background(), cfg, eng, nil, emit); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.Calls) != 2 {
		t.Fatalf("expected up+down steps, got %d calls", len(rec.Calls))
	}
}

func TestRun_upFailureSkipsDown(t *testing.T) {
	t.Parallel()
	rec := &engine.RecordingRunner{
		StepErr: map[string]error{"up:0": errors.New("up boom")},
	}
	cfg := probeCfg()
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	err := Run(context.Background(), cfg, eng, nil, emit)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(rec.Calls) != 1 {
		t.Fatalf("expected only up step, got %d", len(rec.Calls))
	}
}

func TestRunGate_failFastFalseContinues(t *testing.T) {
	t.Parallel()
	rec := &engine.RecordingRunner{
		StepErr: map[string]error{"up:0": errors.New("fail")},
	}
	cfg := probeCfg()
	cfg.InfraProbe.FailFast = false
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	if err := RunGate(context.Background(), cfg, eng, nil, emit, "reset"); err != nil {
		t.Fatalf("RunGate with fail_fast=false: %v", err)
	}
}

func TestRun_checksPVCBound(t *testing.T) {
	t.Parallel()
	rec := &engine.RecordingRunner{}
	cfg := probeCfg()
	cfg.Run.Mode = "live"
	cfg.InfraProbe.Checks = []config.ProbeCheck{{PVCBound: "probe-ns/probe-pvc"}}
	client := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "probe-pvc", Namespace: "probe-ns"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	})
	factory := func(string) (kubernetes.Interface, error) { return client, nil }
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	if err := Run(context.Background(), cfg, eng, factory, emit); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_checksPVCBoundNotBound(t *testing.T) {
	t.Parallel()
	rec := &engine.RecordingRunner{}
	cfg := probeCfg()
	cfg.Run.Mode = "live"
	cfg.InfraProbe.Checks = []config.ProbeCheck{{PVCBound: "probe-ns/probe-pvc"}}
	client := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "probe-pvc", Namespace: "probe-ns"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
	})
	factory := func(string) (kubernetes.Interface, error) { return client, nil }
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	err := Run(context.Background(), cfg, eng, factory, emit)
	if err == nil {
		t.Fatal("expected pvc check error")
	}
	if len(rec.Calls) != 2 {
		t.Fatalf("expected up then down teardown after failed check; got %d calls", len(rec.Calls))
	}
}

func TestRunGate_failFastTrueBlocks(t *testing.T) {
	t.Parallel()
	rec := &engine.RecordingRunner{
		StepErr: map[string]error{"up:0": errors.New("fail")},
	}
	cfg := probeCfg()
	cfg.InfraProbe.FailFast = true
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	if err := RunGate(context.Background(), cfg, eng, nil, emit, "reset"); err == nil {
		t.Fatal("expected gate error with fail_fast=true")
	}
}

func TestRun_emptyPipelineUp(t *testing.T) {
	t.Parallel()
	cfg := probeCfg()
	cfg.InfraProbe.Pipeline.Up = nil
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: &engine.RecordingRunner{}, Log: emit}
	if err := Run(context.Background(), cfg, eng, nil, emit); err == nil {
		t.Fatal("expected empty pipeline error")
	}
}

func TestIsFresh_defaultCacheDir(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Cluster: config.ClusterConfig{Name: "cache-default"},
		InfraProbe: config.InfraProbeConfig{
			CacheTTL: time.Minute,
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{Ref: "release.ns/a"}},
			},
		},
	}
	if err := WriteOK(cfg); err != nil {
		t.Fatalf("WriteOK: %v", err)
	}
	fresh, err := IsFresh(cfg)
	if err != nil || !fresh {
		t.Fatalf("expected fresh with default cache dir, fresh=%v err=%v", fresh, err)
	}
}

func TestRun_cacheSkip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := probeCfg()
	cfg.Run.Mode = "live"
	cfg.Run.ProbeCacheDir = dir
	cfg.InfraProbe.CacheTTL = time.Hour
	if err := WriteOK(cfg); err != nil {
		t.Fatalf("WriteOK: %v", err)
	}
	rec := &engine.RecordingRunner{}
	emit := testEmitter(io.Discard)
	eng := &engine.Engine{Runner: rec, Log: emit}
	if err := Run(context.Background(), cfg, eng, nil, emit); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.Calls) != 0 {
		t.Fatalf("cache hit should skip steps, got %d calls", len(rec.Calls))
	}
}
