package executor

import (
	"errors"
	"os/exec"
	"runtime"
	"testing"
)

func TestWrapSubprocess_classifiesNotFound(t *testing.T) {
	t.Parallel()

	err := WrapSubprocess("/bin/kubectl", []string{"get", "pod", "x"}, []byte(`error: the server could not find the requested resource`), errors.New("exit status 1"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestWrapSubprocess_classifiesForbidden(t *testing.T) {
	t.Parallel()

	err := WrapSubprocess("kubectl", []string{"scale", "deployment/app"}, []byte("Error from server (Forbidden): user cannot get resource"), errors.New("exit status 1"))
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestWrapSubprocess_classifiesTransient(t *testing.T) {
	t.Parallel()

	for _, msg := range []string{
		"dial tcp: connection refused",
		"http2: client connection lost",
		"Unexpected error when reading response body: connection lost",
	} {
		err := WrapSubprocess("helm", []string{"upgrade"}, []byte(msg), errors.New("exit status 1"))
		if !errors.Is(err, ErrTransient) {
			t.Fatalf("msg %q: got %v", msg, err)
		}
	}
}

func TestWrapSubprocess_idempotent(t *testing.T) {
	t.Parallel()

	once := WrapSubprocess("kubectl", nil, []byte("not found"), errors.New("exit status 1"))
	twice := WrapSubprocess("kubectl", nil, []byte("not found"), once)
	if !errors.Is(twice, ErrNotFound) {
		t.Fatalf("got %v", twice)
	}
}

func TestWrapSubprocess_nil(t *testing.T) {
	t.Parallel()
	if WrapSubprocess("kubectl", nil, nil, nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestWrapSubprocess_exitCodeFromCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exit code probe uses sh -c false")
	}
	t.Parallel()

	cmd := exec.Command("sh", "-c", "exit 7")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected failure")
	}
	wrapped := WrapSubprocess("sh", []string{"-c", "exit 7"}, out, err)
	var sub *SubprocessError
	if !errors.As(wrapped, &sub) {
		t.Fatalf("expected SubprocessError, got %T %v", wrapped, wrapped)
	}
	if sub.ExitCode != 7 {
		t.Fatalf("exit code %d want 7", sub.ExitCode)
	}
}

func TestClassifySubprocess_exit137Transient(t *testing.T) {
	t.Parallel()
	if classifySubprocess("", "signal killed", 137) != classTransient {
		t.Fatal("expected transient for 137")
	}
}
