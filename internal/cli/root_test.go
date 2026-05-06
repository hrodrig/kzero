package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootCommand_HasExpectedSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	expected := []string{"analyze", "down", "up", "reset"}
	for _, name := range expected {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("expected subcommand %q to exist: %v", name, err)
		}
	}
}

func TestAnalyze_InvalidConfigExitCode(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - "not-a-valid-step"
run:
  mode: "dry-run"
`), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"analyze", "--config", cfgPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected analyze to fail with invalid config")
	}
}
