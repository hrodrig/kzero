package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
)

func TestExecutionMode_and_WantNative(t *testing.T) {
	t.Parallel()

	if ExecutionMode(nil) != ExecutionShell || WantNative(nil) {
		t.Fatal("nil cfg should be shell")
	}
	cfg := &config.Config{Run: config.RunConfig{Execution: "native"}}
	if ExecutionMode(cfg) != ExecutionNative || !WantNative(cfg) {
		t.Fatal("native")
	}
	cfg.Run.Execution = "auto"
	if !WantNative(cfg) {
		t.Fatal("auto wants native attempt")
	}
	cfg.Run.Execution = ""
	if ExecutionMode(cfg) != ExecutionShell || WantNative(cfg) {
		t.Fatal("empty execution -> shell")
	}
}

func TestKubectlPath(t *testing.T) {
	t.Parallel()

	if KubectlPath(nil) != "kubectl" {
		t.Fatal("nil cfg")
	}
	if KubectlPath(&config.Config{Command: config.CommandConfig{Kubectl: "/k"}}) != "/k" {
		t.Fatal("custom kubectl")
	}
}

func TestNewWorkload_shell(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Run: config.RunConfig{Execution: "shell"}}
	wl, err := NewWorkload(cfg, Deps{
		Run: func(context.Context, string, []string, []string, string) ([]byte, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := wl.(*Shell); !ok {
		t.Fatalf("got %T", wl)
	}
}

func TestNewWorkload_native_invalidKubeconfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Run: config.RunConfig{Execution: "native", Kubeconfig: "/no/such/kubeconfig"}}
	_, err := NewWorkload(cfg, Deps{})
	if err == nil || !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("got %v", err)
	}
}

func TestNewWorkload_auto_fallsBackToShell(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Config{Run: config.RunConfig{Execution: "auto", Kubeconfig: "/no/such/kubeconfig"}}
	wl, err := NewWorkload(cfg, Deps{
		Out: &buf,
		Run: func(context.Context, string, []string, []string, string) ([]byte, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := wl.(*Shell); !ok {
		t.Fatalf("expected shell fallback, got %T", wl)
	}
	if !strings.Contains(buf.String(), "execution auto") {
		t.Fatalf("notice: %q", buf.String())
	}
}

func TestNewWorkload_native_withKubeconfig(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	cfg := &config.Config{Run: config.RunConfig{Execution: "native", Kubeconfig: kc}}
	wl, err := NewWorkload(cfg, Deps{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := wl.(*Native); !ok {
		t.Fatalf("got %T", wl)
	}
}

func TestNewNativeForDryRun(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	wl, err := NewNativeForDryRun(&config.Config{Run: config.RunConfig{Kubeconfig: kc}})
	if err != nil {
		t.Fatal(err)
	}
	n, ok := wl.(*Native)
	if !ok || !n.serverSideDryRun {
		t.Fatalf("got %T dry=%v", wl, ok && n.serverSideDryRun)
	}
}

func TestNewKubernetesClient_valid(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	client, err := NewKubernetesClient(kc)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("nil client")
	}
}
