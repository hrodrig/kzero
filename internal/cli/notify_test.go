package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotifyTest_postsToWebhook(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: "dry-run"
notify:
  webhook:
    enabled: true
    url: "`+srv.URL+`"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"notify", "test", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v stderr=%q", err, stderr.String())
	}
	if posts != 1 {
		t.Fatalf("got %d posts, want 1", posts)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("notify test: sent event")) {
		t.Fatalf("stdout: %q", stdout.String())
	}
}

func TestNotifyTest_noChannelFails(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: "dry-run"
notify:
  slack:
    enabled: false
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"notify", "test", "--config", cfgPath})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when no channel enabled")
	}
}

// TestDown_logsErrOnNotifyDispatchFailure verifies PR1 #35: a notify POST
// failure during EventStart must be surfaced to stderr with [ERR] instead
// of being silently swallowed by `_ = notify.Dispatch(...)` at the call
// site. The pipeline is empty and `run.mode: live` so the engine itself
// would not fail; the only stderr noise should come from the failed POST.
func TestDown_logsErrOnNotifyDispatchFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: "live"
notify:
  webhook:
    enabled: true
    url: "`+srv.URL+`"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"down", "--config", cfgPath})

	// Engines + LiveRunner still require a reachable cluster / kube context.
	// The pipeline is empty so the preflight gate skips the API call, but
	// RunDown on live with no kubeconfig will likely error before we even
	// reach the pipeline stage. We do not assert exit code here — only that
	// the notify failure was logged. Either path (engine error or pipeline
	// completion) must produce the [ERR] line on stderr because EventStart
	// dispatch fires before the engine runs.
	_ = cmd.Execute()

	combined := stderr.String()
	if !strings.Contains(combined, "[ERR]") {
		t.Fatalf("expected stderr to contain [ERR]; got:\n%s", combined)
	}
	if !strings.Contains(combined, "notify dispatch failed") {
		t.Fatalf("expected stderr to contain \"notify dispatch failed\"; got:\n%s", combined)
	}
	// The webhook URL must be redacted in the log line.
	if strings.Contains(combined, srv.URL) {
		t.Fatalf("expected webhook URL to be redacted from stderr; got:\n%s", combined)
	}
}
