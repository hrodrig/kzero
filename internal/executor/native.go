package executor

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Native scales and waits via the Kubernetes API (client-go).
type Native struct {
	client kubernetes.Interface
}

// NewNative returns an API-backed Workload executor.
func NewNative(client kubernetes.Interface) *Native {
	return &Native{client: client}
}

func (n *Native) Scale(ctx context.Context, kind, namespace, name string, replicas int32) error {
	switch kind {
	case "deployment":
		return n.scaleDeployment(ctx, namespace, name, replicas)
	case "statefulset":
		return n.scaleStatefulSet(ctx, namespace, name, replicas)
	default:
		return fmt.Errorf("native: %w: %q", ErrUnsupported, kind)
	}
}

func (n *Native) scaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	dep, err := n.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("deployment %s/%s", namespace, name))
	}
	dep.Spec.Replicas = &replicas
	_, err = n.client.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("deployment %s/%s", namespace, name))
	}
	return nil
}

func (n *Native) scaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) error {
	sts, err := n.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("statefulset %s/%s", namespace, name))
	}
	sts.Spec.Replicas = &replicas
	_, err = n.client.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("statefulset %s/%s", namespace, name))
	}
	return nil
}

func (n *Native) WaitRollout(ctx context.Context, kind, namespace, name string, timeout time.Duration) error {
	if _, ok := scalableKinds[kind]; !ok {
		return fmt.Errorf("native: %w: %q", ErrUnsupported, kind)
	}
	deadline := time.Now().Add(timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	interval := 2 * time.Second
	return wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		switch kind {
		case "deployment":
			return n.deploymentReady(ctx, namespace, name)
		case "statefulset":
			return n.statefulSetReady(ctx, namespace, name)
		default:
			return false, fmt.Errorf("native: %w: %q", ErrUnsupported, kind)
		}
	})
}

func (n *Native) deploymentReady(ctx context.Context, namespace, name string) (bool, error) {
	dep, err := n.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, WrapAPIError(err, fmt.Sprintf("deployment %s/%s", namespace, name))
	}
	if dep.Spec.Replicas != nil && *dep.Spec.Replicas == 0 {
		return true, nil
	}
	if dep.Status.ObservedGeneration < dep.Generation {
		return false, nil
	}
	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	if dep.Status.UpdatedReplicas < desired || dep.Status.ReadyReplicas < desired {
		return false, nil
	}
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Status == "False" {
			return false, nil
		}
	}
	return true, nil
}

func (n *Native) statefulSetReady(ctx context.Context, namespace, name string) (bool, error) {
	sts, err := n.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, WrapAPIError(err, fmt.Sprintf("statefulset %s/%s", namespace, name))
	}
	if sts.Spec.Replicas != nil && *sts.Spec.Replicas == 0 {
		return true, nil
	}
	if sts.Status.ObservedGeneration < sts.Generation {
		return false, nil
	}
	desired := int32(1)
	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}
	return sts.Status.UpdatedReplicas >= desired && sts.Status.ReadyReplicas >= desired, nil
}
