package engine

import (
	"errors"
	"testing"
)

func TestPipelineError_failedStep(t *testing.T) {
	t.Parallel()
	pe := &PipelineError{Phase: "down", Index: 2, Ref: "deployment.app/api", Err: errors.New("boom")}
	if got := pe.FailedStep(); got != "deployment.app/api" {
		t.Fatalf("got %q", got)
	}
	if !errors.As(pe, new(*PipelineError)) {
		t.Fatal("expected errors.As")
	}
}

func TestPipelineError_hookLabel(t *testing.T) {
	t.Parallel()
	pe := &PipelineError{Hook: "pre-down", Err: errors.New("x")}
	if got := pe.FailedStep(); got != "hook:pre-down" {
		t.Fatalf("got %q", got)
	}
}
