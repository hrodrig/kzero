package notify

import (
	"context"
	"fmt"
	"strings"
)

func postSlack(ctx context.Context, client HTTPDoer, webhookURL string, event string, meta Meta, p payload) error {
	body := buildSlackBody(event, meta, p)
	return postJSON(ctx, client, webhookURL, nil, body)
}

func postDiscord(ctx context.Context, client HTTPDoer, webhookURL string, p payload) error {
	line := summaryLine(p)
	body := map[string]string{"content": line}
	return postJSON(ctx, client, webhookURL, nil, body)
}

func postTeams(ctx context.Context, client HTTPDoer, webhookURL string, p payload) error {
	line := summaryLine(p)
	color := "0078D4"
	if p.Event == EventError {
		color = "E81123"
	}
	body := map[string]any{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    line,
		"themeColor": color,
		"title":      fmt.Sprintf("kzero %s", p.Command),
		"text":       line,
	}
	return postJSON(ctx, client, webhookURL, nil, body)
}

func postWebhook(ctx context.Context, client HTTPDoer, url string, headers map[string]string, p payload) error {
	data, err := payloadJSON(p)
	if err != nil {
		return err
	}
	return postRawJSON(ctx, client, url, headers, data)
}

func postPagerDuty(ctx context.Context, client HTTPDoer, routingKey string, p payload) error {
	key := strings.TrimSpace(routingKey)
	if key == "" {
		return fmt.Errorf("notify: pagerduty routing_key is empty")
	}
	severity := "info"
	if p.Event == EventError {
		severity = "error"
	}
	body := map[string]any{
		"routing_key":  key,
		"event_action": "trigger",
		"payload": map[string]any{
			"summary":  summaryLine(p),
			"severity": severity,
			"source":   "kzero",
			"custom_details": map[string]any{
				"event":       p.Event,
				"command":     p.Command,
				"mode":        p.Mode,
				"client_id":   p.ClientID,
				"cluster":     p.Cluster,
				"failed_step": p.FailedStep,
			},
		},
	}
	return postJSON(ctx, client, pagerDutyEventsURL, nil, body)
}

// pagerDutyEventsURL is the PagerDuty Events API v2 endpoint (overridable in tests).
var pagerDutyEventsURL = "https://events.pagerduty.com/v2/enqueue"
