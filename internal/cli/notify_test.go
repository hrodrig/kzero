package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
