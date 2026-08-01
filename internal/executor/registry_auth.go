package executor

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/hrodrig/kzero/internal/config"
	"helm.sh/helm/v4/pkg/registry"
)

// NewHelmRegistryClient builds the registry client Helm SDK needs for oci:// charts.
func NewHelmRegistryClient() (*registry.Client, error) {
	client, err := registry.NewClient()
	if err != nil {
		return nil, fmt.Errorf("helm registry client: %w", err)
	}
	return client, nil
}

// EnsureOCIRegistryAuth logs into configured registries when chartRef uses oci://.
// regClient must be the same client passed to action.Upgrade.SetRegistryClient.
func EnsureOCIRegistryAuth(cfg *config.Config, chartRef string, regClient *registry.Client, cache map[string]struct{}, mu *sync.Mutex) error {
	host, err := ociRegistryHost(chartRef)
	if err != nil {
		return err
	}
	if host == "" {
		return nil
	}
	reg, ok := registryConfigForHost(cfg, host)
	if !ok {
		return nil
	}
	if regClient == nil {
		return fmt.Errorf("helm registry login %s: registry client is nil", host)
	}
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if cache != nil {
		if _, done := cache[host]; done {
			return nil
		}
	}
	password, err := resolveRegistryPassword(reg)
	if err != nil {
		return err
	}
	if err := regClient.Login(host, registry.LoginOptBasicAuth(reg.Username, password)); err != nil {
		return fmt.Errorf("helm registry login %s: %w", host, err)
	}
	if cache != nil {
		cache[host] = struct{}{}
	}
	return nil
}

func registryConfigForHost(cfg *config.Config, host string) (config.HelmRegistryConfig, bool) {
	if cfg == nil {
		return config.HelmRegistryConfig{}, false
	}
	host = strings.TrimSpace(host)
	for _, reg := range cfg.Helm.Registries {
		if strings.EqualFold(strings.TrimSpace(reg.Host), host) {
			return reg, true
		}
	}
	return config.HelmRegistryConfig{}, false
}

func resolveRegistryPassword(reg config.HelmRegistryConfig) (string, error) {
	if env := strings.TrimSpace(reg.PasswordEnv); env != "" {
		v, ok := os.LookupEnv(env)
		if !ok || v == "" {
			return "", fmt.Errorf("helm.registries host %q: env %q is unset or empty", reg.Host, env)
		}
		return v, nil
	}
	if reg.Password != "" {
		return reg.Password, nil
	}
	return "", fmt.Errorf("helm.registries host %q: password or password_env is required", reg.Host)
}

func ociRegistryHost(chartRef string) (string, error) {
	chartRef = strings.TrimSpace(chartRef)
	if !strings.HasPrefix(chartRef, "oci://") {
		return "", nil
	}
	rest := strings.TrimPrefix(chartRef, "oci://")
	host, _, found := strings.Cut(rest, "/")
	if !found || strings.TrimSpace(host) == "" {
		return "", fmt.Errorf("invalid oci chart ref %q", chartRef)
	}
	return host, nil
}
