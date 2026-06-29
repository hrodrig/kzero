package notify

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hrodrig/kzero/internal/redact"
)

type payload struct {
	Event       string `json:"event"`
	Command     string `json:"command"`
	Mode        string `json:"mode"`
	ClientID    string `json:"client_id,omitempty"`
	Cluster     string `json:"cluster_name,omitempty"`
	KubeContext string `json:"kube_context,omitempty"`
	StartedAt   string `json:"started_at"`
	Duration    string `json:"duration,omitempty"`
	FailedStep  string `json:"failed_step,omitempty"`
	Error       string `json:"error,omitempty"`
}

func buildPayload(event string, meta Meta) payload {
	p := payload{
		Event:       event,
		Command:     meta.Command,
		Mode:        meta.Mode,
		ClientID:    meta.ClientID,
		Cluster:     meta.Cluster,
		KubeContext: meta.KubeContext,
		StartedAt:   meta.StartedAt.UTC().Format(time.RFC3339),
	}
	if meta.Duration > 0 {
		p.Duration = meta.Duration.Round(time.Millisecond).String()
	}
	if meta.FailedStep != "" {
		p.FailedStep = meta.FailedStep
	}
	if meta.Error != "" {
		p.Error = redact.String(meta.Error)
	}
	return p
}

func payloadJSON(p payload) ([]byte, error) {
	return json.Marshal(p)
}

func summaryLine(p payload) string {
	switch p.Event {
	case EventStart:
		return fmt.Sprintf("kzero %s started (%s)", p.Command, p.Mode)
	case EventSuccess:
		return fmt.Sprintf("kzero %s succeeded in %s", p.Command, p.Duration)
	case EventError:
		if p.FailedStep != "" {
			return fmt.Sprintf("kzero %s failed at %s: %s", p.Command, p.FailedStep, p.Error)
		}
		return fmt.Sprintf("kzero %s failed: %s", p.Command, p.Error)
	case EventStalled:
		return fmt.Sprintf("kzero %s stalled: %s", p.Command, p.Error)
	case EventTest:
		return fmt.Sprintf("kzero notify test (mode=%s)", p.Mode)
	default:
		return fmt.Sprintf("kzero %s %s", p.Command, p.Event)
	}
}
