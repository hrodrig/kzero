package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
)

func writeAPIServerKubeconfig(t *testing.T, serverURL string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	content := fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: smoke
contexts:
- name: smoke
  context:
    cluster: smoke
    user: smoke
clusters:
- name: smoke
  cluster:
    server: %s
    insecure-skip-tls-verify: true
users:
- name: smoke
  user: {}
`, serverURL)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunDown_apiWatchdogTripsDuringBlockingStep(t *testing.T) {
	t.Parallel()

	var failHealthz atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		if failHealthz.Load() == 1 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	kubeconfig := writeAPIServerKubeconfig(t, srv.URL)
	cfg := &config.Config{
		Run: config.RunConfig{
			Execution: "shell",
			Mode:       "live",
			Kubeconfig: kubeconfig,
			APIWatchdog: &config.APIWatchdogConfig{
				Enabled:   true,
				Interval:  5 * time.Millisecond,
				FailAfter: 25 * time.Millisecond,
			},
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}

	runner := &blockingRunner{started: make(chan struct{})}
	eng := &Engine{
		Runner:           runner,
		Log:              log.New(io.Discard, log.FormatText),
		Command:          "down",
		PreflightFactory: livePreflightOKFactory(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- eng.RunDown(context.Background(), cfg)
	}()

	runner.waitStarted()
	failHealthz.Store(1)

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error when watchdog trips")
		}
		if !eng.Stalled() {
			t.Fatal("expected engine stalled flag")
		}
		var pe *PipelineError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PipelineError, got %v", err)
		}
		if !errors.Is(pe.Err, context.Canceled) {
			t.Fatalf("expected canceled step, got %v", pe.Err)
		}
		if isUserInterrupt(eng, err) {
			t.Fatal("watchdog trip must not classify as user interrupt")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunDown did not return after API watchdog trip")
	}
}

func TestStartAPIObserver_tripsWithFakeAPIServer(t *testing.T) {
	t.Parallel()

	var failHealthz atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		if failHealthz.Load() == 1 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Run: config.RunConfig{
			Execution: "shell",
			Mode:       "live",
			Kubeconfig: writeAPIServerKubeconfig(t, srv.URL),
			APIWatchdog: &config.APIWatchdogConfig{
				Enabled:   true,
				Interval:  5 * time.Millisecond,
				FailAfter: 20 * time.Millisecond,
			},
		},
	}
	eng := &Engine{Log: log.New(io.Discard, log.FormatText)}

	ctx, wd := eng.startAPIObserver(context.Background(), cfg)
	if wd == nil {
		t.Fatal("expected watchdog")
	}
	defer wd.Stop()

	failHealthz.Store(1)

	select {
	case <-ctx.Done():
		if !eng.Stalled() {
			t.Fatal("expected stalled after trip")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog did not cancel derived context")
	}
}
