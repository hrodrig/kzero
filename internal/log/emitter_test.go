package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		want Format
	}{
		{"text", FormatText},
		{"json", FormatJSON},
		{"", FormatText},
	} {
		got, err := ParseFormat(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q", tc.in, got)
		}
	}
	if _, err := ParseFormat("yaml"); err == nil {
		t.Fatal("expected error")
	}
}

func TestEmitter_textLive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatText)
	e.Live("scale deployment.ns/app -> 0 replicas")
	got := buf.String()
	if !strings.Contains(got, "[INF] - [live] scale deployment.ns/app -> 0 replicas") {
		t.Fatalf("got %q", got)
	}
}

func TestEmitter_jsonLive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatJSON)
	e.SetCommand("down")
	e.Live("hook pre-down: ./pre.sh")
	line := strings.TrimSpace(buf.String())
	if !json.Valid([]byte(line)) {
		t.Fatalf("invalid json: %q", line)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatal(err)
	}
	if m["kind"] != "live" || m["command"] != "down" {
		t.Fatalf("%v", m)
	}
	if m["app"] != AppName {
		t.Fatalf("app field: %v", m)
	}
	if m["level"] != "INF" {
		t.Fatalf("level field: %v", m)
	}
	if _, ok := m["client_id"]; ok && m["client_id"] != "" {
		t.Fatalf("live should omit client_id: %v", m)
	}
}

func TestEmitter_dryRunClientID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatText)
	cfg := &config.Config{Client: config.ClientConfig{ID: "ops-a"}}
	e.DryRun(cfg, "hook pre-down: ./pre.sh")
	if !strings.Contains(buf.String(), "client_id=ops-a") {
		t.Fatalf("got %q", buf.String())
	}
}

func TestEmitter_dryRunQuotedClientID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatText)
	cfg := &config.Config{Client: config.ClientConfig{ID: "ops team"}}
	e.DryRun(cfg, "scale deployment.ns/app -> 0 replicas")
	if !strings.Contains(buf.String(), `client_id="ops team"`) {
		t.Fatalf("got %q", buf.String())
	}
}

func TestEmitter_retryJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatJSON)
	e.Retry(nil, "down", 2, "deployment.app/api", 1, 3, 5*time.Second, errTest{})
	line := strings.TrimSpace(buf.String())
	if !json.Valid([]byte(line)) {
		t.Fatalf("%q", line)
	}
}

type errTest struct{}

func (errTest) Error() string { return "boom" }

func TestCommandSummary_textUnchanged(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatText)
	e.CommandSummary("up", 1500*time.Millisecond, false)
	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "[INF] - kzero up finished in 1.5s") {
		t.Fatalf("got %q", got)
	}
}

func TestCommandSummary_failedUsesErrorLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatText)
	e.CommandSummary("reset", 2*time.Second, true)
	got := buf.String()
	if !strings.Contains(got, "[ERR]") || !strings.Contains(got, "failed after") {
		t.Fatalf("got %q", got)
	}
}

func TestEmitter_WriterAndFlush(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatText)
	if e.Format() != FormatText {
		t.Fatalf("format: %v", e.Format())
	}
	if _, err := e.Writer().Write([]byte("line\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Writer().Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	e.FlushSubprocessOutput()
	out := buf.String()
	if !strings.Contains(out, "[INF] - line") || !strings.Contains(out, "[INF] - partial") {
		t.Fatalf("got %q", out)
	}
}

func TestEmitter_WriterJSONPassthrough(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	e := New(&buf, FormatJSON)
	if _, err := e.Writer().Write([]byte("raw\n")); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "raw\n" {
		t.Fatalf("json writer should pass through: %q", buf.String())
	}
}

func TestEmitter_nilSafe(t *testing.T) {
	t.Parallel()
	var nilE *Emitter
	if nilE.Format() != FormatText {
		t.Fatal("nil Format")
	}
	if _, err := nilE.Writer().Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	nilE.FlushSubprocessOutput()
}
