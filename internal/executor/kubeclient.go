package executor

import (
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubernetesClient builds a clientset from kubeconfig path (empty uses default loading rules).
func NewKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	loading := clientcmd.NewDefaultClientConfigLoadingRules()
	if k := strings.TrimSpace(kubeconfig); k != "" {
		loading.ExplicitPath = k
	}
	overrides := &clientcmd.ConfigOverrides{}
	cc, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	return kubernetes.NewForConfig(rest.CopyConfig(cc))
}
