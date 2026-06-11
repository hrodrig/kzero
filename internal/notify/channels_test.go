package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/redact"
)

func TestDispatch_allChannels(t *testing.T) {
	t.Parallel()

	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldPD := pagerDutyEventsURL
	pagerDutyEventsURL = srv.URL
	defer func() { pagerDutyEventsURL = oldPD }()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			Slack:     config.ChannelConfig{Enabled: true, WebhookURL: srv.URL},
			Discord:   config.ChannelConfig{Enabled: true, WebhookURL: srv.URL},
			Teams:     config.ChannelConfig{Enabled: true, WebhookURL: srv.URL},
			PagerDuty: config.PagerDutyConfig{Enabled: true, RoutingKey: "test-key"},
			Webhook:   config.GenericWebhookConfig{Enabled: true, URL: srv.URL},
		},
	}
	meta := Meta{Command: "up", Mode: "live", StartedAt: time.Now()}
	if err := Dispatch(context.Background(), cfg, EventStart, meta, srv.Client()); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if posts != 5 {
		t.Fatalf("got %d posts, want 5", posts)
	}
}

func TestDispatch_errorIncludesFailedStep(t *testing.T) {
	t.Parallel()

	var raw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			Webhook: config.GenericWebhookConfig{Enabled: true, URL: srv.URL},
		},
	}
	meta := Meta{
		Command:    "down",
		Mode:       "live",
		StartedAt:  time.Now(),
		FailedStep: "deployment.app/api",
		Error:      "step failed",
	}
	if err := Dispatch(context.Background(), cfg, EventError, meta, srv.Client()); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.FailedStep != "deployment.app/api" || p.Error != "step failed" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestDispatch_onErrorDisabled(t *testing.T) {
	t.Parallel()

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	onError := false
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			OnError: &onError,
			Webhook: config.GenericWebhookConfig{Enabled: true, URL: srv.URL},
		},
	}
	meta := Meta{Command: "down", Mode: "live", StartedAt: time.Now(), Error: "boom"}
	if err := Dispatch(context.Background(), cfg, EventError, meta, srv.Client()); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if called {
		t.Fatal("expected no POST when on_error is false")
	}
}

func TestRedactURL(t *testing.T) {
	t.Parallel()
	u := "https://hooks.slack.com/services/T00/B00/XXXXXXXXXXXXXXXXXXXXXXXX"
	got := redact.URL(u)
	if got == u {
		t.Fatalf("expected redaction, got %q", got)
	}
}
