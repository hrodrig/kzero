package cli

import (
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
)

// colorEnabledFor decides whether to ANSI-color the elapsed-time summary.
// configMode comes from run.color (auto, always, never). NO_COLOR always wins.
// Legacy KZERO_COLOR env is honored when configMode is auto.
func colorEnabledFor(configMode string) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(configMode))
	if mode == "" {
		mode = "auto"
	}
	switch mode {
	case "never":
		return false
	case "always":
		return true
	}
	// auto: legacy env overrides before TTY detection
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KZERO_COLOR"))) {
	case "always", "1", "true":
		return true
	case "never", "0", "false":
		return false
	}
	if v := os.Getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	return term.IsTerminal(int(os.Stdout.Fd())) || term.IsTerminal(int(os.Stderr.Fd()))
}

func colorizeElapsed(elapsed string, failed bool, configMode string) string {
	if !colorEnabledFor(configMode) {
		return elapsed
	}
	if failed {
		return ansiYellow + elapsed + ansiReset
	}
	return ansiGreen + elapsed + ansiReset
}
