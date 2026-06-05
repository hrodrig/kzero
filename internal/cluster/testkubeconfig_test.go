package cluster

import (
	"os"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestTestKubeconfigPath_writesResolvableKubeconfig(t *testing.T) {
	t.Parallel()

	path := TestKubeconfigPath(t)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveFromConfig(&config.Config{Run: config.RunConfig{Kubeconfig: path}})
	if err != nil {
		t.Fatal(err)
	}
}
