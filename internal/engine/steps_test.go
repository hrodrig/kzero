package engine

import (
	"context"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestRunSteps_orderAndPhases(t *testing.T) {
	t.Parallel()
	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
	}
	up := []config.PipelineStep{
		{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"},
		{Ref: "deployment.ns/b", Type: "deployment", Namespace: "ns", Name: "b"},
	}
	if err := eng.RunSteps(context.Background(), cfg, PhaseUp, up); err != nil {
		t.Fatalf("RunSteps: %v", err)
	}
	if len(rec.Calls) != 2 {
		t.Fatalf("expected 2 step calls, got %d", len(rec.Calls))
	}
	if rec.Calls[0].Phase != PhaseUp || rec.Calls[0].Index != 0 {
		t.Fatalf("unexpected first call: %+v", rec.Calls[0])
	}
	if rec.Calls[1].Phase != PhaseUp || rec.Calls[1].Index != 1 {
		t.Fatalf("unexpected second call: %+v", rec.Calls[1])
	}
}
