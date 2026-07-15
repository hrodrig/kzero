package engine

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/exitcode"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/notify"
)

func TestFinishWithError_requireDeliveryFailsOnNotifyError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := true
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			RequireDelivery: &req,
			Webhook: config.GenericWebhookConfig{
				Enabled: true,
				URL:     srv.URL,
			},
		},
	}

	pipelineErr := &PipelineError{
		Phase: string(PhaseDown),
		Index: 0,
		Ref:   "ns/app",
		Err:   errors.New("step failed"),
	}

	eng := &Engine{
		Command: "down",
		Started: time.Now(),
		Log:     log.New(nil, log.FormatText),
	}

	got := finishWithError(context.Background(), eng, cfg, pipelineErr)
	if got == nil {
		t.Fatal("expected error when require_delivery and notify POST fails")
	}
	if !errors.Is(got, pipelineErr) {
		t.Fatalf("expected wrapped pipeline error, got %v", got)
	}
	if !strings.Contains(got.Error(), "notify delivery required") {
		t.Fatalf("expected notify delivery error, got %v", got)
	}
	if code := exitcode.Of(got); code != exitcode.NotifyFailed {
		t.Fatalf("exit code %d, want %d (notify)", code, exitcode.NotifyFailed)
	}
	if !strings.Contains(got.Error(), notify.EventError) {
		t.Fatalf("expected pipeline.error event in error, got %v", got)
	}
}

func TestFinishWithError_requireDeliveryFailsOnStalledNotify(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := true
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			RequireDelivery: &req,
			Webhook: config.GenericWebhookConfig{
				Enabled: true,
				URL:     srv.URL,
			},
		},
	}

	eng := &Engine{
		Command: "down",
		Started: time.Now(),
		stalled: true,
		Log:     log.New(nil, log.FormatText),
	}

	got := finishWithError(context.Background(), eng, cfg, ErrPipelineStalled)
	if got == nil {
		t.Fatal("expected error when require_delivery and stalled notify POST fails")
	}
	if !errors.Is(got, ErrPipelineStalled) {
		t.Fatalf("expected wrapped stalled error, got %v", got)
	}
	if !strings.Contains(got.Error(), notify.EventStalled) {
		t.Fatalf("expected pipeline.stalled event in error, got %v", got)
	}
	if code := exitcode.Of(got); code != exitcode.NotifyFailed {
		t.Fatalf("exit code %d, want %d (notify)", code, exitcode.NotifyFailed)
	}
}

func TestFinishWithError_notifyFailureWithoutRequireDeliveryReturnsPipelineErr(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req := false
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			RequireDelivery: &req,
			Webhook: config.GenericWebhookConfig{
				Enabled: true,
				URL:     srv.URL,
			},
		},
	}

	pipelineErr := &PipelineError{
		Phase: string(PhaseDown),
		Index: 0,
		Ref:   "ns/app",
		Err:   errors.New("step failed"),
	}

	eng := &Engine{
		Command: "down",
		Started: time.Now(),
		Log:     log.New(nil, log.FormatText),
	}

	got := finishWithError(context.Background(), eng, cfg, pipelineErr)
	if !errors.Is(got, pipelineErr) {
		t.Fatalf("expected original pipeline error only, got %v", got)
	}
	if strings.Contains(got.Error(), "notify delivery required") {
		t.Fatalf("unexpected notify delivery error without require_delivery: %v", got)
	}
}
