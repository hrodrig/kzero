package cluster

import (
	"os"
	"path/filepath"
	"testing"
)

// TestKubeconfigPath returns a minimal kubeconfig in t.TempDir for CLI tests.
func TestKubeconfigPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test-kubeconfig.yaml")
	content := `apiVersion: v1
kind: Config
current-context: test-context
contexts:
- name: test-context
  context:
    cluster: test-cluster
    namespace: test-ns
    user: test-user
clusters:
- name: test-cluster
  cluster:
    server: https://test.kubernetes.local
users:
- name: test-user
  user: {}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test kubeconfig: %v", err)
	}
	return path
}
