package cli

import (
	"os"
	"testing"
)

func TestColorizeElapsed_respectsNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("KZERO_COLOR", "always")
	got := colorizeElapsed("1m2s", false, "always")
	if got != "1m2s" {
		t.Fatalf("got %q", got)
	}
}

func TestColorizeElapsed_configAlways(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("KZERO_COLOR", "")
	got := colorizeElapsed("1m2s", false, "always")
	if got != ansiGreen+"1m2s"+ansiReset {
		t.Fatalf("got %q", got)
	}
	got = colorizeElapsed("500ms", true, "always")
	if got != ansiYellow+"500ms"+ansiReset {
		t.Fatalf("got %q", got)
	}
}

func TestColorizeElapsed_configNever(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("KZERO_COLOR", "always")
	got := colorizeElapsed("3s", false, "never")
	if got != "3s" {
		t.Fatalf("got %q", got)
	}
}

func TestColorEnabledFor_configNever(t *testing.T) {
	t.Setenv("KZERO_COLOR", "always")
	if colorEnabledFor("never") {
		t.Fatal("expected color disabled when run.color is never")
	}
}

func TestColorEnabledFor_forceColorAuto(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("KZERO_COLOR", "")
	t.Setenv("FORCE_COLOR", "1")
	if !colorEnabledFor("auto") {
		t.Fatal("expected color enabled with FORCE_COLOR=1")
	}
}

func TestColorizeElapsed_autoNoTTY_withoutOverride(t *testing.T) {
	if termIsTerminal() {
		t.Skip("test requires non-TTY stdout/stderr")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("KZERO_COLOR", "")
	t.Setenv("FORCE_COLOR", "")
	got := colorizeElapsed("3s", false, "auto")
	if got != "3s" {
		t.Fatalf("got %q", got)
	}
}

func termIsTerminal() bool {
	return colorEnabledFor("auto") && os.Getenv("KZERO_COLOR") == "" && os.Getenv("FORCE_COLOR") == ""
}
