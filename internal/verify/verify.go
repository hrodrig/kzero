// Package verify runs post-up readiness checks against the Kubernetes API.
package verify

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
	"github.com/hrodrig/kzero/internal/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CheckWorkloadsReady = "workloads_ready"
	CheckNodesReady     = "nodes_ready"

	OutcomeReady  = "ready"
	OutcomeFailed = "failed"
)

// DefaultChecks runs when verify.checks is empty.
var DefaultChecks = []string{CheckWorkloadsReady, CheckNodesReady}

// Item is one row inside a named check (e.g. a workload ref).
type Item struct {
	Ref    string `json:"ref,omitempty"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// CheckResult is the outcome of a single configured check.
type CheckResult struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Items []Item `json:"items,omitempty"`
}

// Report is the full verify output document.
type Report struct {
	Outcome  string        `json:"outcome"`
	Cluster  string        `json:"cluster_name,omitempty"`
	ClientID string        `json:"client_id,omitempty"`
	Checks   []CheckResult `json:"checks"`
}

// Run executes configured checks. factory may be nil (uses validate.DefaultClientFactory).
func Run(ctx context.Context, cfg *config.Config, factory validate.ClientFactory) (Report, error) {
	report := Report{Outcome: OutcomeReady}
	if cfg == nil {
		return report, fmt.Errorf("verify: no config")
	}
	report.Cluster = cfg.Cluster.Name
	report.ClientID = correlation.ClientID(cfg)

	if factory == nil {
		factory = validate.DefaultClientFactory
	}
	client, err := factory(cfg.Run.Kubeconfig)
	if err != nil {
		report.Outcome = OutcomeFailed
		return report, fmt.Errorf("verify: cannot load kubeconfig: %w", err)
	}

	checks := cfg.Verify.Checks
	if len(checks) == 0 {
		checks = append([]string(nil), DefaultChecks...)
	}

	for _, name := range checks {
		cr, runErr := runCheck(ctx, client, cfg, name)
		report.Checks = append(report.Checks, cr)
		if runErr != nil {
			report.Outcome = OutcomeFailed
			if err == nil {
				err = runErr
			}
		}
		if !cr.OK {
			report.Outcome = OutcomeFailed
		}
	}
	return report, err
}

func runCheck(ctx context.Context, client kubernetes.Interface, cfg *config.Config, name string) (CheckResult, error) {
	switch name {
	case CheckWorkloadsReady:
		return checkWorkloadsReady(ctx, client, cfg)
	case CheckNodesReady:
		return checkNodesReady(ctx, client)
	case CheckPodsSchedulable:
		return checkPodsSchedulable(ctx, client, cfg)
	default:
		return CheckResult{Name: name, OK: false}, fmt.Errorf("verify: unknown check %q", name)
	}
}

func checkWorkloadsReady(ctx context.Context, client kubernetes.Interface, cfg *config.Config) (CheckResult, error) {
	cr := CheckResult{Name: CheckWorkloadsReady, OK: true}
	steps := collectUpWorkloads(cfg)
	if len(steps) == 0 {
		cr.Items = append(cr.Items, Item{OK: true, Detail: "no deployment/statefulset steps in pipelines.up"})
		return cr, nil
	}
	for _, step := range steps {
		item := Item{Ref: step.Ref}
		ready, detail, err := workloadReady(ctx, client, step)
		if err != nil {
			item.Detail = err.Error()
			cr.OK = false
		} else if !ready {
			item.Detail = detail
			cr.OK = false
		} else {
			item.OK = true
			item.Detail = detail
		}
		cr.Items = append(cr.Items, item)
	}
	if !cr.OK {
		return cr, fmt.Errorf("verify: workloads_ready failed")
	}
	return cr, nil
}

func checkNodesReady(ctx context.Context, client kubernetes.Interface) (CheckResult, error) {
	cr := CheckResult{Name: CheckNodesReady, OK: true}
	list, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		cr.OK = false
		cr.Items = append(cr.Items, Item{Detail: err.Error()})
		return cr, fmt.Errorf("verify: list nodes: %w", err)
	}
	if len(list.Items) == 0 {
		cr.OK = false
		cr.Items = append(cr.Items, Item{Detail: "no nodes in cluster"})
		return cr, fmt.Errorf("verify: nodes_ready failed")
	}
	ready := 0
	for _, n := range list.Items {
		if nodeReady(&n) {
			ready++
		}
	}
	item := Item{
		OK:     ready == len(list.Items),
		Detail: fmt.Sprintf("%d/%d nodes Ready", ready, len(list.Items)),
	}
	cr.Items = append(cr.Items, item)
	if !item.OK {
		cr.OK = false
		return cr, fmt.Errorf("verify: nodes_ready failed")
	}
	return cr, nil
}

func nodeReady(n *corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func collectUpWorkloads(cfg *config.Config) []config.PipelineStep {
	seen := make(map[string]struct{})
	var out []config.PipelineStep
	for _, s := range cfg.Pipelines.Up {
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
		out = append(out, s)
	}
	return out
}

func desiredReplicas(step config.PipelineStep) int32 {
	if step.Replicas != nil {
		return int32(*step.Replicas)
	}
	return 1
}

func workloadReady(ctx context.Context, client kubernetes.Interface, step config.PipelineStep) (bool, string, error) {
	desired := desiredReplicas(step)
	switch step.Type {
	case "deployment":
		return deploymentReady(ctx, client, step.Namespace, step.Name, desired)
	case "statefulset":
		return statefulSetReady(ctx, client, step.Namespace, step.Name, desired)
	default:
		return false, "", fmt.Errorf("unsupported kind %q", step.Type)
	}
}

func deploymentReady(ctx context.Context, client kubernetes.Interface, namespace, name string, desired int32) (bool, string, error) {
	dep, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, "", friendlyAPIError("deployment", namespace, name, err)
	}
	if dep.Spec.Replicas != nil && *dep.Spec.Replicas == 0 {
		return true, "scaled to 0", nil
	}
	return replicasReady(dep.Status.ObservedGeneration, dep.Generation, dep.Status.UpdatedReplicas, dep.Status.ReadyReplicas, desired, depProgressingFalse(dep.Status.Conditions))
}

func statefulSetReady(ctx context.Context, client kubernetes.Interface, namespace, name string, desired int32) (bool, string, error) {
	sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, "", friendlyAPIError("statefulset", namespace, name, err)
	}
	if sts.Spec.Replicas != nil && *sts.Spec.Replicas == 0 {
		return true, "scaled to 0", nil
	}
	return replicasReady(sts.Status.ObservedGeneration, sts.Generation, sts.Status.UpdatedReplicas, sts.Status.ReadyReplicas, desired, "")
}

func depProgressingFalse(conditions []appsv1.DeploymentCondition) string {
	for _, c := range conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Status == corev1.ConditionFalse {
			return c.Message
		}
	}
	return ""
}

func replicasReady(observedGen, generation int64, updated, ready, desired int32, blockMsg string) (bool, string, error) {
	if blockMsg != "" {
		return false, blockMsg, nil
	}
	if observedGen < generation {
		return false, fmt.Sprintf("%d/%d ready (rollout in progress)", ready, desired), nil
	}
	if updated < desired || ready < desired {
		return false, fmt.Sprintf("%d/%d ready", ready, desired), nil
	}
	return true, fmt.Sprintf("%d/%d ready", ready, desired), nil
}

func friendlyAPIError(kind, namespace, name string, err error) error {
	switch {
	case apierrors.IsNotFound(err):
		return fmt.Errorf("%s %s/%s: not found", kind, namespace, name)
	case apierrors.IsForbidden(err):
		return fmt.Errorf("%s %s/%s: forbidden", kind, namespace, name)
	default:
		return fmt.Errorf("%s %s/%s: %w", kind, namespace, name, err)
	}
}

// Failed reports whether the report indicates verify failure.
func Failed(r Report) bool {
	return r.Outcome != OutcomeReady
}

// ErrorMessage joins check failures for CLI errors.
func ErrorMessage(r Report) string {
	if !Failed(r) {
		return ""
	}
	var parts []string
	for _, c := range r.Checks {
		if c.OK {
			continue
		}
		parts = append(parts, c.Name)
	}
	if len(parts) == 0 {
		return "verify failed"
	}
	return "verify failed: " + strings.Join(parts, ", ")
}
