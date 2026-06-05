package executor

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrTransient marks subprocess failures that are likely temporary (network, API overload).
var ErrTransient = errors.New("subprocess transient failure")

// SubprocessError carries argv, exit code, and combined output from kubectl, helm, or hook scripts.
type SubprocessError struct {
	Command  string
	ExitCode int
	Output   string
	Err      error
}

func (e *SubprocessError) Error() string {
	if e.ExitCode >= 0 {
		return fmt.Sprintf("%s failed (exit %d): %s", e.Command, e.ExitCode, trimOutput(e.Output))
	}
	return fmt.Sprintf("%s failed: %v", e.Command, e.Err)
}

func (e *SubprocessError) Unwrap() error { return e.Err }

// WrapSubprocess classifies shell-path failures using exit codes and common stderr/stdout patterns.
func WrapSubprocess(argv0 string, args []string, output []byte, err error) error {
	if err == nil {
		return nil
	}
	var sub *SubprocessError
	if errors.As(err, &sub) {
		return err
	}

	cmd := formatCommand(argv0, args)
	out := string(output)
	base := &SubprocessError{
		Command:  cmd,
		ExitCode: exitCodeFrom(err),
		Output:   out,
		Err:      err,
	}

	switch classifySubprocess(out, err.Error(), base.ExitCode) {
	case classNotFound:
		return fmt.Errorf("%s: %w", cmd, joinSentinel(ErrNotFound, base))
	case classForbidden:
		return fmt.Errorf("%s: %w", cmd, joinSentinel(ErrForbidden, base))
	case classTransient:
		return fmt.Errorf("%s: %w", cmd, joinSentinel(ErrTransient, base))
	default:
		return fmt.Errorf("%s: %w", cmd, base)
	}
}

type subprocessClass int

const (
	classUnknown subprocessClass = iota
	classNotFound
	classForbidden
	classTransient
)

func classifySubprocess(output, errMsg string, exitCode int) subprocessClass {
	msg := strings.ToLower(output + "\n" + errMsg)

	for _, sub := range []string{
		"not found",
		"notfound",
		"no matching resources",
		"does not exist",
		"the server could not find the requested resource",
		"error from server (notfound)",
	} {
		if strings.Contains(msg, sub) {
			return classNotFound
		}
	}
	for _, sub := range []string{
		"forbidden",
		"unauthorized",
		"permission denied",
		"cannot get resource",
		"access denied",
		"error from server (forbidden)",
	} {
		if strings.Contains(msg, sub) {
			return classForbidden
		}
	}
	for _, sub := range []string{
		"connection refused",
		"connection reset",
		"i/o timeout",
		"tls handshake timeout",
		"deadline exceeded",
		"context deadline exceeded",
		"too many requests",
		"temporarily unavailable",
		"server timeout",
		"unable to connect",
		"network is unreachable",
		"dial tcp",
		" 503 ",
		" 429 ",
		" 504 ",
		"http2: server sent goaway",
	} {
		if strings.Contains(msg, sub) {
			return classTransient
		}
	}
	if exitCode == 137 || exitCode == 143 {
		return classTransient
	}
	return classUnknown
}

func formatCommand(argv0 string, args []string) string {
	if len(args) == 0 {
		return argv0
	}
	return argv0 + " " + strings.Join(args, " ")
}

func exitCodeFrom(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
