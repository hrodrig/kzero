package cluster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestResolveFromConfig_explicitKubeconfig(t *testing.T) {
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
    namespace: apps
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

	cfg := &config.Config{Run: config.RunConfig{Kubeconfig: kc}}
	tgt, err := ResolveFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tgt.ContextName != "qa-context" || tgt.ClusterName != "qa-cluster" {
		t.Fatalf("got context=%q cluster=%q", tgt.ContextName, tgt.ClusterName)
	}
	if tgt.Server != "https://qa.example:6443" || tgt.Namespace != "apps" {
		t.Fatalf("got server=%q namespace=%q", tgt.Server, tgt.Namespace)
	}
	if tgt.KubeconfigPath != kc {
		t.Fatalf("kubeconfig path: %q", tgt.KubeconfigPath)
	}
}

func TestPrint_includesClusterBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	kc := filepath.Join(dir, "kc.yaml")
	if err := os.WriteFile(kc, []byte(`
apiVersion: v1
kind: Config
current-context: dev
contexts:
- name: dev
  context:
    cluster: dev-cluster
    user: u
clusters:
- name: dev-cluster
  cluster:
    server: https://dev.local
users:
- name: u
  user: {}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{Name: "develop", Environment: "dev"},
		Run:     config.RunConfig{Kubeconfig: kc},
	}
	var buf strings.Builder
	if err := Print(&buf, cfg); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Kubernetes target:",
		"  context: dev",
		"  cluster: dev-cluster",
		"  api_server: https://dev.local",
		"config_metadata: name=\"develop\"",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
