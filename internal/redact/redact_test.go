package redact

import (
	"strings"
	"testing"
)

func TestString_bearerAndEnvSecret(t *testing.T) {
	t.Parallel()

	in := "auth failed: Bearer abc.def.ghi SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T/B/x"
	got := String(in)
	for _, secret := range []string{"abc.def.ghi", "hooks.slack.com/services/T/B/x"} {
		if strings.Contains(got, secret) {
			t.Fatalf("expected secret redacted, got %q", got)
		}
	}
	if !strings.Contains(got, "Bearer ***") {
		t.Fatalf("expected Bearer redaction, got %q", got)
	}
	if !strings.Contains(got, "SLACK_WEBHOOK_URL=***") {
		t.Fatalf("expected env secret redaction, got %q", got)
	}
}

func TestURL_webhook(t *testing.T) {
	t.Parallel()

	u := "https://hooks.slack.com/services/T000/B000/XXXXXXXXXXXXXXXXXXXXXXXX"
	got := URL(u)
	if strings.Contains(got, "XXXXXXXX") {
		t.Fatalf("expected truncated URL, got %q", got)
	}
}

func TestURL_userinfo(t *testing.T) {
	t.Parallel()

	got := URL("https://user:pass@example.com/hook")
	if strings.Contains(got, "user:pass") {
		t.Fatalf("expected cred redaction, got %q", got)
	}
	if !strings.Contains(got, "***@example.com") {
		t.Fatalf("got %q", got)
	}
}
