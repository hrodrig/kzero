package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/spf13/cobra"
)

func TestRunPipelineCommand_probeGateLiveDryRunSteps(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "kzero.yaml")
	cfgYAML := `
schema_version: "1.0"
helm:
  workspace: ./helm
cluster:
  name: gate-test
pipelines:
  down:
    - deployment.ns/app
infra_probe:
  enabled: true
  before: ["down"]
  pipeline:
    up:
      - release.probe-ns/probe-storage
    down:
      - release.probe-ns/probe-storage
run:
  mode: live
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetContext(context.Background())

	rec := &engine.RecordingRunner{}
	emit := log.New(&out, log.FormatText)
	emit.SetCommand("down")
	eng := engine.New(cfg, emit)
	eng.Runner = rec

	if err := runInfraProbeGate(cmd, cfg, eng, "down", cmd.Context()); err != nil {
		t.Fatalf("runInfraProbeGate: %v\n%s", err, out.String())
	}
	if len(rec.Calls) < 2 {
		t.Fatalf("expected probe up+down, got %d calls\n%s", len(rec.Calls), out.String())
	}
	if !strings.Contains(out.String(), "infra probe") {
		t.Fatalf("expected probe log output, got:\n%s", out.String())
	}
}

func TestRunInfraProbeGate_nilLog(t *testing.T) {
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		InfraProbe: config.InfraProbeConfig{
			Pipeline: config.ProbePipelineConfig{
				Up: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
			},
		},
	}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetContext(context.Background())
	eng := &engine.Engine{Runner: &engine.RecordingRunner{}}
	if err := runInfraProbeGate(cmd, cfg, eng, "down", cmd.Context()); err != nil {
		t.Fatalf("runInfraProbeGate: %v", err)
	}
}
