package notify

import (
	"fmt"
	"strings"
	"time"
)

// AppVersion is the release tag (e.g. v0.7.2), set via -ldflags from GNUmakefile / GoReleaser.
var AppVersion = "dev"

const notifyAppName = "kzero"

const (
	slackColorStart   = "#439FE0" // blue — pipeline.start
	slackColorSuccess = "#36a64f" // green — pipeline.success
	slackColorError   = "#E01E5A" // red — pipeline.error
	slackColorTest    = "#ECB22E" // yellow — notify.test
)

type slackAttachment struct {
	Color     string `json:"color,omitempty"`
	Title     string `json:"title,omitempty"`
	Text      string `json:"text,omitempty"`
	Footer    string `json:"footer,omitempty"`
	Timestamp int64  `json:"ts,omitempty"`
	Fallback  string `json:"fallback,omitempty"`
}

type slackBody struct {
	Attachments []slackAttachment `json:"attachments"`
}

func buildSlackBody(event string, meta Meta, body payload) slackBody {
	color, title := slackColorAndTitle(event)
	displayTime := meta.StartedAt
	if event == EventSuccess && meta.Duration > 0 {
		displayTime = meta.StartedAt.Add(meta.Duration)
	}

	var lines []string
	addField := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("• *%s:* `%s`", label, value))
	}

	addField("Cluster", meta.Cluster)
	addField("Client", meta.ClientID)
	addField("Time", displayTime.UTC().Format(time.RFC3339))
	addField("Context", meta.KubeContext)
	addField("User", meta.OSUser)
	if env := strings.TrimSpace(meta.Environment); env != "" {
		addField("Mode", strings.ToUpper(env))
	}
	if event == EventSuccess && meta.Duration > 0 {
		addField("Duration", formatHumanDuration(meta.Duration))
	}
	if event == EventError {
		addField("Failed step", meta.FailedStep)
		addField("Error", body.Error)
	}

	return slackBody{
		Attachments: []slackAttachment{{
			Color:     color,
			Title:     title,
			Text:      strings.Join(lines, "\n"),
			Footer:    slackFooter(),
			Timestamp: displayTime.Unix(),
			Fallback:  summaryLine(body),
		}},
	}
}

func slackFooter() string {
	v := strings.TrimSpace(AppVersion)
	if v == "" || v == "dev" {
		return notifyAppName
	}
	return fmt.Sprintf("%s %s", notifyAppName, v)
}

func slackColorAndTitle(event string) (color, title string) {
	switch event {
	case EventStart:
		return slackColorStart, fmt.Sprintf("🚀 %s started", notifyAppName)
	case EventSuccess:
		return slackColorSuccess, fmt.Sprintf("✅ %s completed", notifyAppName)
	case EventError:
		return slackColorError, fmt.Sprintf("❌ %s error", notifyAppName)
	case EventTest:
		return slackColorTest, fmt.Sprintf("🔔 %s test", notifyAppName)
	default:
		return slackColorTest, fmt.Sprintf("%s %s", notifyAppName, event)
	}
}

func formatHumanDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d <= 0 {
		return "0sec"
	}
	if d < time.Minute {
		return fmt.Sprintf("%dsec", int(d.Seconds()))
	}
	mins := int(d / time.Minute)
	secs := int(d/time.Second) % 60
	if secs == 0 {
		return fmt.Sprintf("%dmin", mins)
	}
	return fmt.Sprintf("%dmin %dsec", mins, secs)
}
