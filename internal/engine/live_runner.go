package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/log"
)

// LiveRunner executes hooks, custom scripts, kubectl scale/rollout for workloads,
// and release steps via shell helm or Helm SDK (per run.execution).
type LiveRunner struct {
	Log  *log.Emitter
	Exec LiveExec
	// Workload overrides scale/rollout backend (tests). When nil, resolved from cfg.Run.Execution
	// and cached under mu. The engine constructs one LiveRunner per invocation and runs pipeline
	// steps sequentially, so the cache is not contended across goroutines in normal use.
	Workload executor.Workload
	// Helm overrides release backend (tests). When nil, resolved from cfg and cached under mu.
	Helm executor.HelmReleases
	// PVC overrides pvc delete backend (tests). When nil, resolved from cfg and cached under mu.
	PVC executor.PVCDeleter
	// CronJob overrides cronjob suspend backend (tests).
	CronJob executor.CronJobSuspender
	// Job overrides job delete/create/wait backend (tests).
	Job executor.JobRunner
	// PodExec overrides exec backend (tests). When nil, resolved from cfg and cached under mu.
	PodExec executor.PodExec

	mu               sync.Mutex // guards cached executors below
	cachedWL         executor.Workload
	cachedWLKey      string
	cachedHelm       executor.HelmReleases
	cachedHelmKey    string
	cachedPVC        executor.PVCDeleter
	cachedPVCKey     string
	cachedCronJob    executor.CronJobSuspender
	cachedCronJobKey string
	cachedJob        executor.JobRunner
	cachedJobKey     string
	cachedPodExec    executor.PodExec
	cachedPodExecKey string
}

// RunHook implements Runner.
func (r *LiveRunner) RunHook(ctx context.Context, cfg *config.Config, label, scriptPath string) error {
	if scriptPath == "" {
		return nil
	}
	return r.execScript(ctx, cfg, label, scriptPath)
}

// RunPipelineStep implements Runner.
func (r *LiveRunner) RunPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	if err := r.runPipelineStepHook(ctx, cfg, phase, index, "pre", step.PreStep, step); err != nil {
		return err
	}
	if err := r.runMainPipelineStep(ctx, cfg, phase, index, step); err != nil {
		return err
	}
	return r.runPipelineStepHook(ctx, cfg, phase, index, "post", step.PostStep, step)
}

func (r *LiveRunner) runMainPipelineStep(ctx context.Context, cfg *config.Config, phase Phase, index int, step config.PipelineStep) error {
	if step.Custom != "" {
		label := fmt.Sprintf("pipeline-%s-%d", phase, index)
		env := r.stepHookEnv(cfg, phase, index, "main", step)
		return r.execScriptWithEnv(ctx, cfg, label, step.Custom, env)
	}
	if step.Ref == "" {
		return fmt.Errorf("live: empty pipeline step at index %d", index)
	}

	switch step.Type {
	case "deployment", "statefulset":
		return r.runScaledWorkload(ctx, cfg, phase, step)
	case "release":
		return r.runReleaseScript(ctx, cfg, phase, step)
	case "pvc":
		return r.runPVCDelete(ctx, cfg, step)
	case "cronjob":
		return r.runCronJobSuspend(ctx, cfg, phase, step)
	case "job":
		return r.runJobStep(ctx, cfg, phase, step)
	case "exec":
		return r.runPodExec(ctx, cfg, step)
	default:
		return fmt.Errorf("live: unsupported pipeline resource type %q", step.Type)
	}
}

func (r *LiveRunner) runPipelineStepHook(ctx context.Context, cfg *config.Config, phase Phase, index int, hookKind string, scriptPath string, step config.PipelineStep) error {
	if strings.TrimSpace(scriptPath) == "" {
		return nil
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	label := pipelineStepHookLabel(phase, index, hookKind)
	r.logLive("hook %s: %s", label, scriptPath)
	env := r.stepHookEnv(cfg, phase, index, hookKind, step)
	sh := executor.ShellPath(cfg)
	out, err := r.runProcess(opCtx, sh, []string{scriptPath}, env, ".")
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("%s: %w", label, executor.WrapSubprocess(sh, []string{scriptPath}, out, err))
	}
	return nil
}

func (r *LiveRunner) stepHookEnv(cfg *config.Config, phase Phase, index int, hookKind string, step config.PipelineStep) []string {
	env := r.envFor(cfg)
	env = append(env,
		"KZERO_PHASE="+string(phase),
		"KZERO_PIPELINE_STEP_INDEX="+strconv.Itoa(index),
		"KZERO_STEP_HOOK="+hookKind,
	)
	if step.Ref != "" {
		env = append(env, "KZERO_STEP_REF="+step.Ref)
	}
	if step.Custom != "" {
		env = append(env, "KZERO_STEP_CUSTOM="+step.Custom)
	}
	switch step.Type {
	case "deployment", "statefulset", "release", "pvc", "exec", "job", "cronjob":
		env = append(env, "KZERO_STEP_TYPE="+step.Type)
		env = append(env, "KZERO_STEP_NAMESPACE="+step.Namespace)
		env = append(env, "KZERO_STEP_NAME="+step.Name)
	}
	if step.Type == "release" && step.Namespace != "" && step.Name != "" {
		env = append(env,
			"KZERO_RELEASE_NAMESPACE="+step.Namespace,
			"KZERO_RELEASE_NAME="+step.Name,
		)
	}
	return env
}

func (r *LiveRunner) execScript(ctx context.Context, cfg *config.Config, label, scriptPath string) error {
	return r.execScriptWithEnv(ctx, cfg, label, scriptPath, r.envFor(cfg))
}

func (r *LiveRunner) execScriptWithEnv(ctx context.Context, cfg *config.Config, label, scriptPath string, env []string) error {
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	r.logLive("hook %s: %s", label, scriptPath)
	sh := executor.ShellPath(cfg)
	out, err := r.runProcess(opCtx, sh, []string{scriptPath}, env, ".")
	r.writeOutput(out)
	if err != nil {
		return fmt.Errorf("%s: %w", label, executor.WrapSubprocess(sh, []string{scriptPath}, out, err))
	}
	return nil
}
