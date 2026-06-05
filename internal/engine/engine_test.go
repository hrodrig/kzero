package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/validate"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func stepKey(phase Phase, index int) string {
	return fmt.Sprintf("%s:%d", phase, index)
}

func TestNew_liveModeUsesLiveRunner(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{Mode: "live"}}
	e := New(cfg, testEmitter(io.Discard))
	if _, ok := e.Runner.(*LiveRunner); !ok {
		t.Fatalf("expected LiveRunner, got %T", e.Runner)
	}
}

func TestNew_unknownModeFallsBackToDryRunner(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{Mode: "bogus"}}
	e := New(cfg, testEmitter(io.Discard))
	if _, ok := e.Runner.(*DryRunner); !ok {
		t.Fatalf("expected DryRunner for unknown mode, got %T", e.Runner)
	}
}

func TestOnErrorHookFailureWrapsOriginalError(t *testing.T) {
	t.Parallel()

	stepFail := errors.New("step failed")
	onErrFail := errors.New("on-error hook failed")
	rec := &RecordingRunner{
		StepErr: map[string]error{stepKey(PhaseDown, 0): stepFail},
		HookErr: map[string]error{"on-error": onErrFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{OnError: "./err.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}

	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, stepFail) {
		t.Fatalf("expected wrapped chain with step error, got %v", err)
	}
	if !strings.Contains(err.Error(), "on-error hook") {
		t.Fatalf("expected on-error hook mention, got %v", err)
	}
}

func TestDownOrder_PreStepsPost(t *testing.T) {
	t.Parallel()

	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreDown:  "./pre.sh",
			PostDown: "./post.sh",
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}

	if err := eng.RunDown(context.Background(), cfg); err != nil {
		t.Fatalf("RunDown: %v", err)
	}
	if len(rec.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %#v", len(rec.Calls), rec.Calls)
	}
	assertHook(t, rec.Calls[0], "pre-down", "./pre.sh")
	assertStep(t, rec.Calls[1], PhaseDown, 0)
	assertHook(t, rec.Calls[2], "post-down", "./post.sh")
}

func TestDownOrder_PerStepPrePostAroundStep(t *testing.T) {
	t.Parallel()

	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreDown:  "./pre.sh",
			PostDown: "./post.sh",
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{
					Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app",
					PreStep: "./step-pre.sh", PostStep: "./step-post.sh",
				},
			},
		},
	}

	if err := eng.RunDown(context.Background(), cfg); err != nil {
		t.Fatalf("RunDown: %v", err)
	}
	if len(rec.Calls) != 5 {
		t.Fatalf("expected 5 calls, got %d: %#v", len(rec.Calls), rec.Calls)
	}
	assertHook(t, rec.Calls[0], "pre-down", "./pre.sh")
	assertHook(t, rec.Calls[1], pipelineStepHookLabel(PhaseDown, 0, "pre"), "./step-pre.sh")
	assertStep(t, rec.Calls[2], PhaseDown, 0)
	assertHook(t, rec.Calls[3], pipelineStepHookLabel(PhaseDown, 0, "post"), "./step-post.sh")
	assertHook(t, rec.Calls[4], "post-down", "./post.sh")
}

func TestFailureInPerStepPreHook_SkipsStepAndGlobalPost(t *testing.T) {
	t.Parallel()

	preFail := errors.New("step pre failed")
	rec := &RecordingRunner{
		HookErr: map[string]error{pipelineStepHookLabel(PhaseDown, 0, "pre"): preFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{PreDown: "./pre.sh", PostDown: "./post.sh", OnError: "./err.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a", PreStep: "./s-pre.sh"},
			},
		},
	}

	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, preFail) {
		t.Fatalf("expected step pre error, got %v", err)
	}
	joined := callLabels(rec.Calls)
	if strings.Contains(joined, "step:down:0") {
		t.Fatalf("main step should not run: %s", joined)
	}
	if strings.Contains(joined, "hook:post-down") {
		t.Fatalf("post-down must not run: %s", joined)
	}
	if !strings.Contains(joined, "hook:on-error") {
		t.Fatalf("expected on-error: %s", joined)
	}
}

func TestFailureInMainStep_SkipsPerStepPostHook(t *testing.T) {
	t.Parallel()

	stepFail := errors.New("step failed")
	rec := &RecordingRunner{
		StepErr: map[string]error{stepKey(PhaseDown, 0): stepFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{OnError: "./err.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a", PostStep: "./after.sh"},
			},
		},
	}

	_ = eng.RunDown(context.Background(), cfg)
	joined := callLabels(rec.Calls)
	if strings.Contains(joined, pipelineStepHookLabel(PhaseDown, 0, "post")) {
		t.Fatalf("per-step post must not run after step failure, got %s", joined)
	}
}

func TestUpOrder_PreStepsPost(t *testing.T) {
	t.Parallel()

	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreUp:  "./pre-up.sh",
			PostUp: "./post-up.sh",
		},
		Pipelines: config.PipelinesConfig{
			Up: []config.PipelineStep{
				{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}

	if err := eng.RunUp(context.Background(), cfg); err != nil {
		t.Fatalf("RunUp: %v", err)
	}
	if len(rec.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(rec.Calls))
	}
	assertHook(t, rec.Calls[0], "pre-up", "./pre-up.sh")
	assertStep(t, rec.Calls[1], PhaseUp, 0)
	assertHook(t, rec.Calls[2], "post-up", "./post-up.sh")
}

func TestResetOrder_DownThenUp(t *testing.T) {
	t.Parallel()

	rec := &RecordingRunner{}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreDown: "./pd.sh", PostDown: "./pod.sh",
			PreUp: "./pu.sh", PostUp: "./pou.sh",
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.a/b", Type: "deployment", Namespace: "a", Name: "b"}},
			Up:   []config.PipelineStep{{Ref: "deployment.a/b", Type: "deployment", Namespace: "a", Name: "b"}},
		},
	}

	if err := eng.RunReset(context.Background(), cfg); err != nil {
		t.Fatalf("RunReset: %v", err)
	}
	joined := callLabels(rec.Calls)
	want := "hook:pre-down|step:down:0|hook:post-down|hook:pre-up|step:up:0|hook:post-up"
	if joined != want {
		t.Fatalf("call order\ngot:  %s\nwant: %s", joined, want)
	}
}

func TestFailureInStep_TriggersOnErrorAndAbortsPhase(t *testing.T) {
	t.Parallel()

	stepFail := errors.New("step failed")
	rec := &RecordingRunner{
		StepErr: map[string]error{stepKey(PhaseDown, 0): stepFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreDown:  "./pre.sh",
			PostDown: "./post.sh",
			OnError:  "./err.sh",
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"},
				{Ref: "deployment.ns/b", Type: "deployment", Namespace: "ns", Name: "b"},
			},
		},
	}

	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, stepFail) {
		t.Fatalf("expected step error, got %v", err)
	}
	joined := callLabels(rec.Calls)
	if !strings.Contains(joined, "hook:on-error") {
		t.Fatalf("expected on-error hook, got %s", joined)
	}
	if strings.Contains(joined, "step:down:1") {
		t.Fatalf("second step should not run, got %s", joined)
	}
	if strings.Contains(joined, "hook:post-down") {
		t.Fatalf("post-down should not run after failure, got %s", joined)
	}
}

func TestFailureInPreHook_TriggersOnErrorAndSkipsPhase(t *testing.T) {
	t.Parallel()

	preFail := errors.New("pre failed")
	rec := &RecordingRunner{
		HookErr: map[string]error{"pre-down": preFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{PreDown: "./pre.sh", OnError: "./err.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}

	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, preFail) {
		t.Fatalf("expected pre error, got %v", err)
	}
	joined := callLabels(rec.Calls)
	if !strings.Contains(joined, "hook:pre-down") {
		t.Fatalf("missing pre-down: %s", joined)
	}
	if strings.Contains(joined, "step:down") {
		t.Fatalf("pipeline should not run: %s", joined)
	}
	if !strings.Contains(joined, "hook:on-error") {
		t.Fatalf("expected on-error: %s", joined)
	}
}

func TestReset_FailureInDown_SkipsUp(t *testing.T) {
	t.Parallel()

	downFail := errors.New("down step fail")
	rec := &RecordingRunner{
		StepErr: map[string]error{stepKey(PhaseDown, 0): downFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{OnError: "./err.sh"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
			Up:   []config.PipelineStep{{Ref: "deployment.ns/b", Type: "deployment", Namespace: "ns", Name: "b"}},
		},
	}

	err := eng.RunReset(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	joined := callLabels(rec.Calls)
	if strings.Contains(joined, "pre-up") || strings.Contains(joined, "step:up") {
		t.Fatalf("up phase should not run: %s", joined)
	}
}

func TestPostHookNotRunAfterPipelineFailure(t *testing.T) {
	t.Parallel()

	stepFail := errors.New("step failed")
	rec := &RecordingRunner{
		StepErr: map[string]error{stepKey(PhaseDown, 0): stepFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreDown:  "./pre.sh",
			PostDown: "./post.sh",
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}

	_ = eng.RunDown(context.Background(), cfg)
	joined := callLabels(rec.Calls)
	if strings.Contains(joined, "hook:post-down") {
		t.Fatalf("post-down must not run after pipeline failure, got %s", joined)
	}
}

func TestFailureInPostDown_InvokesOnError(t *testing.T) {
	t.Parallel()

	postFail := errors.New("post failed")
	rec := &RecordingRunner{
		HookErr: map[string]error{"post-down": postFail},
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run"},
		Hooks: config.HooksConfig{
			PreDown:  "./pre.sh",
			PostDown: "./post.sh",
			OnError:  "./err.sh",
		},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"}},
		},
	}

	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, postFail) {
		t.Fatalf("expected post-down error, got %v", err)
	}
	joined := callLabels(rec.Calls)
	if !strings.Contains(joined, "hook:post-down") {
		t.Fatalf("expected post-down attempt: %s", joined)
	}
	if !strings.Contains(joined, "hook:on-error") {
		t.Fatalf("expected on-error after post-down failure: %s", joined)
	}
}

func assertHook(t *testing.T, c RecordedCall, label, path string) {
	t.Helper()
	if c.Kind != "hook" || c.Label != label || c.Path != path {
		t.Fatalf("expected hook %q path %q, got %#v", label, path, c)
	}
}

func assertStep(t *testing.T, c RecordedCall, phase Phase, index int) {
	t.Helper()
	if c.Kind != "step" || c.Phase != phase || c.Index != index {
		t.Fatalf("expected step %s[%d], got %#v", phase, index, c)
	}
}

func stubLivePreflightOK(t *testing.T) {
	t.Helper()
	old := validate.DefaultClientFactory
	validate.DefaultClientFactory = func(string) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	}
	t.Cleanup(func() { validate.DefaultClientFactory = old })
}

func TestRunDown_retriesTransientPipelineStep(t *testing.T) {
	t.Parallel()
	stubLivePreflightOK(t)

	rec := &RecordingRunner{
		StepFailRemaining: map[string]int{stepKey(PhaseDown, 0): 1},
		StepFailErr:       executor.ErrConflict,
	}
	var logBuf strings.Builder
	eng := &Engine{Runner: rec, Log: testEmitter(&logBuf)}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "live"},
		Retry: config.RetryConfig{Attempts: 3, Delay: time.Millisecond},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"},
			},
		},
	}
	if err := eng.RunDown(context.Background(), cfg); err != nil {
		t.Fatalf("RunDown: %v", err)
	}
	stepCalls := 0
	for _, c := range rec.Calls {
		if c.Kind == "step" {
			stepCalls++
		}
	}
	if stepCalls != 2 {
		t.Fatalf("expected 2 step executions (1 fail + 1 ok), got %d calls: %#v", stepCalls, rec.Calls)
	}
	if !strings.Contains(logBuf.String(), "[retry]") {
		t.Fatalf("expected retry log, got %q", logBuf.String())
	}
}

func TestRunDown_doesNotRetryNotFound(t *testing.T) {
	t.Parallel()
	stubLivePreflightOK(t)

	rec := &RecordingRunner{
		StepFailRemaining: map[string]int{stepKey(PhaseDown, 0): 2},
		StepFailErr:       executor.ErrNotFound,
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "live"},
		Retry: config.RetryConfig{Attempts: 3, Delay: time.Millisecond},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"},
			},
		},
	}
	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, executor.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	stepCalls := 0
	for _, c := range rec.Calls {
		if c.Kind == "step" {
			stepCalls++
		}
	}
	if stepCalls != 1 {
		t.Fatalf("expected 1 step attempt, got %d", stepCalls)
	}
}

func TestRunDown_dryRunSkipsRetry(t *testing.T) {
	t.Parallel()

	rec := &RecordingRunner{
		StepFailRemaining: map[string]int{stepKey(PhaseDown, 0): 2},
		StepFailErr:       executor.ErrConflict,
	}
	eng := &Engine{Runner: rec}
	cfg := &config.Config{
		Run:   config.RunConfig{Mode: "dry-run"},
		Retry: config.RetryConfig{Attempts: 3, Delay: time.Millisecond},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Ref: "deployment.ns/a", Type: "deployment", Namespace: "ns", Name: "a"},
			},
		},
	}
	err := eng.RunDown(context.Background(), cfg)
	if err == nil || !errors.Is(err, executor.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	stepCalls := 0
	for _, c := range rec.Calls {
		if c.Kind == "step" {
			stepCalls++
		}
	}
	if stepCalls != 1 {
		t.Fatalf("dry-run should not retry, got %d step calls", stepCalls)
	}
}

func callLabels(calls []RecordedCall) string {
	var b strings.Builder
	for i, c := range calls {
		if i > 0 {
			b.WriteString("|")
		}
		switch c.Kind {
		case "hook":
			b.WriteString("hook:")
			b.WriteString(c.Label)
		case "step":
			b.WriteString("step:")
			b.WriteString(string(c.Phase))
			b.WriteString(":")
			fmt.Fprintf(&b, "%d", c.Index)
		default:
			b.WriteString(c.Kind)
		}
	}
	return b.String()
}
