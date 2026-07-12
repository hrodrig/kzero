package engine

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	"k8s.io/client-go/kubernetes"
)

func TestRunDown_dryRunPreflightBeforeHooks(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	emit := log.New(&buf, log.FormatText)
	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec, Log: emit}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{PreDown: "./pre.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}
	if err := eng.RunDown(context.Background(), cfg); err != nil {
		t.Fatalf("RunDown: %v", err)
	}
	if !strings.Contains(buf.String(), "preflight") {
		t.Fatalf("expected preflight dry-run line, got %q", buf.String())
	}
	if len(rec.Calls) < 1 || rec.Calls[0].Label != "pre-down" {
		t.Fatalf("expected pre-down after preflight, got %#v", rec.Calls)
	}
}

func TestRunDown_livePreflightBlocksPipeline(t *testing.T) {
	rec := &RecordingRunner{}
	eng := &Engine{
		Runner: rec,
		Log:    log.New(io.Discard, log.FormatText),
		PreflightFactory: func(string) (kubernetes.Interface, error) {
			return nil, errors.New("api down")
		},
	}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "live"},
		Hooks: config.HooksConfig{PreDown: "./pre.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}
	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "preflight") {
		t.Fatalf("expected preflight error, got %v", err)
	}
	if len(rec.Calls) != 0 {
		t.Fatalf("pre-down must not run when preflight fails, got %d calls", len(rec.Calls))
	}
}

func TestRunDown_livePreflightOk(t *testing.T) {
	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec, PreflightFactory: livePreflightOKFactory()}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}
	if err := eng.RunDown(context.Background(), cfg); err != nil {
		t.Fatalf("RunDown: %v", err)
	}
}
