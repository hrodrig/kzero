package diff

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/engine"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/hrodrig/kzero/internal/validate"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Line is one step comparison result.
type Line struct {
	Ref    string
	OK     bool
	Detail string
	Skip   bool
}

// Run compares pipelines.<phase> desired state against the live cluster.
// Returns lines in pipeline order and a non-nil error when any line drifts or the API fails.
func Run(ctx context.Context, cfg *config.Config, phase engine.Phase, factory validate.ClientFactory) (lines []Line, err error) {
	if cfg == nil {
		return nil, fmt.Errorf("diff: no config")
	}
	if factory == nil {
		factory = validate.ClientFactoryDefault()
	}
	client, clientErr := factory(cfg.Run.Kubeconfig)
	if clientErr != nil {
		return nil, fmt.Errorf("cannot load kubeconfig: %w", clientErr)
	}

	steps := stepsForPhase(cfg, phase)
	lines = make([]Line, 0, len(steps))
	var drifts []string
	for _, step := range steps {
		line := compareStep(ctx, client, phase, step)
		lines = append(lines, line)
		if line.Skip {
			continue
		}
		if !line.OK {
			drifts = append(drifts, line.Ref+": "+line.Detail)
		}
	}
	if len(drifts) > 0 {
		return lines, fmt.Errorf("diff drift (%d):\n  - %s", len(drifts), strings.Join(drifts, "\n  - "))
	}
	return lines, nil
}

func stepsForPhase(cfg *config.Config, phase engine.Phase) []config.PipelineStep {
	switch phase {
	case engine.PhaseDown:
		return cfg.Pipelines.Down
	case engine.PhaseUp:
		return cfg.Pipelines.Up
	default:
		return nil
	}
}

func compareStep(ctx context.Context, client kubernetes.Interface, phase engine.Phase, step config.PipelineStep) Line {
	ref := step.Ref
	if ref == "" && step.Custom != "" {
		ref = "custom: " + step.Custom
	}
	if step.Custom != "" || step.Type == "exec" {
		return Line{Ref: ref, OK: true, Skip: true, Detail: "skipped (no cluster state)"}
	}
	if step.Namespace == "" || step.Name == "" {
		return Line{Ref: ref, OK: false, Detail: "missing namespace/name"}
	}
	switch step.Type {
	case "deployment", "statefulset":
		return compareWorkload(ctx, client, phase, step)
	case "pvc":
		return comparePVC(ctx, client, phase, step)
	case "release":
		return compareRelease(ctx, client, phase, step)
	case "cronjob":
		return compareCronJob(ctx, client, phase, step)
	case "job":
		return compareJob(ctx, client, phase, step)
	default:
		return Line{Ref: ref, OK: true, Skip: true, Detail: fmt.Sprintf("skipped (unsupported kind %q)", step.Type)}
	}
}

func compareWorkload(ctx context.Context, client kubernetes.Interface, phase engine.Phase, step config.PipelineStep) Line {
	desired := engine.DesiredReplicas(phase, step)
	var live *int32
	var getErr error
	switch step.Type {
	case "deployment":
		dep, err := client.AppsV1().Deployments(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
		getErr = err
		if err == nil {
			live = dep.Spec.Replicas
		}
	case "statefulset":
		sts, err := client.AppsV1().StatefulSets(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
		getErr = err
		if err == nil {
			live = sts.Spec.Replicas
		}
	}
	if getErr != nil {
		return Line{Ref: step.Ref, OK: false, Detail: friendlyGet(step.Type, step.Namespace, step.Name, getErr)}
	}
	if live == nil {
		return Line{Ref: step.Ref, OK: false, Detail: fmt.Sprintf("replicas=%d (live=unset)", desired)}
	}
	if int(*live) != desired {
		return Line{Ref: step.Ref, OK: false, Detail: fmt.Sprintf("replicas=%d (live=%d)", desired, *live)}
	}
	return Line{Ref: step.Ref, OK: true, Detail: fmt.Sprintf("replicas=%d (live=%d)", desired, *live)}
}

func comparePVC(ctx context.Context, client kubernetes.Interface, phase engine.Phase, step config.PipelineStep) Line {
	// pvc.* always deletes (both phases); desired after the step is absent.
	_ = phase
	_, err := client.CoreV1().PersistentVolumeClaims(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	present := err == nil
	if err != nil && !apierrors.IsNotFound(err) {
		return Line{Ref: step.Ref, OK: false, Detail: friendlyGet("pvc", step.Namespace, step.Name, err)}
	}
	return presenceLine(step.Ref, false, present, "pvc")
}

func compareRelease(ctx context.Context, client kubernetes.Interface, phase engine.Phase, step config.PipelineStep) Line {
	wantPresent := phase == engine.PhaseUp
	present, err := helmReleasePresent(ctx, client, step.Namespace, step.Name)
	if err != nil {
		return Line{Ref: step.Ref, OK: false, Detail: err.Error()}
	}
	return presenceLine(step.Ref, wantPresent, present, "release")
}

func compareCronJob(ctx context.Context, client kubernetes.Interface, phase engine.Phase, step config.PipelineStep) Line {
	wantSuspend := phase == engine.PhaseDown
	cj, err := client.BatchV1().CronJobs(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if err != nil {
		return Line{Ref: step.Ref, OK: false, Detail: friendlyGet("cronjob", step.Namespace, step.Name, err)}
	}
	liveSuspend := cj.Spec.Suspend != nil && *cj.Spec.Suspend
	if liveSuspend != wantSuspend {
		return Line{Ref: step.Ref, OK: false, Detail: fmt.Sprintf("suspend=%t (live=%t)", wantSuspend, liveSuspend)}
	}
	return Line{Ref: step.Ref, OK: true, Detail: fmt.Sprintf("suspend=%t (live=%t)", wantSuspend, liveSuspend)}
}

func compareJob(ctx context.Context, client kubernetes.Interface, phase engine.Phase, step config.PipelineStep) Line {
	wantPresent := phase == engine.PhaseUp
	_, err := client.BatchV1().Jobs(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	present := err == nil
	if err != nil && !apierrors.IsNotFound(err) {
		return Line{Ref: step.Ref, OK: false, Detail: friendlyGet("job", step.Namespace, step.Name, err)}
	}
	return presenceLine(step.Ref, wantPresent, present, "job")
}

func presenceLine(ref string, wantPresent, present bool, kind string) Line {
	if wantPresent == present {
		state := "absent"
		if present {
			state = "present"
		}
		return Line{Ref: ref, OK: true, Detail: state}
	}
	if wantPresent {
		return Line{Ref: ref, OK: false, Detail: fmt.Sprintf("%s: expected present (live=absent)", kind)}
	}
	return Line{Ref: ref, OK: false, Detail: fmt.Sprintf("%s: expected absent (live=present)", kind)}
}

// helmReleasePresent detects Helm v3 secret storage (labels owner=helm, name=<release>).
func helmReleasePresent(ctx context.Context, client kubernetes.Interface, namespace, name string) (bool, error) {
	list, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("owner=helm,name=%s", name),
	})
	if err != nil {
		return false, fmt.Errorf("release %s/%s: list helm secrets: %w", namespace, name, err)
	}
	return len(list.Items) > 0, nil
}

func friendlyGet(kind, namespace, name string, err error) string {
	switch {
	case apierrors.IsNotFound(err):
		return fmt.Sprintf("%s %s/%s: not found", kind, namespace, name)
	case apierrors.IsForbidden(err):
		return fmt.Sprintf("%s %s/%s: forbidden", kind, namespace, name)
	default:
		return fmt.Sprintf("%s %s/%s: %v", kind, namespace, name, err)
	}
}

// Print writes Diff output to w. Returns the Run error (caller maps exit codes).
func Print(w, errW io.Writer, cfg *config.Config, phase engine.Phase, factory validate.ClientFactory) error {
	if _, err := fmt.Fprintf(w, "Diff (phase=%s):\n", phase); err != nil {
		return err
	}
	lines, err := Run(context.Background(), cfg, phase, factory)
	if len(lines) == 0 && err == nil {
		_ = log.WriteLine(w, log.LevelInfo, "  (no comparable steps in pipelines."+string(phase)+")")
		return nil
	}
	for _, line := range lines {
		status := "DRIFT"
		level := log.LevelWarn
		if line.Skip {
			status = "SKIP"
			level = log.LevelInfo
		} else if line.OK {
			status = "OK"
			level = log.LevelInfo
		}
		msg := fmt.Sprintf("  %-6s %s  %s", status, line.Ref, line.Detail)
		_ = log.WriteLine(w, level, msg)
	}
	if err != nil && strings.Contains(err.Error(), "cannot load kubeconfig") {
		_ = log.WriteLine(errW, log.LevelWarn, "diff: "+err.Error())
	}
	return err
}
