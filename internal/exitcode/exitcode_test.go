package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestOf_nil(t *testing.T) {
	if got := Of(nil); got != Success {
		t.Fatalf("nil err => %d, want %d", got, Success)
	}
}

func TestOf_plainError(t *testing.T) {
	if got := Of(errors.New("boom")); got != ConfigError {
		t.Fatalf("plain err => %d, want %d", got, ConfigError)
	}
}

func TestOf_coded(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"config", New(ConfigError, errors.New("bad yaml")), ConfigError},
		{"kubernetes", New(KubernetesError, errors.New("client")), KubernetesError},
		{"executor", New(ExecutorAborted, errors.New("step")), ExecutorAborted},
		{"notify", New(NotifyFailed, errors.New("slack 500")), NotifyFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Of(tc.err); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestOf_wrapped(t *testing.T) {
	base := New(KubernetesError, errors.New("list namespaces"))
	wrapped := fmt.Errorf("analyze: %w", base)
	if got := Of(wrapped); got != KubernetesError {
		t.Fatalf("wrapped exit error must surface its code; got %d", got)
	}
}

func TestEnsure_preservesCoded(t *testing.T) {
	inner := New(NotifyFailed, errors.New("webhook"))
	got := Ensure(ExecutorAborted, inner)
	if Of(got) != NotifyFailed {
		t.Fatalf("Ensure must not overwrite coded error; got %d", Of(got))
	}
}

func TestEnsure_wrapsPlain(t *testing.T) {
	got := Ensure(ExecutorAborted, errors.New("step failed"))
	if Of(got) != ExecutorAborted {
		t.Fatalf("got %d want %d", Of(got), ExecutorAborted)
	}
}

func TestError_messageIsCause(t *testing.T) {
	err := New(KubernetesError, errors.New("kubernetes target: no config"))
	if got := err.Error(); got != "kubernetes target: no config" {
		t.Fatalf("Error() should be cause only, got %q", got)
	}
}

func TestError_clampsOutOfRange(t *testing.T) {
	e := &Error{Code: 999, Err: errors.New("oob")}
	if got := e.ExitCode(); got != Success {
		t.Fatalf("out-of-range code must clamp to 0; got %d", got)
	}
	e = &Error{Code: -3, Err: errors.New("neg")}
	if got := e.ExitCode(); got != Success {
		t.Fatalf("negative code must clamp to 0; got %d", got)
	}
}

func TestNewf_preservesCode(t *testing.T) {
	err := Newf(NotifyFailed, "send slack: %d", 500)
	if got := Of(err); got != NotifyFailed {
		t.Fatalf("got %d want %d", got, NotifyFailed)
	}
}
