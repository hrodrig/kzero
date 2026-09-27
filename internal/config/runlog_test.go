package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_runLogDirAndFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: dry-run
  log_dir: "./.logs"
  log_file: "/var/log/kzero/run.log"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Run.LogDir != "./.logs" {
		t.Fatalf("LogDir = %q", cfg.Run.LogDir)
	}
	if cfg.Run.LogFile != "/var/log/kzero/run.log" {
		t.Fatalf("LogFile = %q", cfg.Run.LogFile)
	}
}

func TestLoad_runLogDirFromEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(path, []byte(`schema_version: "1.0"
pipelines:
  down: []
  up: []
run:
  mode: dry-run
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KZERO_RUN_LOG_DIR", "/tmp/kzero-logs")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Run.LogDir != "/tmp/kzero-logs" {
		t.Fatalf("LogDir from env = %q", cfg.Run.LogDir)
	}
}
