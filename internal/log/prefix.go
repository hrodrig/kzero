package log

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	// AppName is the application label on every text log line.
	AppName = "kzero"
	// TextTimestampLayout matches operator tools that prefix each line (e.g. cbctl-unma).
	TextTimestampLayout = "2006/01/02 15:04:05"
)

// PrefixText formats one line: "2006/01/02 15:04:05: kzero - [INF] - message".
func PrefixText(level Level, msg string) string {
	return fmt.Sprintf("%s: %s - [%s] - %s", time.Now().Format(TextTimestampLayout), AppName, level.Tag(), msg)
}

// WriteLine writes a single prefixed line to w when level meets the active minimum.
func WriteLine(w io.Writer, level Level, msg string) error {
	if !level.Enabled() {
		return nil
	}
	_, err := fmt.Fprintln(w, PrefixText(level, msg))
	return err
}

// linePrefixWriter prefixes each complete line written to the underlying writer.
type linePrefixWriter struct {
	w     io.Writer
	level Level
	buf   bytes.Buffer
}

func newLinePrefixWriter(w io.Writer, level Level) *linePrefixWriter {
	if w == nil {
		w = io.Discard
	}
	if level == 0 {
		level = LevelInfo
	}
	return &linePrefixWriter{w: w, level: level}
}

func (p *linePrefixWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if _, err := p.buf.Write(data); err != nil {
		return 0, err
	}
	for {
		raw := p.buf.Bytes()
		idx := bytes.IndexByte(raw, '\n')
		if idx < 0 {
			break
		}
		line := string(raw[:idx])
		p.buf.Next(idx + 1)
		if line == "" {
			if _, err := fmt.Fprintln(p.w); err != nil {
				return len(data), err
			}
			continue
		}
		if !p.level.Enabled() {
			continue
		}
		if _, err := fmt.Fprintln(p.w, PrefixText(p.level, line)); err != nil {
			return len(data), err
		}
	}
	return len(data), nil
}

// flush emits any buffered bytes without a trailing newline (subprocess tail).
func (p *linePrefixWriter) flush() error {
	if p.buf.Len() == 0 {
		return nil
	}
	line := strings.TrimSuffix(p.buf.String(), "\r")
	p.buf.Reset()
	if line == "" {
		return nil
	}
	return WriteLine(p.w, p.level, line)
}
