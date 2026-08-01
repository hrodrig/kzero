package validate

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClientFactory builds a Kubernetes clientset from run.kubeconfig (empty = default rules).
type ClientFactory func(kubeconfig string) (kubernetes.Interface, error)

// Line is one workload check result for analyze output.
type Line struct {
	Ref    string
	OK     bool
	Detail string
}

// CheckPipelineWorkloads verifies deployment/statefulset/pvc/exec/job/cronjob steps against the API when a client is available.
// Returns lines in stable ref order, a skip reason (non-empty if checks were not run), and a combined error if any check failed.
func CheckPipelineWorkloads(ctx context.Context, cfg *config.Config, factory ClientFactory) (lines []Line, skipped string, err error) {
	if cfg == nil {
		return nil, "no config", nil
	}
	if factory == nil {
		factory = ClientFactoryDefault()
	}
	client, clientErr := factory(cfg.Run.Kubeconfig)
	if clientErr != nil {
		return nil, fmt.Sprintf("cannot load kubeconfig: %v", clientErr), nil
	}

	refs := collectPipelineResourceRefs(cfg)
	lines = make([]Line, 0, len(refs))
	var fail []string
	for _, ref := range refs {
		step := ref.step
		line := Line{Ref: ref.key}
		okDetail, checkErr := checkPipelineResource(ctx, client, step)
		if checkErr != nil {
			line.Detail = checkErr.Error()
			fail = append(fail, ref.key+": "+line.Detail)
		} else {
			line.OK = true
			line.Detail = okDetail
		}
		lines = append(lines, line)
	}
	if len(fail) > 0 {
		return lines, "", fmt.Errorf("cluster validation failed:\n  - %s", strings.Join(fail, "\n  - "))
	}
	return lines, "", nil
}

type pipelineResourceRef struct {
	key  string
	step config.PipelineStep
}

func collectPipelineResourceRefs(cfg *config.Config) []pipelineResourceRef {
	seen := make(map[string]struct{})
	var out []pipelineResourceRef
	add := func(steps []config.PipelineStep) {
		for _, s := range steps {
			if s.Type != "deployment" && s.Type != "statefulset" && s.Type != "pvc" && s.Type != "exec" && s.Type != "job" && s.Type != "cronjob" {
				continue
			}
			if s.Namespace == "" || s.Name == "" {
				continue
			}
			key := s.Type + "." + s.Namespace + "/" + s.Name
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, pipelineResourceRef{key: key, step: s})
		}
	}
	add(cfg.Pipelines.Down)
	add(cfg.Pipelines.Up)
	return out
}

func collectWorkloadRefs(cfg *config.Config) []pipelineResourceRef {
	return collectPipelineResourceRefs(cfg)
}

func checkPipelineResource(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (okDetail string, err error) {
	switch step.Type {
	case "deployment":
		return checkScalableDeployment(ctx, client, step)
	case "statefulset":
		return checkScalableStatefulSet(ctx, client, step)
	case "pvc":
		return checkPVC(ctx, client, step)
	case "exec":
		return checkExec(ctx, client, step)
	case "cronjob":
		return checkCronJob(ctx, client, step)
	case "job":
		return checkJob(ctx, client, step)
	default:
		return "", fmt.Errorf("unsupported kind %q", step.Type)
	}
}

func checkScalableDeployment(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (string, error) {
	dep, getErr := client.AppsV1().Deployments(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if getErr != nil {
		return "", friendlyAPIError(step.Type, step.Namespace, step.Name, getErr)
	}
	if dep.Spec.Replicas == nil {
		return "", fmt.Errorf("%s %s/%s: spec.replicas unset (cannot scale)", step.Type, step.Namespace, step.Name)
	}
	return "found", nil
}

func checkScalableStatefulSet(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (string, error) {
	sts, getErr := client.AppsV1().StatefulSets(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if getErr != nil {
		return "", friendlyAPIError(step.Type, step.Namespace, step.Name, getErr)
	}
	if sts.Spec.Replicas == nil {
		return "", fmt.Errorf("%s %s/%s: spec.replicas unset (cannot scale)", step.Type, step.Namespace, step.Name)
	}
	return "found", nil
}

func checkPVC(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (string, error) {
	_, getErr := client.CoreV1().PersistentVolumeClaims(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			return "not found (delete no-op)", nil
		}
		return "", friendlyAPIError(step.Type, step.Namespace, step.Name, getErr)
	}
	return "found", nil
}

func checkExec(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (string, error) {
	pod, getErr := client.CoreV1().Pods(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if getErr != nil {
		return "", friendlyAPIError(step.Type, step.Namespace, step.Name, getErr)
	}
	if !podHasContainer(pod, step.Container) {
		return "", fmt.Errorf("exec %s/%s: container %q not found in pod", step.Namespace, step.Name, step.Container)
	}
	return "pod found, container ok", nil
}

func checkCronJob(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (string, error) {
	_, getErr := client.BatchV1().CronJobs(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if getErr != nil {
		return "", friendlyAPIError(step.Type, step.Namespace, step.Name, getErr)
	}
	return "found", nil
}

func checkJob(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (string, error) {
	_, getErr := client.BatchV1().Jobs(step.Namespace).Get(ctx, step.Name, metav1.GetOptions{})
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			return "not found (create on up / delete no-op)", nil
		}
		return "", friendlyAPIError(step.Type, step.Namespace, step.Name, getErr)
	}
	return "found", nil
}

func podHasContainer(pod *corev1.Pod, name string) bool {
	for _, c := range pod.Spec.Containers {
		if c.Name == name {
			return true
		}
	}
	for _, c := range pod.Spec.InitContainers {
		if c.Name == name {
			return true
		}
	}
	return false
}

func friendlyAPIError(kind, namespace, name string, err error) error {
	switch {
	case apierrors.IsNotFound(err):
		return fmt.Errorf("%s %s/%s: not found", kind, namespace, name)
	case apierrors.IsForbidden(err):
		return fmt.Errorf("%s %s/%s: forbidden (%v)", kind, namespace, name, err)
	default:
		return fmt.Errorf("%s %s/%s: %w", kind, namespace, name, err)
	}
}

// PrintClusterValidation writes validation results to w (stdout section) and skip notes to errW (stderr).
func PrintClusterValidation(w, errW io.Writer, cfg *config.Config, factory ClientFactory) error {
	lines, skipped, err := CheckPipelineWorkloads(context.Background(), cfg, factory)
	if skipped != "" {
		if len(collectWorkloadRefs(cfg)) == 0 {
			return nil
		}
		_ = log.WriteLine(errW, log.LevelWarn, fmt.Sprintf("note: cluster validation skipped (%s)", skipped))
		return nil
	}
	if len(lines) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if err := log.WriteLine(w, log.LevelInfo, "Cluster validation:"); err != nil {
		return err
	}
	for _, line := range lines {
		status := "FAIL"
		level := log.LevelWarn
		if line.OK {
			status = "OK"
			level = log.LevelInfo
		}
		detail := line.Detail
		if line.OK {
			detail = ""
		}
		if detail != "" {
			if err := log.WriteLine(w, level, fmt.Sprintf("  %s  %s (%s)", status, line.Ref, detail)); err != nil {
				return err
			}
		} else {
			if err := log.WriteLine(w, level, fmt.Sprintf("  %s  %s", status, line.Ref)); err != nil {
				return err
			}
		}
	}
	return err
}
