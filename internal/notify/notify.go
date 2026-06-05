// Package notify sends pipeline lifecycle events to configured outbound channels.
package notify

import (
	"context"
	"net/http"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
)

// Event names for outbound payloads.
const (
	EventStart   = "pipeline.start"
	EventSuccess = "pipeline.success"
	EventError   = "pipeline.error"
)

// HTTPDoer posts notify payloads (stdlib default or httptest in tests).
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Meta is attached to every notify payload.
type Meta struct {
	Command    string
	Mode       string
	StartedAt  time.Time
	Duration   time.Duration
	ClientID   string
	Cluster    string
	FailedStep string
	Error      string
}

// AnyEnabled reports whether at least one notify channel is enabled.
func AnyEnabled(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	n := cfg.Notify
	return n.Slack.Enabled || n.Discord.Enabled || n.Teams.Enabled ||
		n.PagerDuty.Enabled || n.Webhook.Enabled
}

// OnErrorEnabled reports whether pipeline.error should fire (default true when any channel is on).
func OnErrorEnabled(cfg *config.Config) bool {
	if !AnyEnabled(cfg) {
		return false
	}
	if cfg.Notify.OnError == nil {
		return true
	}
	return *cfg.Notify.OnError
}

// MetaFromConfig builds Meta with correlation and cluster fields from cfg.
func MetaFromConfig(cfg *config.Config, command string, started time.Time, duration time.Duration) Meta {
	m := Meta{
		Command:   command,
		Mode:      "",
		StartedAt: started,
		Duration:  duration,
	}
	if cfg != nil {
		m.Mode = cfg.Run.Mode
		m.ClientID = correlation.ClientID(cfg)
		m.Cluster = cfg.Cluster.Name
	}
	return m
}

// Dispatch sends event to all enabled channels. Errors from individual channels are joined;
// secrets in error text are redacted. No-op when no channel is enabled or mode is not live.
func Dispatch(ctx context.Context, cfg *config.Config, event string, meta Meta, client HTTPDoer) error {
	if cfg == nil || !AnyEnabled(cfg) || meta.Mode != "live" {
		return nil
	}
	if event == EventError && !OnErrorEnabled(cfg) {
		return nil
	}
	if client == nil {
		client = http.DefaultClient
	}
	return joinErrors(dispatchChannels(ctx, client, cfg.Notify, buildPayload(event, meta)))
}

func dispatchChannels(ctx context.Context, client HTTPDoer, n config.NotifyConfig, body payload) []error {
	var errs []error
	if n.Slack.Enabled {
		if err := postSlack(ctx, client, n.Slack.WebhookURL, body); err != nil {
			errs = append(errs, err)
		}
	}
	if n.Discord.Enabled {
		if err := postDiscord(ctx, client, n.Discord.WebhookURL, body); err != nil {
			errs = append(errs, err)
		}
	}
	if n.Teams.Enabled {
		if err := postTeams(ctx, client, n.Teams.WebhookURL, body); err != nil {
			errs = append(errs, err)
		}
	}
	if n.PagerDuty.Enabled {
		if err := postPagerDuty(ctx, client, n.PagerDuty.RoutingKey, body); err != nil {
			errs = append(errs, err)
		}
	}
	if n.Webhook.Enabled {
		if err := postWebhook(ctx, client, n.Webhook.URL, n.Webhook.Headers, body); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
