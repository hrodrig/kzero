package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDispatch_liveSlackPostsPayload(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		data, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		mu.Lock()
		bodies = append(bodies, m)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "live"},
		Notify: config.NotifyConfig{
			Slack: config.ChannelConfig{Enabled: true, WebhookURL: srv.URL},
		},
		Client:  config.ClientConfig{ID: "pilot"},
		Cluster: config.ClusterConfig{Name: "dev"},
	}
	started := time.Now().Add(-2 * time.Second)
	meta := MetaFromConfig(cfg, "down", started, 2*time.Second)
	meta.Error = ""
	if err := Dispatch(context.Background(), cfg, EventSuccess, meta, srv.Client()); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 1 {
		t.Fatalf("got %d bodies, want 1", len(bodies))
	}
	if bodies[0]["text"] == nil {
		t.Fatalf("slack body missing text: %v", bodies[0])
	}
}

func TestDispatch_skipsDryRun(t *testing.T) {
	t.Parallel()

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Notify: config.NotifyConfig{
			Webhook: config.GenericWebhookConfig{Enabled: true, URL: srv.URL},
		},
	}
	meta := Meta{Command: "up", Mode: "dry-run", StartedAt: time.Now()}
	if err := Dispatch(context.Background(), cfg, EventStart, meta, srv.Client()); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if called {
		t.Fatal("expected no HTTP call in dry-run mode")
	}
}

func TestDispatch_webhookJSONPayload(t *testing.T) {
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
	started := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	meta := Meta{Command: "reset", Mode: "live", StartedAt: started, Duration: time.Minute}
	if err := Dispatch(context.Background(), cfg, EventStart, meta, srv.Client()); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, raw)
	}
	if p.Event != EventStart || p.Command != "reset" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestOnErrorEnabled_defaultTrue(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Notify: config.NotifyConfig{Slack: config.ChannelConfig{Enabled: true}},
	}
	if !OnErrorEnabled(cfg) {
		t.Fatal("expected default on_error true")
	}
}

func TestAnyEnabled_falseWhenEmpty(t *testing.T) {
	t.Parallel()
	if AnyEnabled(&config.Config{}) {
		t.Fatal("expected false")
	}
}
