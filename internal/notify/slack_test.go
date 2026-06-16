package notify

import (
	"strings"
	"testing"
	"time"
)

func TestFormatHumanDuration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0sec"},
		{45 * time.Second, "45sec"},
		{2 * time.Minute, "2min"},
		{7*time.Minute + 34*time.Second, "7min 34sec"},
		{3 * time.Minute, "3min"},
	}
	for _, tc := range cases {
		if got := formatHumanDuration(tc.d); got != tc.want {
			t.Fatalf("formatHumanDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestSlackFooter_version(t *testing.T) {
	old := AppVersion
	t.Cleanup(func() { AppVersion = old })
	AppVersion = "v0.7.2"
	if got := slackFooter(); got != "kzero v0.7.2" {
		t.Fatalf("got %q", got)
	}
	AppVersion = "dev"
	if got := slackFooter(); got != "kzero" {
		t.Fatalf("dev fallback: got %q", got)
	}
}

func TestSlackColorAndTitle_eventColors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		event     string
		wantColor string
	}{
		{EventStart, slackColorStart},
		{EventSuccess, slackColorSuccess},
		{EventError, slackColorError},
		{EventTest, slackColorTest},
	}
	for _, tc := range cases {
		color, _ := slackColorAndTitle(tc.event)
		if color != tc.wantColor {
			t.Fatalf("event %s: color %q, want %q", tc.event, color, tc.wantColor)
		}
	}
	color, title := slackColorAndTitle("custom")
	if color != slackColorTest || title != "kzero custom" {
		t.Fatalf("default: color=%q title=%q", color, title)
	}
}

func TestBuildSlackBody_richAttachmentFields(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 6, 11, 7, 10, 6, 0, time.UTC)
	meta := Meta{
		Command:     "reset",
		Mode:        "live",
		StartedAt:   start,
		Duration:    7*time.Minute + 34*time.Second,
		ClientID:    "AZVM220001",
		Cluster:     "AZKB220001",
		KubeContext: "my-k8s",
		Environment: "production",
		OSUser:      "posadmin01",
	}
	body := buildPayload(EventSuccess, meta)
	msg := buildSlackBody(EventSuccess, meta, body)
	if len(msg.Attachments) != 1 {
		t.Fatalf("attachments: %+v", msg.Attachments)
	}
	att := msg.Attachments[0]
	if att.Color != slackColorSuccess {
		t.Fatalf("color: %q", att.Color)
	}
	if att.Title != "✅ kzero completed" {
		t.Fatalf("title: %q", att.Title)
	}
	if att.Footer != "kzero" {
		t.Fatalf("footer without ldflags: got %q", att.Footer)
	}
	for _, want := range []string{
		"*Cluster:* `AZKB220001`",
		"*Client:* `AZVM220001`",
		"*Context:* `my-k8s`",
		"*User:* `posadmin01`",
		"*Mode:* `PRODUCTION`",
		"*Duration:* `7min 34sec`",
	} {
		if !strings.Contains(att.Text, want) {
			t.Fatalf("text missing %q:\n%s", want, att.Text)
		}
	}
}

func TestBuildSlackBody_errorIncludesFailedStep(t *testing.T) {
	t.Parallel()
	meta := Meta{
		Command:    "down",
		StartedAt:  time.Now(),
		FailedStep: "deployment.app/api",
		Error:      "rollout timeout",
	}
	body := buildPayload(EventError, meta)
	msg := buildSlackBody(EventError, meta, body)
	if !strings.Contains(msg.Attachments[0].Text, "*Failed step:* `deployment.app/api`") {
		t.Fatalf("text: %q", msg.Attachments[0].Text)
	}
	if msg.Attachments[0].Color != slackColorError {
		t.Fatalf("color: %q", msg.Attachments[0].Color)
	}
}
