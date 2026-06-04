package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestShell_ScaleAndWaitRollout(t *testing.T) {
	t.Parallel()

	var calls []string
	run := func(_ context.Context, argv0 string, args, env []string, _ string) ([]byte, error) {
		calls = append(calls, argv0+":"+strings.Join(args, " "))
		if !strings.Contains(strings.Join(env, "\n"), "KUBECONFIG=/tmp/kc") {
			t.Fatalf("env missing KUBECONFIG: %v", env)
		}
		return []byte("ok"), nil
	}
	var written []byte
	s := NewShell(ShellDeps{
		Kubectl:    "/bin/kubectl",
		Kubeconfig: "/tmp/kc",
		Run:        run,
		WriteOut:   func(b []byte) { written = append(written, b...) },
	})

	if err := s.Scale(context.Background(), "deployment", "ns", "app", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.WaitRollout(context.Background(), "deployment", "ns", "app", 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v", calls)
	}
	if string(written) != "okok" {
		t.Fatalf("writeOut = %q", written)
	}
}

func TestShell_UnsupportedKind(t *testing.T) {
	t.Parallel()

	s := NewShell(ShellDeps{Run: func(context.Context, string, []string, []string, string) ([]byte, error) {
		return nil, nil
	}})
	err := s.Scale(context.Background(), "daemonset", "ns", "x", 0)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("got %v", err)
	}
}

func TestShell_ScaleRunError(t *testing.T) {
	t.Parallel()

	s := NewShell(ShellDeps{
		Run: func(context.Context, string, []string, []string, string) ([]byte, error) {
			return []byte("fail"), errors.New("exit 1")
		},
	})
	err := s.Scale(context.Background(), "statefulset", "ns", "sts", 2)
	if err == nil || !strings.Contains(err.Error(), "kubectl scale") {
		t.Fatalf("got %v", err)
	}
}

func TestShellFromConfig_defaults(t *testing.T) {
	t.Parallel()

	var ran bool
	s := ShellFromConfig(&config.Config{
		Command: config.CommandConfig{Kubectl: "/opt/kubectl"},
		Run:     config.RunConfig{Kubeconfig: "/kc"},
	}, func(context.Context, string, []string, []string, string) ([]byte, error) {
		ran = true
		return nil, nil
	}, nil)
	if s.deps.Kubectl != "/opt/kubectl" || s.deps.Kubeconfig != "/kc" {
		t.Fatalf("deps = %+v", s.deps)
	}
	if err := s.Scale(context.Background(), "deployment", "ns", "a", 1); err != nil || !ran {
		t.Fatalf("scale err=%v ran=%v", err, ran)
	}
}
