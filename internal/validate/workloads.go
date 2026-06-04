package validate

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClientFactory builds a Kubernetes clientset from run.kubeconfig (empty = default rules).
type ClientFactory func(kubeconfig string) (kubernetes.Interface, error)

// DefaultClientFactory is used by analyze; tests may replace it.
var DefaultClientFactory ClientFactory = executor.NewKubernetesClient

// Line is one workload check result for analyze output.
type Line struct {
	Ref    string
	OK     bool
	Detail string
}

// CheckPipelineWorkloads verifies deployment/statefulset steps exist in the API when a client is available.
// Returns lines in stable ref order, a skip reason (non-empty if checks were not run), and a combined error if any check failed.
func CheckPipelineWorkloads(ctx context.Context, cfg *config.Config, factory ClientFactory) (lines []Line, skipped string, err error) {
	if cfg == nil {
		return nil, "no config", nil
	}
	if factory == nil {
		factory = DefaultClientFactory
	}
	client, clientErr := factory(cfg.Run.Kubeconfig)
	if clientErr != nil {
		return nil, fmt.Sprintf("cannot load kubeconfig: %v", clientErr), nil
	}

	refs := collectWorkloadRefs(cfg)
	lines = make([]Line, 0, len(refs))
	var fail []string
	for _, ref := range refs {
		step := ref.step
		line := Line{Ref: ref.key}
		checkErr := checkWorkload(ctx, client, step.Type, step.Namespace, step.Name)
		if checkErr != nil {
			line.Detail = checkErr.Error()
			fail = append(fail, ref.key+": "+line.Detail)
		} else {
			line.OK = true
			line.Detail = "found"
		}
		lines = append(lines, line)
	}
	if len(fail) > 0 {
		return lines, "", fmt.Errorf("cluster validation failed:\n  - %s", strings.Join(fail, "\n  - "))
	}
	return lines, "", nil
}

type workloadRef struct {
	key  string
	step config.PipelineStep
}

func collectWorkloadRefs(cfg *config.Config) []workloadRef {
	seen := make(map[string]struct{})
	var out []workloadRef
	add := func(steps []config.PipelineStep) {
		for _, s := range steps {
			if s.Type != "deployment" && s.Type != "statefulset" {
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
			out = append(out, workloadRef{key: key, step: s})
		}
	}
	add(cfg.Pipelines.Down)
	add(cfg.Pipelines.Up)
	return out
}

func checkWorkload(ctx context.Context, client kubernetes.Interface, kind, namespace, name string) error {
	switch kind {
	case "deployment":
		dep, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return friendlyAPIError(kind, namespace, name, err)
		}
		if dep.Spec.Replicas == nil {
			return fmt.Errorf("%s %s/%s: spec.replicas unset (cannot scale)", kind, namespace, name)
		}
		return nil
	case "statefulset":
		sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return friendlyAPIError(kind, namespace, name, err)
		}
		if sts.Spec.Replicas == nil {
			return fmt.Errorf("%s %s/%s: spec.replicas unset (cannot scale)", kind, namespace, name)
		}
		return nil
	default:
		return fmt.Errorf("unsupported kind %q", kind)
	}
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
		_, _ = fmt.Fprintf(errW, "note: cluster validation skipped (%s)\n", skipped)
		return nil
	}
	if len(lines) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nCluster validation:"); err != nil {
		return err
	}
	for _, line := range lines {
		status := "FAIL"
		if line.OK {
			status = "OK"
		}
		detail := line.Detail
		if line.OK {
			detail = ""
		}
		if detail != "" {
			if _, err := fmt.Fprintf(w, "  %s  %s (%s)\n", status, line.Ref, detail); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "  %s  %s\n", status, line.Ref); err != nil {
				return err
			}
		}
	}
	return err
}
