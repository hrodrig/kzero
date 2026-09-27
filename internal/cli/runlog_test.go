package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestResolveRunLogPath_empty(t *testing.T) {
	logFileFlag = ""
	logDirFlag = ""
	path, err := resolveRunLogPath(&config.Config{}, "down")
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Fatalf("want empty path, got %q", path)
	}
}

func TestResolveRunLogPath_logFileWins(t *testing.T) {
	t.Cleanup(func() {
		logFileFlag = ""
		logDirFlag = ""
	})
	logFileFlag = "/tmp/explicit.log"
	logDirFlag = "/tmp/should-not-use"
	cfg := &config.Config{}
	cfg.Run.LogDir = "/tmp/cfg-dir"
	cfg.Run.LogFile = "/tmp/cfg-file.log"
	path, err := resolveRunLogPath(cfg, "down")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/explicit.log" {
		t.Fatalf("got %q", path)
	}
}

func TestResolveRunLogPath_configLogFile(t *testing.T) {
	t.Cleanup(func() {
		logFileFlag = ""
		logDirFlag = ""
	})
	cfg := &config.Config{}
	cfg.Run.LogFile = filepath.Join(t.TempDir(), "from-cfg.log")
	path, err := resolveRunLogPath(cfg, "up")
	if err != nil {
		t.Fatal(err)
	}
	if path != cfg.Run.LogFile {
		t.Fatalf("got %q want %q", path, cfg.Run.LogFile)
	}
}

func TestResolveRunLogPath_logDirPattern(t *testing.T) {
	t.Cleanup(func() {
		logFileFlag = ""
		logDirFlag = ""
		runLogClock = time.Now
	})
	dir := t.TempDir()
	logDirFlag = dir
	runLogClock = func() time.Time {
		return time.Date(2026, 9, 26, 15, 4, 5, 0, time.UTC)
	}
	cfg := &config.Config{}
	cfg.Run.Kubeconfig = filepath.Join(t.TempDir(), "missing-kubeconfig")
	path, err := resolveRunLogPath(cfg, "down")
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(path)
	if base != "kzero-down-unknown-20260926-150405.log" {
		t.Fatalf("unexpected name %q", base)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("dir = %q want %q", filepath.Dir(path), dir)
	}
}

func TestDown_logDirCreatesFile(t *testing.T) {
	stubClusterValidationSkipped(t)
	t.Cleanup(func() {
		logFileFlag = ""
		logDirFlag = ""
		runLogClock = time.Now
	})
	runLogClock = func() time.Time {
		return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	}

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.argocd/argocd-server
  up: []
run:
  mode: "dry-run"
  execution: shell
`), 0o600); err != nil {
		t.Fatal(err)
	}

	logDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"down", "--config", cfgPath, "--log-dir", logDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 log file, got %d (%v)", len(entries), entries)
	}
	name := entries[0].Name()
	if !strings.HasPrefix(name, "kzero-down-") || !strings.HasSuffix(name, ".log") {
		t.Fatalf("unexpected log name %q", name)
	}
	body, err := os.ReadFile(filepath.Join(logDir, name))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "log_file:") {
		t.Fatalf("expected log_file line in file, got %q", text)
	}
	if !strings.Contains(stdout.String(), "log_file:") {
		t.Fatalf("expected log_file on stdout, got %q", stdout.String())
	}
}

func TestDown_noLogFileByDefault(t *testing.T) {
	stubClusterValidationSkipped(t)
	t.Cleanup(func() {
		logFileFlag = ""
		logDirFlag = ""
	})
	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: "dry-run"
  execution: shell
`), 0o600); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"down", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(work)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			t.Fatalf("unexpected log file created: %s", e.Name())
		}
	}
}
