package engine

import (
	"context"
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
)

// RecordingRunner records calls and can inject hook or step failures for tests.
type RecordingRunner struct {
	Calls   []RecordedCall
	HookErr map[string]error
	StepErr map[string]error
}

// RecordedCall is one hook or pipeline invocation observed in tests.
type RecordedCall struct {
	Kind  string // hook | step
	Label string
	Phase Phase
	Index int
	Path  string
	Step  config.PipelineStep
}

// RunHook implements Runner.
func (r *RecordingRunner) RunHook(ctx context.Context, cfg *config.Config, label, scriptPath string) error {
	if scriptPath == "" {
		return nil
	}
	r.Calls = append(r.Calls, RecordedCall{Kind: "hook", Label: label, Path: scriptPath})
	if err, ok := r.HookErr[label]; ok && err != nil {
		return err
	}
	return nil
}

// RunPipelineStep implements Runner.
func (r *RecordingRunner) RunPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	if err := r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "pre"), step.PreStep); err != nil {
		return err
	}
	r.Calls = append(r.Calls, RecordedCall{Kind: "step", Phase: phase, Index: index, Step: step})
	key := fmt.Sprintf("%s:%d", phase, index)
	if err, ok := r.StepErr[key]; ok && err != nil {
		return err
	}
	return r.RunHook(ctx, cfg, pipelineStepHookLabel(phase, index, "post"), step.PostStep)
}
