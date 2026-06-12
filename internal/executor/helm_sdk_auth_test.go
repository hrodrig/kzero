package executor

import (
	"sync"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	"helm.sh/helm/v3/pkg/action"
)

func TestSDKHelm_prepareOCIRegistry_nonOCI(t *testing.T) {
	t.Parallel()
	h := &SDKHelm{cfg: &config.Config{}}
	client := action.NewUpgrade(&action.Configuration{})
	reg, err := h.prepareOCIRegistry(client, "charts/redis", "release.ns/redis")
	if err != nil || reg != nil {
		t.Fatalf("reg=%v err=%v", reg, err)
	}
}

func TestEnsureOCIRegistryAuth_withMutexNoMatch(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	cfg := &config.Config{
		Helm: config.HelmConfig{
			Registries: []config.HelmRegistryConfig{{Host: "ghcr.io", Username: "u", Password: "p"}},
		},
	}
	regClient, err := NewHelmRegistryClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureOCIRegistryAuth(cfg, "oci://docker.io/bitnamicharts/redis", regClient, map[string]struct{}{}, &mu); err != nil {
		t.Fatalf("unmatched host should skip: %v", err)
	}
}
