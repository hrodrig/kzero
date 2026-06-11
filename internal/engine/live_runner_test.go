package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/cluster"
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

func TestLiveRunner_ReleaseDownHelmUninstall(t *testing.T) {
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

	cfg.Command = config.CommandConfig{Helm: "/bin/helm"}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %v", calls)
	}
	if calls[0][0] != "/bin/helm" || calls[0][1] != "uninstall" || calls[0][2] != "prom" {
		t.Fatalf("expected helm uninstall prom, got %v", calls[0])
	}
	env := envs[0]
	if !envHas(env, "KZERO_PHASE=down") || !envHas(env, "KZERO_RELEASE_NAME=prom") || !envHas(env, "KZERO_RELEASE_NAMESPACE=monitoring") {
		t.Fatalf("missing KZERO_* env, got %v", env)
	}
}

func TestLiveRunner_ReleaseUpRunsInstallScript(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "prom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, d string) ([]byte, error) {
			calls = append(calls, append([]string{argv0}, args...))
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

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0][0] != "/bin/sh" || calls[0][1] != script || calls[0][2] != "up" {
		t.Fatalf("expected /bin/sh script up, got %v", calls)
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

func TestLiveRunner_PerStepPreRunsBeforeKubectlWithStepEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pre := filepath.Join(dir, "pre.sh")
	if err := os.WriteFile(pre, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
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
		Run:     config.RunConfig{Mode: "live", Kubeconfig: "/tmp/kube"},
		Command: config.CommandConfig{Kubectl: "/bin/kubectl"},
	}
	step := config.PipelineStep{
		Ref:       "deployment.ns/app",
		Type:      "deployment",
		Namespace: "ns",
		Name:      "app",
		PreStep:   pre,
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 2, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected pre script + kubectl, got %d: %v", len(calls), calls)
	}
	if calls[0][0] != "/bin/sh" || calls[0][1] != pre {
		t.Fatalf("expected /bin/sh pre script first, got %v", calls[0])
	}
	preEnv := envs[0]
	if !envHas(preEnv, "KZERO_PHASE=down") || !envHas(preEnv, "KZERO_PIPELINE_STEP_INDEX=2") || !envHas(preEnv, "KZERO_STEP_HOOK=pre") {
		t.Fatalf("missing KZERO_* step-hook env in pre, got %v", preEnv)
	}
	if !envHas(preEnv, "KZERO_STEP_REF=deployment.ns/app") {
		t.Fatalf("missing KZERO_STEP_REF, got %v", preEnv)
	}
	want := []string{"/bin/kubectl", "scale", "deployment/app", "-n", "ns", "--replicas", "0"}
	if !reflect.DeepEqual(calls[1], want) {
		t.Fatalf("kubectl argv\ngot:  %#v\nwant: %#v", calls[1], want)
	}
}

func TestLiveRunner_ReleasePostHookGetsReleaseEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	post := filepath.Join(dir, "post.sh")
	if err := os.WriteFile(post, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "prom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var postEnv []string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, d string) ([]byte, error) {
			if len(args) > 0 && args[0] == post {
				postEnv = env
			}
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
		PostStep:  post,
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 5, step); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"KZERO_RELEASE_NAME=prom",
		"KZERO_RELEASE_NAMESPACE=monitoring",
		"KZERO_STEP_HOOK=post",
		"KZERO_STEP_NAME=prom",
	} {
		if !envHas(postEnv, want) {
			t.Fatalf("missing %s in post hook env, got %v", want, postEnv)
		}
	}
}

func TestLiveRunner_LogsClientIDOnScale(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	r := &LiveRunner{
		Log: testEmitter(&buf),
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			return nil, nil
		},
	}
	cfg := &config.Config{
		Client:  config.ClientConfig{ID: "audit-runner"},
		Run:     config.RunConfig{Mode: "live"},
		Command: config.CommandConfig{Kubectl: "kubectl"},
	}
	step := config.PipelineStep{
		Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[live] scale deployment.ns/app -> 0 replicas") {
		t.Fatalf("expected live scale log, got %q", buf.String())
	}
	if strings.Contains(buf.String(), "client_id=") {
		t.Fatalf("live lines must not repeat client_id (see Kubernetes target block), got %q", buf.String())
	}
}

func TestLiveRunner_releaseSDKUsesInjectedHelm(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	called := false
	r := &LiveRunner{
		Log: testEmitter(&buf),
		Helm: &stubHelmReleases{
			usesSDK: true,
			upgrade: func(ctx context.Context, step config.PipelineStep) error {
				called = true
				return nil
			},
		},
	}
	cfg := &config.Config{
		Run:  config.RunConfig{Mode: "live", Execution: "native"},
		Helm: config.HelmConfig{Workspace: t.TempDir()},
	}
	step := config.PipelineStep{
		Ref: "release.mon/prom", Type: "release", Namespace: "mon", Name: "prom",
		Chart: "oci://example/prom", Version: "1.0.0",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseUp, 0, step); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected injected helm upgrade")
	}
	if !strings.Contains(buf.String(), "helm sdk upgrade --install") {
		t.Fatalf("log: %q", buf.String())
	}
}

type stubHelmReleases struct {
	usesSDK   bool
	uninstall func(context.Context, config.PipelineStep) error
	upgrade   func(context.Context, config.PipelineStep) error
}

func (s *stubHelmReleases) UsesSDK() bool { return s.usesSDK }
func (s *stubHelmReleases) Uninstall(ctx context.Context, step config.PipelineStep) error {
	if s.uninstall != nil {
		return s.uninstall(ctx, step)
	}
	return nil
}
func (s *stubHelmReleases) UpgradeInstall(ctx context.Context, step config.PipelineStep) error {
	if s.upgrade != nil {
		return s.upgrade(ctx, step)
	}
	return nil
}

func TestLiveRunner_helmForUsesCache(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			return nil, nil
		},
	}
	cfg := &config.Config{
		Run:     config.RunConfig{Mode: "live", Execution: "shell"},
		Command: config.CommandConfig{Helm: "helm"},
	}
	first, err := r.helmFor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.helmFor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("expected cached helm backend")
	}
}

func TestLiveRunner_LogsClientIDOnHelmUninstall(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	r := &LiveRunner{
		Log: testEmitter(&buf),
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			return nil, nil
		},
	}
	cfg := &config.Config{
		Client:  config.ClientConfig{ID: "audit-runner"},
		Run:     config.RunConfig{Mode: "live"},
		Command: config.CommandConfig{Helm: "helm"},
	}
	step := config.PipelineStep{
		Ref: "release.monitoring/prom", Type: "release", Namespace: "monitoring", Name: "prom",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	want := "[live] helm uninstall monitoring/prom"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("expected %q in log, got %q", want, buf.String())
	}
}

func TestLiveRunner_HookEnvIncludesClientID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var env []string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, e []string, d string) ([]byte, error) {
			env = e
			return nil, nil
		},
	}
	cfg := &config.Config{
		Client: config.ClientConfig{ID: "pilot-ns"},
		Run:    config.RunConfig{Mode: "live"},
	}
	if err := r.RunHook(context.Background(), cfg, "pre-down", script); err != nil {
		t.Fatal(err)
	}
	if !envHas(env, "KZERO_CLIENT_ID=pilot-ns") {
		t.Fatalf("missing KZERO_CLIENT_ID, got %v", env)
	}
}

func TestLiveRunner_RunHook_emptyPathNoOp(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			t.Fatal("Exec should not run for empty hook path")
			return nil, nil
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}}
	if err := r.RunHook(context.Background(), cfg, "pre-down", ""); err != nil {
		t.Fatal(err)
	}
}

func TestLiveRunner_RunHook_invokesSh(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, d string) ([]byte, error) {
			calls = append(calls, append([]string{argv0}, args...))
			return []byte("out\n"), nil
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live", Kubeconfig: "/tmp/k"}}
	if err := r.RunHook(context.Background(), cfg, "pre-down", script); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0][0] != "/bin/sh" || calls[0][1] != script {
		t.Fatalf("expected /bin/sh %s, got %v", script, calls)
	}
}

func TestLiveRunner_RunMainStep_customScript(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "custom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, d string) ([]byte, error) {
			calls = append(calls, append([]string{argv0}, args...))
			return nil, nil
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}}
	step := config.PipelineStep{Custom: script}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0][0] != "/bin/sh" || calls[0][1] != script {
		t.Fatalf("expected custom via /bin/sh, got %v", calls)
	}
}

func TestLiveRunner_RunMainStep_emptyRefErrors(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}}
	step := config.PipelineStep{Type: "deployment", Namespace: "ns", Name: "x"}

	err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 3, step)
	if err == nil || !strings.Contains(err.Error(), "empty pipeline step") {
		t.Fatalf("expected empty ref error, got %v", err)
	}
}

func TestLiveRunner_RunMainStep_unsupportedType(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}}
	step := config.PipelineStep{Ref: "job.batch/x", Type: "job", Namespace: "batch", Name: "x"}

	err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step)
	if err == nil || !strings.Contains(err.Error(), "unsupported pipeline resource type") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

// TestLiveRunner_DaemonSetUnsupported is defense in depth: config validation
// already rejects daemonset refs at load time, but a hand-built step that
// bypasses Load must also fail in live mode rather than invoke kubectl scale
// (which the API server rejects for DaemonSet — there is no /scale subresource).
func TestLiveRunner_DaemonSetUnsupported(t *testing.T) {
	t.Parallel()

	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			t.Fatalf("Exec should not run for unsupported daemonset step: %s %v", argv0, args)
			return nil, nil
		},
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}, Command: config.CommandConfig{Kubectl: "kubectl"}}
	step := config.PipelineStep{
		Ref:       "daemonset.kube-system/fluent-bit",
		Type:      "daemonset",
		Namespace: "kube-system",
		Name:      "fluent-bit",
	}

	err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step)
	if err == nil || !strings.Contains(err.Error(), "unsupported pipeline resource type") {
		t.Fatalf("expected unsupported type error for daemonset, got %v", err)
	}
}

func TestLiveRunner_noEnvPassthroughHook(t *testing.T) {
	t.Setenv("HOME", "/tmp/kzero-hook-home")
	t.Setenv("USER", "hook-user")

	dir := t.TempDir()
	pre := filepath.Join(dir, "pre.sh")
	if err := os.WriteFile(pre, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var preEnv []string
	r := &LiveRunner{
		Exec: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			if argv0 == "/bin/sh" {
				preEnv = env
			}
			return nil, nil
		},
	}
	cfg := &config.Config{
		Run:     config.RunConfig{Mode: "live", NoEnvPassthrough: true},
		Command: config.CommandConfig{Kubectl: "/bin/kubectl"},
	}
	step := config.PipelineStep{
		Ref:       "deployment.ns/app",
		Type:      "deployment",
		Namespace: "ns",
		Name:      "app",
		PreStep:   pre,
	}

	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if envHas(preEnv, "HOME=/tmp/kzero-hook-home") || envHas(preEnv, "USER=hook-user") {
		t.Fatalf("host env leaked into hook: %v", preEnv)
	}
	if !envHas(preEnv, "KZERO_PHASE=down") {
		t.Fatalf("missing KZERO_PHASE, got %v", preEnv)
	}
}

func TestLiveRunner_PVCDelete(t *testing.T) {
	t.Parallel()

	var deleted []string
	var buf strings.Builder
	r := &LiveRunner{
		PVC: &stubPVCDeleter{
			deleteFn: func(_ context.Context, ns, name string) error {
				deleted = append(deleted, ns+"/"+name)
				return nil
			},
		},
		Log: testEmitter(&buf),
	}
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}}
	step := config.PipelineStep{
		Ref: "pvc.database/data-postgresql-0", Type: "pvc",
		Namespace: "database", Name: "data-postgresql-0",
	}
	if err := r.RunPipelineStep(context.Background(), cfg, PhaseDown, 0, step); err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "database/data-postgresql-0" {
		t.Fatalf("deleted=%v", deleted)
	}
}

type stubPVCDeleter struct {
	deleteFn func(context.Context, string, string) error
}

func (s *stubPVCDeleter) Delete(ctx context.Context, namespace, name string) error {
	return s.deleteFn(ctx, namespace, name)
}

func TestLiveRunner_pvcFor_caches(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	r := &LiveRunner{}
	cfg := &config.Config{Run: config.RunConfig{Kubeconfig: kc}}
	p1, err := r.pvcFor(cfg)
	if err != nil {
		t.Fatalf("pvcFor: %v", err)
	}
	p2, err := r.pvcFor(cfg)
	if err != nil {
		t.Fatalf("pvcFor second: %v", err)
	}
	if p1 != p2 {
		t.Fatal("expected cached pvc deleter")
	}
}
