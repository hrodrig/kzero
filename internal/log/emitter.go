package log

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
	"github.com/hrodrig/kzero/internal/redact"
)

// Kind labels engine and CLI summary events.
type Kind string

const (
	KindLive           Kind = "live"
	KindDryRun         Kind = "dry-run"
	KindRetry          Kind = "retry"
	KindCommandSummary Kind = "command.summary"
)

// Entry is one log event (text or JSON line).
type Entry struct {
	Kind       Kind
	Level      Level
	Msg        string
	ClientID   string
	Command    string
	Phase      string
	StepIndex  int
	Ref        string
	Hook       string
	Script     string
	Err        string
	Attempt    int
	MaxAttempt int
	Wait       time.Duration
	Outcome    string
	Duration   time.Duration
	HasStep    bool
}

// Emitter writes structured pipeline logs.
type Emitter struct {
	w       io.Writer
	format  Format
	command string
	lineW   *linePrefixWriter
}

// New builds an Emitter. command is optional default for Entry.Command (set via SetCommand).
func New(w io.Writer, format Format) *Emitter {
	if w == nil {
		w = io.Discard
	}
	return &Emitter{w: w, format: format}
}

// SetCommand sets the CLI command name attached to subsequent entries (down, up, reset).
func (e *Emitter) SetCommand(command string) {
	if e != nil {
		e.command = command
	}
}

// Writer returns the underlying output stream (e.g. kubectl subprocess stdout).
func (e *Emitter) Writer() io.Writer {
	if e == nil {
		return io.Discard
	}
	if e.format == FormatJSON {
		return e.w
	}
	if e.lineW == nil {
		e.lineW = newLinePrefixWriter(e.w, LevelInfo)
	}
	return e.lineW
}

// FlushSubprocessOutput emits any buffered subprocess line without a trailing newline.
func (e *Emitter) FlushSubprocessOutput() {
	if e != nil && e.lineW != nil {
		_ = e.lineW.flush()
	}
}

// Format returns the active output format.
func (e *Emitter) Format() Format {
	if e == nil {
		return FormatText
	}
	return e.format
}

// Emit writes one event.
func (e *Emitter) Emit(entry Entry) {
	if e == nil || e.w == nil {
		return
	}
	entry.Msg = redact.String(entry.Msg)
	entry.Err = redact.String(entry.Err)
	if entry.Command == "" {
		entry.Command = e.command
	}
	if entry.ClientID == "" && entry.Kind != KindLive {
		// live lines omit client_id by contract; other kinds may still carry it when set on entry
	}
	switch e.format {
	case FormatJSON:
		e.emitJSON(entry)
	default:
		e.emitText(entry)
	}
}

// Live writes a [live] action line (no client_id prefix).
func (e *Emitter) Live(msg string) {
	e.Emit(Entry{Kind: KindLive, Msg: msg})
}

// DryRun writes a [dry-run] line with optional client_id from cfg.
func (e *Emitter) DryRun(cfg *config.Config, msg string) {
	e.Emit(Entry{Kind: KindDryRun, Msg: msg, ClientID: correlation.ClientID(cfg)})
}

// Retry writes a [retry] pipeline line.
func (e *Emitter) Retry(cfg *config.Config, phase string, index int, stepRef string, try, max int, wait time.Duration, err error) {
	ref := stepRef
	if ref == "" {
		ref = fmt.Sprintf("index %d", index)
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	e.Emit(Entry{
		Kind:       KindRetry,
		Msg:        fmt.Sprintf("pipeline %s step %s attempt %d/%d failed (%s); retrying in %s", phase, ref, try, max, errStr, wait.Round(time.Millisecond)),
		ClientID:   correlation.ClientID(cfg),
		Phase:      phase,
		StepIndex:  index,
		Ref:        stepRef,
		Err:        errStr,
		Attempt:    try,
		MaxAttempt: max,
		Wait:       wait,
		HasStep:    true,
	})
}

// CommandSummary writes the timing line (stderr); JSON when format is json.
func (e *Emitter) CommandSummary(command string, elapsed time.Duration, failed bool) {
	outcome := "success"
	if failed {
		outcome = "failed"
	}
	msg := fmt.Sprintf("kzero %s finished in %s", command, elapsed.Round(time.Millisecond))
	if failed {
		msg = fmt.Sprintf("kzero %s failed after %s", command, elapsed.Round(time.Millisecond))
	}
	e.Emit(Entry{
		Kind:     KindCommandSummary,
		Msg:      msg,
		Command:  command,
		Outcome:  outcome,
		Duration: elapsed,
	})
}

func (e *Emitter) emitText(entry Entry) {
	line := textLine(entry)
	if line == "" {
		return
	}
	level := entryLevel(entry)
	if !level.Enabled() {
		return
	}
	_, _ = fmt.Fprintln(e.w, PrefixText(level, line))
}

func textLine(entry Entry) string {
	switch entry.Kind {
	case KindLive:
		return fmt.Sprintf("[live] %s", entry.Msg)
	case KindDryRun:
		return fmt.Sprintf("[dry-run] %s%s", clientPrefix(entry.ClientID), entry.Msg)
	case KindRetry:
		return fmt.Sprintf("[retry] %s%s", clientPrefix(entry.ClientID), entry.Msg)
	case KindCommandSummary:
		return entry.Msg
	default:
		return entry.Msg
	}
}

func clientPrefix(clientID string) string {
	if clientID == "" {
		return ""
	}
	return "client_id=" + quoteIfNeeded(clientID) + " "
}

func quoteIfNeeded(s string) string {
	for _, r := range s {
		if r == ' ' || r == '=' || r == '"' {
			return fmt.Sprintf("%q", s)
		}
	}
	return s
}

type jsonRecord struct {
	Time       string `json:"time"`
	App        string `json:"app"`
	Level      string `json:"level"`
	Kind       string `json:"kind"`
	Msg        string `json:"msg"`
	ClientID   string `json:"client_id,omitempty"`
	Command    string `json:"command,omitempty"`
	Phase      string `json:"phase,omitempty"`
	StepIndex  *int   `json:"step_index,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Hook       string `json:"hook,omitempty"`
	Script     string `json:"script,omitempty"`
	Err        string `json:"error,omitempty"`
	Attempt    int    `json:"attempt,omitempty"`
	MaxAttempt int    `json:"max_attempt,omitempty"`
	Wait       string `json:"wait,omitempty"`
	Outcome    string `json:"outcome,omitempty"`
	Duration   string `json:"duration,omitempty"`
}

func (e *Emitter) emitJSON(entry Entry) {
	level := entryLevel(entry)
	if !level.Enabled() {
		return
	}
	rec := jsonRecord{
		Time:       time.Now().UTC().Format(time.RFC3339Nano),
		App:        AppName,
		Level:      level.Tag(),
		Kind:       string(entry.Kind),
		Msg:        entry.Msg,
		ClientID:   entry.ClientID,
		Command:    entry.Command,
		Phase:      entry.Phase,
		Ref:        entry.Ref,
		Hook:       entry.Hook,
		Script:     entry.Script,
		Err:        entry.Err,
		Attempt:    entry.Attempt,
		MaxAttempt: entry.MaxAttempt,
		Outcome:    entry.Outcome,
	}
	if entry.Kind == KindLive {
		rec.ClientID = ""
	}
	if entry.HasStep {
		idx := entry.StepIndex
		rec.StepIndex = &idx
	}
	if entry.Wait > 0 {
		rec.Wait = entry.Wait.Round(time.Millisecond).String()
	}
	if entry.Duration > 0 {
		rec.Duration = entry.Duration.Round(time.Millisecond).String()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		_, _ = fmt.Fprintf(e.w, "[log] encode error: %v\n", err)
		return
	}
	_, _ = fmt.Fprintln(e.w, string(data))
}
