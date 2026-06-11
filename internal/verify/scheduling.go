package verify

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const CheckPodsSchedulable = "pods_schedulable"

// NamespacesFromSteps returns unique non-empty namespaces from pipeline steps.
func NamespacesFromSteps(steps []config.PipelineStep) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range steps {
		ns := strings.TrimSpace(s.Namespace)
		if ns == "" {
			continue
		}
		if _, ok := seen[ns]; ok {
			continue
		}
		seen[ns] = struct{}{}
		out = append(out, ns)
	}
	return out
}

func collectUpNamespaces(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	return NamespacesFromSteps(cfg.Pipelines.Up)
}

func checkPodsSchedulable(ctx context.Context, client kubernetes.Interface, cfg *config.Config) (CheckResult, error) {
	cr := CheckResult{Name: CheckPodsSchedulable, OK: true}
	namespaces := collectUpNamespaces(cfg)
	if len(namespaces) == 0 {
		cr.Items = append(cr.Items, Item{OK: true, Detail: "no namespaces in pipelines.up"})
		return cr, nil
	}
	items, err := FindUnschedulablePods(ctx, client, namespaces)
	if err != nil {
		cr.OK = false
		cr.Items = append(cr.Items, Item{Detail: err.Error()})
		return cr, fmt.Errorf("verify: pods_schedulable: %w", err)
	}
	cr.Items = append(cr.Items, items...)
	for _, item := range items {
		if !item.OK {
			cr.OK = false
		}
	}
	if !cr.OK {
		return cr, fmt.Errorf("verify: pods_schedulable failed")
	}
	return cr, nil
}

// FindUnschedulablePods lists Pending pods with PodScheduled=False / Unschedulable in each namespace.
func FindUnschedulablePods(ctx context.Context, client kubernetes.Interface, namespaces []string) ([]Item, error) {
	var items []Item
	for _, ns := range namespaces {
		list, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			FieldSelector: "status.phase=Pending",
		})
		if err != nil {
			return nil, fmt.Errorf("list pods in %s: %w", ns, err)
		}
		for i := range list.Items {
			pod := &list.Items[i]
			if msg := unschedulableMessage(pod); msg != "" {
				items = append(items, Item{
					Ref:    fmt.Sprintf("pod.%s/%s", ns, pod.Name),
					OK:     false,
					Detail: msg,
				})
			}
		}
	}
	if len(items) == 0 {
		items = append(items, Item{OK: true, Detail: "no unschedulable Pending pods"})
	}
	return items, nil
}

func unschedulableMessage(pod *corev1.Pod) string {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodScheduled &&
			c.Status == corev1.ConditionFalse &&
			c.Reason == corev1.PodReasonUnschedulable {
			if strings.TrimSpace(c.Message) != "" {
				return c.Message
			}
			return "pod unschedulable (affinity, taints, or node selector)"
		}
	}
	return ""
}
