package cluster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

func TestLoadRESTConfig_explicitKubeconfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	kc := filepath.Join(dir, "kc.yaml")
	if err := os.WriteFile(kc, []byte(`
apiVersion: v1
kind: Config
current-context: qa-context
contexts:
- name: qa-context
  context:
    cluster: qa-cluster
    user: qa-user
clusters:
- name: qa-cluster
  cluster:
    server: https://qa.example:6443
users:
- name: qa-user
  user: {}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadRESTConfig(kc)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "https://qa.example:6443" {
		t.Fatalf("host: %q", cfg.Host)
	}
}

func TestLoadRESTConfig_missingExplicitKubeconfig(t *testing.T) {
	t.Parallel()

	_, err := LoadRESTConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil || !strings.Contains(err.Error(), "load kubeconfig") {
		t.Fatalf("expected load kubeconfig error, got %v", err)
	}
}

func TestLoadRESTConfig_emptyWithoutKubeconfigOrCluster(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing.yaml"))
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := LoadRESTConfig("")
	if err == nil || !strings.Contains(err.Error(), "load kubeconfig") {
		t.Fatalf("expected load kubeconfig error, got %v", err)
	}
}

func TestTargetFromInCluster(t *testing.T) {
	t.Parallel()

	tgt, err := targetFromInCluster(&rest.Config{Host: "https://10.96.0.1:443"})
	if err != nil {
		t.Fatal(err)
	}
	if tgt.ContextName != "in-cluster" || tgt.ClusterName != "in-cluster" {
		t.Fatalf("context=%q cluster=%q", tgt.ContextName, tgt.ClusterName)
	}
	if tgt.Server != "https://10.96.0.1:443" {
		t.Fatalf("server: %q", tgt.Server)
	}
	if tgt.KubeconfigPath != "(in-cluster service account)" {
		t.Fatalf("kubeconfig path: %q", tgt.KubeconfigPath)
	}
	if tgt.Namespace == "" {
		t.Fatal("expected namespace")
	}
}
