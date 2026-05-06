package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func stepKey(phase Phase, index int) string {
	return fmt.Sprintf("%s:%d", phase, index)
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
