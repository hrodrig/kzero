package notify

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

// EventTest is sent by `kzero notify test` (does not mutate the cluster).
const EventTest = "notify.test"

// TestEvents are valid --event values for `kzero notify test`.
var TestEvents = []string{EventTest, EventStart, EventSuccess, EventError}

// ValidateTestEvent reports whether event is allowed for notify test.
func ValidateTestEvent(event string) error {
	switch strings.TrimSpace(event) {
	case EventTest, EventStart, EventSuccess, EventError:
		return nil
	default:
		return fmt.Errorf("notify test: unknown event %q (want: notify.test, pipeline.start, pipeline.success, pipeline.error)", event)
	}
}

// DispatchTest posts event to all enabled channels regardless of run.mode.
// Use for `kzero notify test` — no pipeline execution.
func DispatchTest(ctx context.Context, cfg *config.Config, event string, client HTTPDoer) error {
	if err := ValidateTestEvent(event); err != nil {
		return err
	}
	if cfg == nil || !AnyEnabled(cfg) {
		return fmt.Errorf("notify test: no notify channel enabled in config")
	}
	if client == nil {
		client = http.DefaultClient
	}
	started := time.Now()
	meta := MetaFromConfig(cfg, "notify-test", started, 2*time.Second)
	meta.Mode = "test"
	switch event {
	case EventError:
		meta.FailedStep = "deployment.example/app (sample)"
		meta.Error = "notify test: simulated pipeline error"
	case EventStart:
		meta.Duration = 0
	case EventSuccess:
		meta.Duration = 2 * time.Second
	case EventTest:
		meta.Duration = 0
	}
	return joinErrors(dispatchChannels(ctx, client, cfg.Notify, buildPayload(event, meta)))
}
