package cluster

import (
	"fmt"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// LoadRESTConfig builds REST client settings from run.kubeconfig.
// When the path is empty, default kubeconfig discovery is tried first, then
// in-cluster service account credentials (typical for Pods with run.kubeconfig unset).
func LoadRESTConfig(kubeconfig string) (*rest.Config, error) {
	if k := strings.TrimSpace(kubeconfig); k != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", k)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig: %w", err)
		}
		return cfg, nil
	}

	loading := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, overrides).ClientConfig()
	if err == nil {
		return cfg, nil
	}

	ic, icErr := rest.InClusterConfig()
	if icErr == nil {
		return ic, nil
	}

	return nil, fmt.Errorf("load kubeconfig: %w (in-cluster: %v)", err, icErr)
}
