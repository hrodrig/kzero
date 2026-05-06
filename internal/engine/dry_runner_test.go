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
	r := &DryRunner{Out: &buf}
	step := config.PipelineStep{Custom: "./hooks/x.sh"}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[dry-run]") || !strings.Contains(out, "./hooks/x.sh") {
		t.Fatalf("unexpected output: %q", out)
	}
}
