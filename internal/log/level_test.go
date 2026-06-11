package log

import "testing"

func TestParseLevel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		want Level
	}{
		{"", LevelInfo},
		{"info", LevelInfo},
		{"debug", LevelDebug},
		{"warn", LevelWarn},
		{"error", LevelError},
	} {
		got, err := ParseLevel(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
	if _, err := ParseLevel("trace"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLevelEnabled(t *testing.T) {
	t.Parallel()
	SetMinLevel(LevelInfo)
	if LevelDebug.Enabled() {
		t.Fatal("debug should be filtered at info")
	}
	if !LevelInfo.Enabled() || !LevelWarn.Enabled() {
		t.Fatal("info/warn should pass at info min")
	}
}

func TestLevelTag(t *testing.T) {
	t.Parallel()
	if LevelDebug.Tag() != "DBG" || LevelError.Tag() != "ERR" {
		t.Fatal("unexpected tags")
	}
}
