package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDispatchTest_defaultEvent(t *testing.T) {
	t.Parallel()

	var raw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Notify: config.NotifyConfig{
			Webhook: config.GenericWebhookConfig{Enabled: true, URL: srv.URL},
		},
	}
	if err := DispatchTest(context.Background(), cfg, EventTest, srv.Client()); err != nil {
		t.Fatalf("DispatchTest: %v", err)
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Event != EventTest || p.Mode != "test" || p.Command != "notify-test" {
		t.Fatalf("payload: %+v", p)
	}
}

func TestDispatchTest_noChannelEnabled(t *testing.T) {
	t.Parallel()
	err := DispatchTest(context.Background(), &config.Config{}, EventTest, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateTestEvent_unknown(t *testing.T) {
	t.Parallel()
	if err := ValidateTestEvent("nope"); err == nil {
		t.Fatal("expected error")
	}
}
