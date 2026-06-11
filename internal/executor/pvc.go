package executor

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PVCDeleter removes named PersistentVolumeClaims via the Kubernetes API.
type PVCDeleter interface {
	Delete(ctx context.Context, namespace, name string) error
}

// PVC deletes claims with background propagation; not-found is ignored.
type PVC struct {
	client kubernetes.Interface
}

// NewPVCDeleter builds an API-backed PVC deleter from kubeconfig (empty = default rules).
func NewPVCDeleter(kubeconfig string) (*PVC, error) {
	client, err := NewKubernetesClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &PVC{client: client}, nil
}

// Delete removes the claim. Uses DeletePropagationBackground; missing claims are no-ops.
func (p *PVC) Delete(ctx context.Context, namespace, name string) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("pvc delete: nil client")
	}
	propagation := metav1.DeletePropagationBackground
	err := p.client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return WrapAPIError(err, fmt.Sprintf("pvc %s/%s", namespace, name))
	}
	return nil
}
