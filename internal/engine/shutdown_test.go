package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
)

type blockingRunner struct {
	startOnce sync.Once
	started   chan struct{}
}

func (b *blockingRunner) waitStarted() {
	select {
	case <-b.started:
	case <-time.After(2 * time.Second):
		panic("blocking runner did not start")
	}
}

func (b *blockingRunner) RunHook(_ context.Context, _ *config.Config, _, _ string) error {
	return nil
}

func (b *blockingRunner) RunPipelineStep(ctx context.Context, _ *config.Config, _ Phase, _ int, _ config.PipelineStep) error {
	b.startOnce.Do(func() {
		close(b.started)
	})
	<-ctx.Done()
	return ctx.Err()
}

func TestRunDown_cancelsDuringStepWithinBoundedTime(t *testing.T) {
	t.Parallel()

	stubLivePreflightOK(t)

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Type: "deployment", Ref: "ns/app"},
			},
		},
	}

	var buf strings.Builder
	runner := &blockingRunner{started: make(chan struct{})}
	eng := &Engine{
		Runner:  runner,
		Log:     log.New(&buf, log.FormatText),
		Command: "down",
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- eng.RunDown(ctx, cfg)
	}()

	runner.waitStarted()
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error on cancel")
		}
		var pe *PipelineError
		if !errors.As(err, &pe) || !errors.Is(pe.Err, context.Canceled) {
			t.Fatalf("expected canceled pipeline error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunDown did not return within bounded time after cancel")
	}

	out := buf.String()
	if !strings.Contains(out, "pipeline interrupted (signal received)") {
		t.Fatalf("expected interrupt log line, got: %q", out)
	}
	if !strings.Contains(out, "ns/app") {
		t.Fatalf("expected last step in interrupt log, got: %q", out)
	}
}

func TestFinishWithError_watchdogCancelDoesNotLogUserInterrupt(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	eng := &Engine{
		Log:     log.New(&buf, log.FormatText),
		stalled: true,
	}
	eng.setProgressStep(PhaseDown, 0, "ns/app")

	err := finishWithError(context.Background(), eng, &config.Config{}, &PipelineError{
		Phase: string(PhaseDown),
		Index: 0,
		Ref:   "ns/app",
		Err:   context.Canceled,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(buf.String(), "pipeline interrupted") {
		t.Fatalf("watchdog cancel should not emit user interrupt line: %q", buf.String())
	}
}

func TestIsUserInterrupt(t *testing.T) {
	t.Parallel()

	eng := &Engine{}
	if !isUserInterrupt(eng, context.Canceled) {
		t.Fatal("expected user interrupt for context.Canceled")
	}
	if !isUserInterrupt(eng, &PipelineError{Err: context.Canceled}) {
		t.Fatal("expected user interrupt through PipelineError")
	}
	eng.stalled = true
	if isUserInterrupt(eng, context.Canceled) {
		t.Fatal("stalled engine should not count as user interrupt")
	}
}
