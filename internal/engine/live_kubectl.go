package engine

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/redact"
	"github.com/hrodrig/kzero/internal/subprocess"
)

// LiveExec runs argv0 with args; env is the full environment; dir is the working directory.
// Used by tests to stub process execution. If nil on LiveRunner, os/exec is used.
type LiveExec func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error)

var scalableKinds = map[string]struct{}{
	"deployment":  {},
	"statefulset": {},
}

func kubectlPath(cfg *config.Config) string {
	if p := strings.TrimSpace(cfg.Command.Kubectl); p != "" {
		return p
	}
	return "kubectl"
}

func (r *LiveRunner) envFor(cfg *config.Config) []string {
	return subprocess.Env(cfg)
}

func (r *LiveRunner) runProcess(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
	if r.Exec != nil {
		return r.Exec(ctx, argv0, args, env, dir)
	}
	return defaultExecCombined(ctx, argv0, args, env, dir)
}

func (r *LiveRunner) writeOutput(out []byte) {
	if len(out) > 0 && r.Log != nil {
		_, _ = r.Log.Writer().Write([]byte(redact.String(string(out))))
		r.Log.FlushSubprocessOutput()
	}
}

func (r *LiveRunner) runScaledWorkload(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	if _, ok := scalableKinds[step.Type]; !ok {
		return fmt.Errorf("live: unsupported resource kind %q for scale", step.Type)
	}

	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	wl, err := r.workloadFor(cfg)
	if err != nil {
		return err
	}

	replicas := int32(scaleReplicas(phase, step))
	r.logLive("scale %s -> %d replicas", step.Ref, replicas)
	if err := wl.Scale(opCtx, step.Type, step.Namespace, step.Name, replicas); err != nil {
		return err
	}
	if phase == PhaseUp && step.WaitForReady {
		timeout := rolloutTimeout(cfg, step)
		r.logLive("wait rollout %s (timeout %s)", step.Ref, timeout)
		return r.withThrottledProgress(ctx, step, "waiting rollout", func(_ context.Context) error {
			return wl.WaitRollout(opCtx, step.Type, step.Namespace, step.Name, timeout)
		})
	}
	return nil
}

func (r *LiveRunner) workloadFor(cfg *config.Config) (executor.Workload, error) {
	if r.Workload != nil {
		return r.Workload, nil
	}
	key := cfg.Run.Kubeconfig + "|" + cfg.Run.Execution + "|" + cfg.Command.Kubectl
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedWL != nil && r.cachedWLKey == key {
		return r.cachedWL, nil
	}
	var rawOut io.Writer
	if r.Log != nil {
		rawOut = r.Log.Writer()
	}
	wl, err := executor.NewWorkload(cfg, executor.Deps{
		Run:      r.runProcess,
		WriteOut: r.writeOutput,
		Out:      rawOut,
	})
	if err != nil {
		return nil, err
	}
	r.cachedWL = wl
	r.cachedWLKey = key
	return wl, nil
}

func (r *LiveRunner) runPVCDelete(ctx context.Context, cfg *config.Config, step config.PipelineStep) error {
	pvc, err := r.pvcFor(cfg)
	if err != nil {
		return err
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	r.logLive("delete pvc %s/%s (background propagation, ignore-not-found)", step.Namespace, step.Name)
	return pvc.Delete(opCtx, step.Namespace, step.Name)
}

func (r *LiveRunner) runCronJobSuspend(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	cj, err := r.cronJobFor(cfg)
	if err != nil {
		return err
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()
	if step.Timeout > 0 {
		var cancelTimeout context.CancelFunc
		opCtx, cancelTimeout = context.WithTimeout(opCtx, step.Timeout)
		defer cancelTimeout()
	}

	suspend := phase == PhaseDown
	r.logLive("%s %s", executor.FormatCronJobPlan(suspend), step.Ref)
	return cj.Suspend(opCtx, step.Namespace, step.Name, suspend)
}

func (r *LiveRunner) runJobStep(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	job, err := r.jobFor(cfg)
	if err != nil {
		return err
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()
	if step.Timeout > 0 {
		var cancelTimeout context.CancelFunc
		opCtx, cancelTimeout = context.WithTimeout(opCtx, step.Timeout)
		defer cancelTimeout()
	}

	if phase == PhaseDown {
		r.logLive("delete job %s/%s (background propagation, ignore-not-found)", step.Namespace, step.Name)
		return job.Delete(opCtx, step.Namespace, step.Name)
	}

	r.logLive("%s", executor.FormatJobPlan(false, step.Manifest, step.JobWaitForComplete()))
	if err := job.CreateFromManifest(opCtx, step.Namespace, step.Name, step.Manifest); err != nil {
		return err
	}
	if !step.JobWaitForComplete() {
		return nil
	}
	timeout := rolloutTimeout(cfg, step)
	r.logLive("wait job complete %s (timeout %s)", step.Ref, timeout)
	return r.withThrottledProgress(ctx, step, "waiting job complete", func(_ context.Context) error {
		return job.WaitComplete(opCtx, step.Namespace, step.Name, timeout)
	})
}

func (r *LiveRunner) pvcFor(cfg *config.Config) (executor.PVCDeleter, error) {
	if r.PVC != nil {
		return r.PVC, nil
	}
	key := cfg.Run.Kubeconfig
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedPVC != nil && r.cachedPVCKey == key {
		return r.cachedPVC, nil
	}
	pvc, err := executor.NewPVCDeleter(cfg.Run.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("pvc deleter: %w", err)
	}
	r.cachedPVC = pvc
	r.cachedPVCKey = key
	return pvc, nil
}

func (r *LiveRunner) cronJobFor(cfg *config.Config) (executor.CronJobSuspender, error) {
	if r.CronJob != nil {
		return r.CronJob, nil
	}
	key := cfg.Run.Kubeconfig
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedCronJob != nil && r.cachedCronJobKey == key {
		return r.cachedCronJob, nil
	}
	cj, err := executor.NewCronJob(cfg.Run.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("cronjob: %w", err)
	}
	r.cachedCronJob = cj
	r.cachedCronJobKey = key
	return cj, nil
}

func (r *LiveRunner) jobFor(cfg *config.Config) (executor.JobRunner, error) {
	if r.Job != nil {
		return r.Job, nil
	}
	key := cfg.Run.Kubeconfig
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedJob != nil && r.cachedJobKey == key {
		return r.cachedJob, nil
	}
	j, err := executor.NewJob(cfg.Run.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("job: %w", err)
	}
	r.cachedJob = j
	r.cachedJobKey = key
	return j, nil
}

func (r *LiveRunner) runPodExec(ctx context.Context, cfg *config.Config, step config.PipelineStep) error {
	execRunner, err := r.podExecFor(cfg)
	if err != nil {
		return err
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()
	if step.Timeout > 0 {
		var cancelTimeout context.CancelFunc
		opCtx, cancelTimeout = context.WithTimeout(opCtx, step.Timeout)
		defer cancelTimeout()
	}

	r.logLive("%s", executor.FormatExecPlan(step))
	stdout, stderr, err := execRunner.Run(opCtx, step)
	r.writeOutput(stdout)
	r.writeOutput(stderr)
	if err != nil {
		return fmt.Errorf("live: %w", err)
	}
	return nil
}

func (r *LiveRunner) podExecFor(cfg *config.Config) (executor.PodExec, error) {
	if r.PodExec != nil {
		return r.PodExec, nil
	}
	key := cfg.Run.Kubeconfig
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedPodExec != nil && r.cachedPodExecKey == key {
		return r.cachedPodExec, nil
	}
	pe, err := executor.NewPodExec(cfg.Run.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("pod exec: %w", err)
	}
	r.cachedPodExec = pe
	r.cachedPodExecKey = key
	return pe, nil
}

func (r *LiveRunner) runReleaseScript(ctx context.Context, cfg *config.Config, phase Phase, step config.PipelineStep) error {
	helm, err := r.helmFor(cfg)
	if err != nil {
		return err
	}
	opCtx, cancel := withOpTimeout(ctx, cfg)
	defer cancel()

	if phase == PhaseDown {
		if helm.UsesSDK() {
			r.logLive("helm sdk uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
		} else {
			r.logLive("helm uninstall %s/%s (--wait --ignore-not-found)", step.Namespace, step.Name)
		}
		return r.withThrottledProgress(ctx, step, "uninstalling release", func(_ context.Context) error {
			return helm.Uninstall(opCtx, step)
		})
	}

	if helm.UsesSDK() {
		spec, err := executor.ResolveChartSpec(cfg, step)
		if err != nil {
			return err
		}
		r.logLive("helm sdk upgrade --install %s/%s (%s)", step.Namespace, step.Name, executor.FormatChartPlan(spec))
		return r.withThrottledProgress(ctx, step, "installing release", func(_ context.Context) error {
			return helm.UpgradeInstall(opCtx, step)
		})
	}

	ws := cfg.Helm.Workspace
	if ws == "" {
		return fmt.Errorf("helm.workspace is empty (required for release step %s on up)", step.Ref)
	}
	script, err := executor.ResolveReleaseScript(cfg, step)
	if err != nil {
		return err
	}
	r.logLive("release script %s (%s)", script, phase)
	return r.withThrottledProgress(ctx, step, "running release script", func(_ context.Context) error {
		return helm.UpgradeInstall(opCtx, step)
	})
}

func (r *LiveRunner) helmFor(cfg *config.Config) (executor.HelmReleases, error) {
	if r.Helm != nil {
		return r.Helm, nil
	}
	key := cfg.Run.Kubeconfig + "|" + cfg.Run.Execution + "|" + cfg.Helm.Workspace + "|" + cfg.Command.Helm
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedHelm != nil && r.cachedHelmKey == key {
		return r.cachedHelm, nil
	}
	var writeOut func([]byte)
	if r.Log != nil {
		writeOut = r.writeOutput
	}
	helm, err := executor.NewHelmReleases(cfg, executor.HelmDeps{
		Cfg:      cfg,
		Run:      r.runProcess,
		WriteOut: writeOut,
	})
	if err != nil {
		return nil, fmt.Errorf("helm releases: %w", err)
	}
	r.cachedHelm = helm
	r.cachedHelmKey = key
	return helm, nil
}

func withOpTimeout(ctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	if cfg.Run.OperationTimeout > 0 {
		return context.WithTimeout(ctx, cfg.Run.OperationTimeout)
	}
	return ctx, func() {}
}

func scaleReplicas(phase Phase, step config.PipelineStep) int {
	if phase == PhaseDown {
		return 0
	}
	if step.Replicas != nil {
		return *step.Replicas
	}
	return 1
}

func rolloutTimeout(cfg *config.Config, step config.PipelineStep) time.Duration {
	if step.Timeout > 0 {
		return step.Timeout
	}
	if cfg.Run.OperationTimeout > 0 {
		return cfg.Run.OperationTimeout
	}
	return 5 * time.Minute
}

// withThrottledProgress wraps a long-running operation with periodic
// progress lines (every 30s). Avoids log spam by throttling to one
// line per 30s even if the underlying operation lasts minutes.
func (r *LiveRunner) withThrottledProgress(ctx context.Context, step config.PipelineStep, action string, fn func(context.Context) error) error {
	if r.Log == nil {
		return fn(ctx)
	}
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- fn(ctx)
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			r.Log.Emit(log.Entry{
				Kind:  log.KindLive,
				Level: log.LevelInfo,
				Msg:   fmt.Sprintf("still %s %s (elapsed %s)", action, step.Ref, time.Since(start).Round(time.Second)),
			})
		}
	}
}
