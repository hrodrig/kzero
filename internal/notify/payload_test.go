package notify

import (
	"strings"
	"testing"
	"time"
)

func TestSummaryLine_events(t *testing.T) {
	t.Parallel()
	cases := []struct {
		event string
		want  string
	}{
		{EventStart, "started"},
		{EventSuccess, "succeeded"},
		{EventError, "failed"},
	}
	for _, tc := range cases {
		line := summaryLine(payload{Event: tc.event, Command: "reset", Mode: "live", Duration: "1s"})
		if !strings.Contains(line, tc.want) {
			t.Fatalf("event %s: got %q", tc.event, line)
		}
	}
}

func TestBuildPayload_duration(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	p := buildPayload(EventSuccess, Meta{
		Command:   "up",
		Mode:      "live",
		StartedAt: start,
		Duration:  1500 * time.Millisecond,
	})
	if p.Duration != "1.5s" {
		t.Fatalf("got %q", p.Duration)
	}
}

func TestBuildPayload_redactsError(t *testing.T) {
	t.Parallel()

	p := buildPayload(EventError, Meta{
		Command: "down",
		Mode:    "live",
		Error:   "notify failed Bearer secret-token SLACK_WEBHOOK_URL=https://hooks.example/x",
	})
	if strings.Contains(p.Error, "secret-token") || strings.Contains(p.Error, "hooks.example") {
		t.Fatalf("expected redacted error, got %q", p.Error)
	}
}
