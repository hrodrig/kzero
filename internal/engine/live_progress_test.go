package engine

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
)

func TestWithThrottledProgress_returnsWhenContextCanceled(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{Log: log.New(io.Discard, log.FormatText)}
	step := config.PipelineStep{Ref: "deployment.ns/app"}

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})

	go func() {
		<-started
		cancel()
	}()

	err := r.withThrottledProgress(ctx, step, "waiting rollout", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWithThrottledProgress_propagatesOperationError(t *testing.T) {
	t.Parallel()

	want := errors.New("rollout failed")
	r := &LiveRunner{Log: log.New(io.Discard, log.FormatText)}
	step := config.PipelineStep{Ref: "deployment.ns/app"}

	err := r.withThrottledProgress(context.Background(), step, "waiting rollout", func(context.Context) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestWithThrottledProgress_nilLogRunsFnDirectly(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{}
	called := false
	err := r.withThrottledProgress(context.Background(), config.PipelineStep{}, "waiting", func(context.Context) error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Fatalf("expected direct fn call, called=%v err=%v", called, err)
	}
}

func TestWithThrottledProgress_emitsThrottledLines(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("throttled progress timing test")
	}

	// Override ticker interval indirectly by completing quickly; this test
	// only verifies the happy path returns without hanging.
	r := &LiveRunner{Log: log.New(io.Discard, log.FormatText)}
	step := config.PipelineStep{Ref: "release.ns/chart"}

	err := r.withThrottledProgress(context.Background(), step, "installing release", func(context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
