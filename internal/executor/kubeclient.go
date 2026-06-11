package executor

import (
	"github.com/hrodrig/kzero/internal/cluster"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewKubernetesClient builds a clientset from run.kubeconfig (empty uses default loading rules
// and in-cluster service account credentials when running inside a Pod).
func NewKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	cc, err := cluster.LoadRESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(rest.CopyConfig(cc))
}
