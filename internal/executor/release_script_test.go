package executor

import (
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestResolveReleaseScriptIn_workspaceOverride(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	step := config.PipelineStep{Ref: "release.mon/prom", Name: "prom", Type: "release"}
	got, err := ResolveReleaseScriptIn(cfg, step, "./helm-assets")
	if err != nil || got != "helm-assets/prom.sh" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestEnsureOCIRegistryAuth_invalidOCIRef(t *testing.T) {
	t.Parallel()
	err := EnsureOCIRegistryAuth(&config.Config{}, "oci://bad", nil, nil)
	if err == nil {
		t.Fatal("expected invalid oci ref error")
	}
}
