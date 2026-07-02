package cli

import (
	"bytes"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestPrintDeferredSummary_empty(t *testing.T) {
	var buf bytes.Buffer
	if err := printDeferredSummary(&buf, &config.Config{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected empty output, got %q", buf.String())
	}
}

func TestPrintDeferredSummary_withWarnings(t *testing.T) {
	req := true
	cfg := &config.Config{
		Notify: config.NotifyConfig{RequireDelivery: &req},
	}
	var buf bytes.Buffer
	if err := printDeferredSummary(&buf, cfg); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected empty deferred summary for implemented require_delivery, got %q", buf.String())
	}
}

func TestWriteDeferredFeatureWarnings_emitsStderrLines(t *testing.T) {
	req := true
	cfg := &config.Config{
		Notify: config.NotifyConfig{RequireDelivery: &req},
	}
	var buf bytes.Buffer
	writeDeferredFeatureWarnings(&buf, cfg)
	if buf.Len() != 0 {
		t.Fatalf("expected no stderr warnings for implemented require_delivery, got %q", buf.String())
	}
}

func TestWriteDeferredFeatureWarnings_silentWhenNone(t *testing.T) {
	var buf bytes.Buffer
	writeDeferredFeatureWarnings(&buf, &config.Config{})
	if buf.Len() != 0 {
		t.Fatalf("expected no stderr warnings, got %q", buf.String())
	}
}
