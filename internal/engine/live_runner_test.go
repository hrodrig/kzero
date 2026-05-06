package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestLiveRunner_ScaleDownDeployment(t *testing.T) {
	t.Parallel()

	var lastEnv []string
	var calls [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			lastEnv = env
			calls = append(calls, append([]string{argv0}, args...))
			return nil, nil
		},
	}
	cfg := &config.Config{
		Run:     config.RunConfig{Mode: "live", Kubeconfig: "/tmp/kube"},
		Command: config.CommandConfig{Kubectl: "/bin/kubectl"},
	}
	step := config.PipelineStep{
		Ref:       "deployment.ns/app",
		Type:      "deployment",
		Namespace: "ns",
		Name:      "app",
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 kubectl call, got %d: %v", len(calls), calls)
	}
	want := []string{"/bin/kubectl", "scale", "deployment/app", "-n", "ns", "--replicas", "0"}
	if !reflect.DeepEqual(calls[0], want) {
		t.Fatalf("kubectl argv\ngot:  %#v\nwant: %#v", calls[0], want)
	}
	if !envHas(lastEnv, "KUBECONFIG=/tmp/kube") {
		t.Fatalf("missing KUBECONFIG in env")
	}
}

func TestLiveRunner_ScaleUpDefaultReplicas(t *testing.T) {
	t.Parallel()

	var calls [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			calls = append(calls, append([]string{argv0}, args...))
			return nil, nil
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}, Command: config.CommandConfig{Kubectl: "kubectl"}}
	step := config.PipelineStep{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 0, step); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls[0], " "); !strings.Contains(got, "--replicas 1") {
		t.Fatalf("expected default replicas 1, got %q", got)
	}
}

func TestLiveRunner_RolloutStatusWhenWaitForReady(t *testing.T) {
	t.Parallel()

	var calls [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			calls = append(calls, append([]string{argv0}, args...))
			return nil, nil
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}, Command: config.CommandConfig{Kubectl: "kubectl"}}
	three := 3
	step := config.PipelineStep{
		Ref:          "statefulset.db/pg",
		Type:         "statefulset",
		Namespace:    "db",
		Name:         "pg",
		Replicas:     &three,
		WaitForReady: true,
		Timeout:      2 * time.Minute,
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected scale + rollout, got %d calls: %v", len(calls), calls)
	}
	roll := calls[1]
	if roll[0] != "kubectl" || roll[1] != "rollout" || roll[2] != "status" {
		t.Fatalf("expected rollout status, got %v", roll)
	}
	if !rolloutHasTimeout(roll, "2m") {
		t.Fatalf("expected rollout --timeout ~2m, got %v", roll)
	}
}

func rolloutHasTimeout(roll []string, prefix string) bool {
	for i := 0; i < len(roll)-1; i++ {
		if roll[i] == "--timeout" && strings.HasPrefix(roll[i+1], prefix) {
			return true
		}
	}
	return false
}

func TestLiveRunner_ReleaseScriptInvokesShWithPhase(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "prom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	var envs [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, d string) ([]byte, error) {
			calls = append(calls, append([]string{argv0}, args...))
			envs = append(envs, env)
			return nil, nil
		},
	}
	cfg := &config.Config{
		Run:  config.RunConfig{Mode: "live"},
		Helm: config.HelmConfig{Workspace: dir},
	}
	step := config.PipelineStep{
		Ref:       "release.monitoring/prom",
		Type:      "release",
		Namespace: "monitoring",
		Name:      "prom",
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %v", calls)
	}
	if calls[0][0] != "/bin/sh" || calls[0][1] != script || calls[0][2] != "down" {
		t.Fatalf("expected /bin/sh script down, got %v", calls[0])
	}
	env := envs[0]
	if !envHas(env, "KZERO_PHASE=down") || !envHas(env, "KZERO_RELEASE_NAME=prom") || !envHas(env, "KZERO_RELEASE_NAMESPACE=monitoring") {
		t.Fatalf("missing KZERO_* env, got %v", env)
	}
}

func envHas(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
