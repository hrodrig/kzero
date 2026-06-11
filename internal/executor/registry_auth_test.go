package executor

import (
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestOCIRegistryHost(t *testing.T) {
	t.Parallel()
	host, err := ociRegistryHost("oci://registry-1.docker.io/bitnamicharts/redis")
	if err != nil || host != "registry-1.docker.io" {
		t.Fatalf("host=%q err=%v", host, err)
	}
	if _, err := ociRegistryHost("oci://bad"); err == nil {
		t.Fatal("expected error for invalid ref")
	}
	if host, err := ociRegistryHost("./local/chart"); err != nil || host != "" {
		t.Fatalf("local chart host=%q err=%v", host, err)
	}
}

func TestResolveRegistryPassword_inline(t *testing.T) {
	t.Parallel()
	p, err := resolveRegistryPassword(config.HelmRegistryConfig{Host: "x", Password: "inline"})
	if err != nil || p != "inline" {
		t.Fatalf("password=%q err=%v", p, err)
	}
}

func TestRegistryConfigForHost_caseInsensitive(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Helm: config.HelmConfig{
			Registries: []config.HelmRegistryConfig{{Host: "GHCR.IO", Username: "u", Password: "p"}},
		},
	}
	reg, ok := registryConfigForHost(cfg, "ghcr.io")
	if !ok || reg.Username != "u" {
		t.Fatalf("reg=%+v ok=%v", reg, ok)
	}
}

func TestResolveRegistryPassword_env(t *testing.T) {
	t.Setenv("REG_PASS", "secret")
	p, err := resolveRegistryPassword(config.HelmRegistryConfig{Host: "ghcr.io", PasswordEnv: "REG_PASS"})
	if err != nil || p != "secret" {
		t.Fatalf("password=%q err=%v", p, err)
	}
}

func TestEnsureOCIRegistryAuth_skipsWithoutConfig(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Helm: config.HelmConfig{
			Registries: []config.HelmRegistryConfig{{Host: "ghcr.io", Username: "u", PasswordEnv: "MISSING"}},
		},
	}
	err := EnsureOCIRegistryAuth(cfg, "oci://docker.io/bitnamicharts/redis", nil, nil)
	if err != nil {
		t.Fatalf("public oci without matching host should skip: %v", err)
	}
}

func TestEnsureOCIRegistryAuth_missingPasswordEnv(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Helm: config.HelmConfig{
			Registries: []config.HelmRegistryConfig{{Host: "ghcr.io", Username: "u", PasswordEnv: "MISSING_GHCR_PASS"}},
		},
	}
	err := EnsureOCIRegistryAuth(cfg, "oci://ghcr.io/org/chart", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "MISSING_GHCR_PASS") {
		t.Fatalf("expected password env error, got %v", err)
	}
}

func TestResolveReleaseScript_defaultAndOverride(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Helm: config.HelmConfig{Workspace: "/charts"}}
	step := config.PipelineStep{Ref: "release.mon/prom", Name: "prom", Type: "release"}
	got, err := ResolveReleaseScript(cfg, step)
	if err != nil || got != "/charts/prom.sh" {
		t.Fatalf("default=%q err=%v", got, err)
	}
	step.Script = "monitoring/kube-prometheus-stack.sh"
	got, err = ResolveReleaseScript(cfg, step)
	if err != nil || got != "/charts/monitoring/kube-prometheus-stack.sh" {
		t.Fatalf("override=%q err=%v", got, err)
	}
}
