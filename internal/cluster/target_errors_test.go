package cluster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	api "k8s.io/client-go/tools/clientcmd/api"
)

func TestResolveFromConfig_nilConfig(t *testing.T) {
	t.Parallel()

	_, err := ResolveFromConfig(nil)
	if err == nil || !strings.Contains(err.Error(), "no config") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveFromConfig_missingContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	kc := filepath.Join(dir, "kc.yaml")
	if err := os.WriteFile(kc, []byte(`
apiVersion: v1
kind: Config
contexts: []
clusters: []
users: []
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveFromConfig(&config.Config{Run: config.RunConfig{Kubeconfig: kc}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTargetFromRaw_missingCluster(t *testing.T) {
	t.Parallel()

	_, err := targetFromRaw(&api.Config{
		CurrentContext: "bad",
		Contexts: map[string]*api.Context{
			"bad": {Cluster: "missing", AuthInfo: "u"},
		},
		Clusters: map[string]*api.Cluster{},
		AuthInfos: map[string]*api.AuthInfo{
			"u": {},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("got %v", err)
	}
}
