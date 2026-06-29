package config

import (
	"strings"
	"testing"
)

func TestDeferredFeatureWarnings_nilConfig(t *testing.T) {
	if got := DeferredFeatureWarnings(nil); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestDeferredFeatureWarnings_minimalDryRun(t *testing.T) {
	cfg := &Config{
		Run:   RunConfig{Mode: "dry-run"},
		Retry: RetryConfig{Attempts: 1},
	}
	if got := DeferredFeatureWarnings(cfg); len(got) != 0 {
		t.Fatalf("expected no warnings, got %q", got)
	}
}

func TestDeferredFeatureWarnings_notifyEnabledNoWarning(t *testing.T) {
	cfg := &Config{
		Run: RunConfig{Mode: "dry-run"},
		Notify: NotifyConfig{
			Slack:   ChannelConfig{Enabled: true},
			Discord: ChannelConfig{Enabled: true},
			Teams:   ChannelConfig{Enabled: true},
		},
	}
	if got := DeferredFeatureWarnings(cfg); len(got) != 0 {
		t.Fatalf("expected no warnings for implemented notify channels, got %q", got)
	}
}

func TestDeferredFeatureWarnings_apiWatchdogEnabledNoWarning(t *testing.T) {
	cfg := &Config{
		Run: RunConfig{
			Mode:        "live",
			APIWatchdog: &APIWatchdogConfig{Enabled: true, Interval: 60_000_000_000, FailAfter: 300_000_000_000},
		},
	}
	if got := DeferredFeatureWarnings(cfg); len(got) != 0 {
		t.Fatalf("expected no deferred warning for implemented api_watchdog, got %q", got)
	}
}

func TestDeferredFeatureWarnings_apiWatchdogDisabledSilent(t *testing.T) {
	cfg := &Config{
		Run: RunConfig{
			Mode:        "live",
			APIWatchdog: &APIWatchdogConfig{Enabled: false},
		},
	}
	if got := DeferredFeatureWarnings(cfg); len(got) != 0 {
		t.Fatalf("expected no warnings when api_watchdog.enabled=false, got %q", got)
	}
}

func TestDeferredFeatureWarnings_notifyRequireDelivery_warns(t *testing.T) {
	req := true
	cfg := &Config{
		Run:    RunConfig{Mode: "live"},
		Notify: NotifyConfig{RequireDelivery: &req},
	}
	got := DeferredFeatureWarnings(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one deferred warning, got %d: %v", len(got), got)
	}
	if !strings.Contains(got[0], "notify.require_delivery") {
		t.Fatalf("warning missing key: %q", got[0])
	}
}

func TestDeferredFeatureWarnings_apiWatchdogAndRequireDelivery_onlyRequireDeliveryWarns(t *testing.T) {
	req := true
	cfg := &Config{
		Run: RunConfig{
			Mode:        "live",
			APIWatchdog: &APIWatchdogConfig{Enabled: true, Interval: 60_000_000_000, FailAfter: 300_000_000_000},
		},
		Notify: NotifyConfig{RequireDelivery: &req},
	}
	got := DeferredFeatureWarnings(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one deferred warning, got %d: %v", len(got), got)
	}
	if !strings.Contains(got[0], "notify.require_delivery") {
		t.Fatalf("expected require_delivery warning only, got %q", got[0])
	}
}

func TestDeferredFeatureWarnings_notifyRequireDeliveryFalseSilent(t *testing.T) {
	req := false
	cfg := &Config{
		Run:    RunConfig{Mode: "live"},
		Notify: NotifyConfig{RequireDelivery: &req},
	}
	if got := DeferredFeatureWarnings(cfg); len(got) != 0 {
		t.Fatalf("expected no warnings when notify.require_delivery=false, got %q", got)
	}
}
