package config

import (
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
