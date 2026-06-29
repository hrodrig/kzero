package cli

import (
	"bytes"
	"strings"
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
	out := buf.String()
	if !strings.Contains(out, "Deferred (accepted by schema; not implemented by v1 engine):") {
		t.Fatalf("missing deferred header: %q", out)
	}
	if !strings.Contains(out, "notify.require_delivery") {
		t.Fatalf("missing require_delivery warning: %q", out)
	}
}

func TestWriteDeferredFeatureWarnings_emitsStderrLines(t *testing.T) {
	req := true
	cfg := &config.Config{
		Notify: config.NotifyConfig{RequireDelivery: &req},
	}
	var buf bytes.Buffer
	writeDeferredFeatureWarnings(&buf, cfg)
	out := buf.String()
	if !strings.Contains(out, "warning: notify.require_delivery") {
		t.Fatalf("missing stderr warning line: %q", out)
	}
}

func TestWriteDeferredFeatureWarnings_silentWhenNone(t *testing.T) {
	var buf bytes.Buffer
	writeDeferredFeatureWarnings(&buf, &config.Config{})
	if buf.Len() != 0 {
		t.Fatalf("expected no stderr warnings, got %q", buf.String())
	}
}
