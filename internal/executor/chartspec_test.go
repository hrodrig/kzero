package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestResolveChartSpec_fromManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "prom.yaml")
	content := `chart: oci://example/charts/prom
version: "1.2.3"
values_files:
  - values.yaml
create_namespace: true
wait: true
timeout: 2m
`
	if err := os.WriteFile(manifest, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("replicaCount: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Helm: config.HelmConfig{Workspace: dir}}
	step := config.PipelineStep{
		Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom",
	}
	spec, err := ResolveChartSpec(cfg, step)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Chart != "oci://example/charts/prom" || spec.Version != "1.2.3" {
		t.Fatalf("unexpected chart/version: %+v", spec)
	}
	if !spec.CreateNamespace || !spec.Wait || spec.Timeout != 2*time.Minute {
		t.Fatalf("unexpected flags: %+v", spec)
	}
	if len(spec.ValuesFiles) != 1 || spec.ValuesFiles[0] != filepath.Join(dir, "values.yaml") {
		t.Fatalf("values files: %+v", spec.ValuesFiles)
	}
}

func TestResolveChartSpec_stepOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{Helm: config.HelmConfig{Workspace: dir}}
	step := config.PipelineStep{
		Ref:       "release.mon/prom",
		Type:      "release",
		Namespace: "mon",
		Name:      "prom",
		Chart:     "oci://example/charts/prom",
		Version:   "9.9.9",
	}
	spec, err := ResolveChartSpec(cfg, step)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Chart != "oci://example/charts/prom" || spec.Version != "9.9.9" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestResolveChartSpec_missingChart(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Helm: config.HelmConfig{Workspace: t.TempDir()}}
	step := config.PipelineStep{Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom"}
	if _, err := ResolveChartSpec(cfg, step); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveChartSpec_badManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "prom.yaml"), []byte("timeout: not-a-duration\nchart: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Helm: config.HelmConfig{Workspace: dir}}
	step := config.PipelineStep{Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom"}
	if _, err := ResolveChartSpec(cfg, step); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestWantHelmSDK(t *testing.T) {
	t.Parallel()

	if WantHelmSDK(&config.Config{Run: config.RunConfig{Execution: "shell"}}) {
		t.Fatal("shell should not want sdk")
	}
	if !WantHelmSDK(&config.Config{Run: config.RunConfig{Execution: "native"}}) {
		t.Fatal("native should want sdk")
	}
}

func TestResolveChartSpec_emptyWorkspace(t *testing.T) {
	t.Parallel()

	step := config.PipelineStep{Ref: "release.mon/prom", Type: "release", Name: "prom"}
	if _, err := ResolveChartSpec(&config.Config{}, step); err == nil {
		t.Fatal("expected workspace error")
	}
}

func TestSDKHelm_Uninstall_cancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h, err := NewSDKHelm(&config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	step := config.PipelineStep{Name: "prom", Namespace: "mon", Type: "release"}
	if err := h.Uninstall(ctx, step); err == nil {
		t.Fatal("expected context error")
	}
}
