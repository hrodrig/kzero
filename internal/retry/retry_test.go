package retry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestExponential_backoff(t *testing.T) {
	t.Parallel()
	base := 8 * time.Second
	if got := Exponential(base, 1); got != 8*time.Second {
		t.Fatalf("failedTry 1: got %v want 8s", got)
	}
	if got := Exponential(base, 2); got != 16*time.Second {
		t.Fatalf("failedTry 2: got %v want 16s", got)
	}
	if got := Exponential(base, 3); got != 32*time.Second {
		t.Fatalf("failedTry 3: got %v want 32s", got)
	}
}

func TestExponential_capsAtMax(t *testing.T) {
	t.Parallel()
	base := time.Minute
	if got := Exponential(base, 10); got != maxBackoff {
		t.Fatalf("got %v want cap %v", got, maxBackoff)
	}
}

func TestBackoff_fullJitterInRange(t *testing.T) {
	t.Parallel()
	base := 8 * time.Second
	ceiling := Exponential(base, 2) // 16s
	seen := make(map[time.Duration]struct{})
	for i := 0; i < 80; i++ {
		got := Backoff(base, 2)
		if got < 0 || got > ceiling {
			t.Fatalf("jitter out of range: got %v want [0, %v]", got, ceiling)
		}
		seen[got] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected varied jitter samples, got %d distinct values", len(seen))
	}
}

func TestFullJitter_zero(t *testing.T) {
	t.Parallel()
	if got := fullJitter(0); got != 0 {
		t.Fatalf("got %v", got)
	}
	if got := fullJitter(-1); got != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestIsRetriable_commonCases(t *testing.T) {
	t.Parallel()
	conflict := fmtConflict()
	if !IsRetriable(conflict) {
		t.Fatal("conflict should retry")
	}
	if IsRetriable(executor.ErrNotFound) {
		t.Fatal("not found should not retry")
	}
	if !IsRetriable(context.DeadlineExceeded) {
		t.Fatal("deadline exceeded should retry")
	}
	if IsRetriable(context.Canceled) {
		t.Fatal("canceled should not retry")
	}
	if !IsRetriable(errors.New("dial tcp: connection refused")) {
		t.Fatal("connection refused should retry")
	}
	if !IsRetriable(fmt.Errorf("helm: %w", executor.ErrTransient)) {
		t.Fatal("ErrTransient should retry")
	}
}

func fmtConflict() error {
	st := apierrors.NewConflict(schema.GroupResource{Resource: "deployments"}, "app", errors.New("rv"))
	return executor.WrapAPIError(st, "deployment ns/app")
}

func TestAttempts_defaults(t *testing.T) {
	t.Parallel()
	if got := Attempts(nil); got != 1 {
		t.Fatalf("nil cfg: got %d want 1", got)
	}
	if got := Attempts(&config.Config{Retry: config.RetryConfig{Attempts: 0}}); got != 1 {
		t.Fatalf("zero attempts: got %d want 1", got)
	}
	if got := Attempts(&config.Config{Retry: config.RetryConfig{Attempts: 4}}); got != 4 {
		t.Fatalf("got %d want 4", got)
	}
}

func TestExponential_zeroBaseUsesDefault(t *testing.T) {
	t.Parallel()
	if got := Exponential(0, 1); got != 5*time.Second {
		t.Fatalf("got %v want 5s", got)
	}
	if got := Exponential(-1, 2); got != 10*time.Second {
		t.Fatalf("got %v want 10s", got)
	}
}

func TestExponential_failedTryBelowOne(t *testing.T) {
	t.Parallel()
	if got := Exponential(4*time.Second, 0); got != 4*time.Second {
		t.Fatalf("got %v", got)
	}
}

func TestExponential_exceedsMaxWithoutEarlyLoopExit(t *testing.T) {
	t.Parallel()
	// failedTry=2 with huge base hits d > maxBackoff after single double
	if got := Exponential(90*time.Second, 2); got != maxBackoff {
		t.Fatalf("got %v want %v", got, maxBackoff)
	}
}

func TestIsRetriable_apiStatusErrors(t *testing.T) {
	t.Parallel()
	gr := schema.GroupResource{Resource: "deployments"}
	cases := []struct {
		err  error
		want bool
	}{
		{apierrors.NewTooManyRequests("x", 5), true},
		{apierrors.NewServiceUnavailable("x"), true},
		{apierrors.NewServerTimeout(gr, "x", 1), true},
		{apierrors.NewTimeoutError("x", 1), true},
		{executor.WrapAPIError(apierrors.NewForbidden(gr, "x", errors.New("denied")), "dep"), false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := IsRetriable(tc.err); got != tc.want {
			t.Fatalf("IsRetriable(%v)=%v want %v", tc.err, got, tc.want)
		}
	}
}

func TestIsRetriable_messageSubstrings(t *testing.T) {
	t.Parallel()
	for _, msg := range []string{
		"HTTP 503 Service Unavailable",
		"error 504 gateway",
		"got 429 too many",
		"temporarily unavailable",
		"server timeout waiting",
		"TLS handshake timeout",
		"connection reset by peer",
	} {
		if !IsRetriable(errors.New(msg)) {
			t.Fatalf("expected retriable: %q", msg)
		}
	}
	if IsRetriable(errors.New("permission denied permanent")) {
		t.Fatal("expected non-retriable generic error")
	}
}

func TestLogRetry_writesLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cfg := &config.Config{Client: config.ClientConfig{ID: "e2e"}}
	LogRetry(&buf, cfg, "down", 2, "deployment.ns/app", 1, 3, 8*time.Second, errors.New("boom"))
	got := buf.String()
	for _, want := range []string{
		"[retry]", "client_id=e2e", "pipeline down", "deployment.ns/app", "attempt 1/3", "boom", "8s",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestLogRetry_nilWriterNoPanic(t *testing.T) {
	t.Parallel()
	LogRetry(nil, nil, "up", 0, "", 1, 2, time.Second, errors.New("x"))
}

func TestLogRetry_emptyStepRefUsesIndex(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	LogRetry(&buf, nil, "up", 5, "", 2, 5, time.Second, errors.New("x"))
	if !strings.Contains(buf.String(), "index 5") {
		t.Fatalf("got %q", buf.String())
	}
}
