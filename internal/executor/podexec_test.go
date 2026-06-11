package executor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestWrapPodExec_notFound(t *testing.T) {
	t.Parallel()

	step := config.PipelineStep{
		Ref: "exec.db/pg-0", Type: "exec", Namespace: "db", Name: "pg-0",
		Container: "postgres", Command: []string{"psql", "-c", "select 1"},
	}
	err := WrapPodExec(step, nil, []byte("pods \"pg-0\" not found"), errors.New("NotFound"))
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWrapPodExec_forbidden(t *testing.T) {
	t.Parallel()

	step := config.PipelineStep{Ref: "exec.ns/p", Namespace: "ns", Name: "p", Container: "c", Command: []string{"id"}}
	err := WrapPodExec(step, nil, []byte("Forbidden"), errors.New("403"))
	if err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestWrapPodExec_genericExit(t *testing.T) {
	t.Parallel()

	step := config.PipelineStep{Ref: "exec.ns/p", Namespace: "ns", Name: "p", Container: "c", Command: []string{"false"}}
	sub := &PodExecError{Ref: "exec", ExitCode: 1, Output: "fail", Err: errors.New("exit 1")}
	err := WrapPodExec(step, []byte("fail"), nil, sub)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPodExecError_stringAndUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("root")
	e := &PodExecError{Ref: "exec ns/p", ExitCode: 2, Output: "oops", Err: cause}
	if !strings.Contains(e.Error(), "exit 2") {
		t.Fatalf("got %q", e.Error())
	}
	if !errors.Is(e, cause) {
		t.Fatal("Unwrap")
	}
	e2 := &PodExecError{Ref: "exec", ExitCode: -1, Err: cause}
	if !strings.Contains(e2.Error(), "failed:") {
		t.Fatalf("got %q", e2.Error())
	}
}

func TestFormatExecPlan_truncatesLongCommand(t *testing.T) {
	t.Parallel()

	step := config.PipelineStep{
		Namespace: "db", Name: "pg-0", Container: "postgres",
		Command: []string{strings.Repeat("x", 100)},
	}
	got := FormatExecPlan(step)
	if len(got) > 120 {
		t.Fatalf("expected truncation, got len=%d", len(got))
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("expected ellipsis in %q", got)
	}
}

func TestNewPodExec_resolvesKubeconfig(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	p, err := NewPodExec(kc)
	if err != nil {
		t.Fatalf("NewPodExec: %v", err)
	}
	if p.client == nil || p.config == nil {
		t.Fatal("expected client and config")
	}
}

type stubPodExec struct {
	runFn func(context.Context, config.PipelineStep) ([]byte, []byte, error)
}

func (s *stubPodExec) Run(ctx context.Context, step config.PipelineStep) ([]byte, []byte, error) {
	return s.runFn(ctx, step)
}

func TestRemotePodExec_requiresContainerAndCommand(t *testing.T) {
	t.Parallel()

	r := &RemotePodExec{client: fake.NewClientset(), config: &rest.Config{Host: "https://example"}}
	_, _, err := r.Run(context.Background(), config.PipelineStep{Ref: "exec.ns/p", Namespace: "ns", Name: "p"})
	if err == nil || !strings.Contains(err.Error(), "container and command required") {
		t.Fatalf("got %v", err)
	}
}

func TestRemotePodExec_nilRunner(t *testing.T) {
	t.Parallel()

	var r *RemotePodExec
	_, _, err := r.Run(context.Background(), config.PipelineStep{Container: "c", Command: []string{"id"}})
	if err == nil || !strings.Contains(err.Error(), "nil runner") {
		t.Fatalf("got %v", err)
	}
}
