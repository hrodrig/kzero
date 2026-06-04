package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWriteCommandSummary_success(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	writeCommandSummary(&buf, "down", 12*time.Second+345*time.Millisecond, nil, "never")
	got := buf.String()
	if !strings.Contains(got, "kzero down finished in 12.345s") {
		t.Fatalf("got %q", got)
	}
}

func TestWriteCommandSummary_failure(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	writeCommandSummary(&buf, "up", 500*time.Millisecond, errors.New("boom"), "never")
	got := buf.String()
	if !strings.Contains(got, "kzero up failed after 500ms") {
		t.Fatalf("got %q", got)
	}
}
