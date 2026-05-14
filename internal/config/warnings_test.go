package config

import (
	"reflect"
	"testing"
)

func TestDeferredFeatureWarnings_nilConfig(t *testing.T) {
	if got := DeferredFeatureWarnings(nil); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestDeferredFeatureWarnings_minimalDryRun(t *testing.T) {
	cfg := &Config{
		Run:   RunConfig{Mode: "dry-run", WorkerConcurrency: 1},
		Retry: RetryConfig{Attempts: 1},
	}
	if got := DeferredFeatureWarnings(cfg); len(got) != 0 {
		t.Fatalf("expected no warnings, got %q", got)
	}
}

func TestDeferredFeatureWarnings_allSignals(t *testing.T) {
	cfg := &Config{
		Run:   RunConfig{Mode: "dry-run", WorkerConcurrency: 4},
		Retry: RetryConfig{Attempts: 3},
		Notify: NotifyConfig{
			Slack:   ChannelConfig{Enabled: true},
			Discord: ChannelConfig{Enabled: true},
		},
	}
	got := DeferredFeatureWarnings(cfg)
	want := []string{
		"run.worker_concurrency=4 is set but the v1 engine runs pipeline steps sequentially; only one worker is used",
		"retry.attempts=3 is set but step retries are not implemented yet",
		"notify.slack.enabled is true but Slack notifications are not implemented yet",
		"notify.discord.enabled is true but Discord notifications are not implemented yet",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("warnings mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
