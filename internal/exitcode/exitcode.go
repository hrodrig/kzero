package exitcode

import (
	"errors"
	"fmt"
)

// Exit code taxonomy (1.0.0 #42; SPEC — exit semantics).
//
// Pattern matches groot (internal/cmd/exitcode.go): RunE returns *Error;
// main maps via Of → os.Exit.
//
//	0  success
//	1  config validation / load failure (YAML, schema, flags)
//	2  Kubernetes client / API error (target, handshake, cluster validation)
//	3  pipeline / executor aborted (step, hook, watchdog stall, signal cancel)
//	4  notify delivery failed (require_delivery or notify test POST)
//
// Wrappers that only check non-zero stay compatible. Codes 2–4 are emitted
// only where the failure class is unambiguous (explicit New wrap).
const (
	Success         = 0
	ConfigError     = 1
	KubernetesError = 2
	ExecutorAborted = 3
	NotifyFailed    = 4
)

// Coder is the boundary Cobra RunE funnels coded errors through.
type Coder interface {
	error
	ExitCode() int
}

// Error wraps a cause with a stable exit code.
type Error struct {
	Code int
	Err  error
}

// New builds an Error. Cause may be nil.
func New(code int, cause error) *Error {
	return &Error{Code: code, Err: cause}
}

// Newf builds an Error from a formatted message.
func Newf(code int, format string, a ...any) *Error {
	return &Error{Code: code, Err: fmt.Errorf(format, a...)}
}

// Error implements error. Message is the cause only (no "exit code N:" prefix)
// so stderr remains operator-friendly; the process status carries the taxonomy.
func (e *Error) Error() string {
	if e == nil {
		return "<nil exit error>"
	}
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

// Unwrap exposes the wrapped error for errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.Err }

// ExitCode satisfies Coder.
func (e *Error) ExitCode() int {
	if e == nil {
		return Success
	}
	if e.Code < 0 || e.Code > 255 {
		return Success
	}
	return e.Code
}

// Of returns the exit code for err (0 for nil). Walks the chain via errors.As.
// Plain errors default to ConfigError (same as groot).
func Of(err error) int {
	if err == nil {
		return Success
	}
	var ec Coder
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return ConfigError
}

// Ensure wraps err with code unless err already carries a Coder.
func Ensure(code int, err error) error {
	if err == nil {
		return nil
	}
	var ec Coder
	if errors.As(err, &ec) {
		return err
	}
	return New(code, err)
}
