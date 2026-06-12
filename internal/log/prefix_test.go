package log

import (
	"strings"
	"testing"
)

func TestPrefixText_format(t *testing.T) {
	t.Parallel()
	got := PrefixText(LevelInfo, "hello")
	if !strings.Contains(got, ": kzero - [INF] - hello") {
		t.Fatalf("got %q", got)
	}
	if !strings.HasPrefix(got, "20") {
		t.Fatalf("expected timestamp prefix, got %q", got)
	}
}

func TestLinePrefixWriter_multiline(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := newLinePrefixWriter(&buf, LevelInfo)
	if _, err := w.Write([]byte("line one\nline two\n")); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[INF] - line one") || !strings.Contains(out, "[INF] - line two") {
		t.Fatalf("got %q", out)
	}
}

func TestWriteLine_respectsMinLevel(t *testing.T) {
	old := MinLevel()
	t.Cleanup(func() { SetMinLevel(old) })
	SetMinLevel(LevelWarn)
	var buf strings.Builder
	if err := WriteLine(&buf, LevelInfo, "hidden"); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected filter, got %q", buf.String())
	}
}

func TestLinePrefixWriter_flushPartialLine(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := newLinePrefixWriter(&buf, LevelInfo)
	if _, err := w.Write([]byte("tail")); err != nil {
		t.Fatal(err)
	}
	if err := w.flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[INF] - tail") {
		t.Fatalf("got %q", buf.String())
	}
}

func TestLinePrefixWriter_emptyLine(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := newLinePrefixWriter(&buf, LevelInfo)
	if _, err := w.Write([]byte("\n")); err != nil {
		t.Fatal(err)
	}
	if buf.String() == "" {
		t.Fatal("expected blank line output")
	}
}

func TestMinLevel_roundTrip(t *testing.T) {
	old := MinLevel()
	t.Cleanup(func() { SetMinLevel(old) })
	SetMinLevel(LevelDebug)
	if MinLevel() != LevelDebug {
		t.Fatalf("got %v", MinLevel())
	}
}
