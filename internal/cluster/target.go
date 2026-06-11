package cluster

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
	"github.com/hrodrig/kzero/internal/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	api "k8s.io/client-go/tools/clientcmd/api"
)

// Target is the Kubernetes API endpoint resolved from run.kubeconfig (or default loading rules).
type Target struct {
	ContextName    string
	ClusterName    string
	Server         string
	Namespace      string
	KubeconfigPath string
}

// ResolveFromConfig uses cfg.Run.Kubeconfig, default client-go loading rules when empty,
// and in-cluster service account credentials when no kubeconfig is available.
func ResolveFromConfig(cfg *config.Config) (Target, error) {
	if cfg == nil {
		return Target{}, fmt.Errorf("no config")
	}
	loading := clientcmd.NewDefaultClientConfigLoadingRules()
	kc := strings.TrimSpace(cfg.Run.Kubeconfig)
	if kc != "" {
		loading.ExplicitPath = kc
		raw, err := loading.Load()
		if err != nil {
			return Target{}, fmt.Errorf("load kubeconfig: %w", err)
		}
		t, err := targetFromRaw(raw)
		if err != nil {
			return Target{}, err
		}
		t.KubeconfigPath = kc
		return t, nil
	}

	raw, err := loading.Load()
	if err == nil {
		if t, tErr := targetFromRaw(raw); tErr == nil {
			if f := loading.GetDefaultFilename(); f != "" {
				t.KubeconfigPath = f
			} else {
				t.KubeconfigPath = "(default kubeconfig search path)"
			}
			return t, nil
		}
	}

	return resolveInClusterTarget(err)
}

func resolveInClusterTarget(kubeconfigErr error) (Target, error) {
	ic, icErr := rest.InClusterConfig()
	if icErr == nil {
		return targetFromInCluster(ic)
	}
	if kubeconfigErr != nil {
		return Target{}, fmt.Errorf("load kubeconfig: %w", kubeconfigErr)
	}
	return Target{}, fmt.Errorf("load kubeconfig: no valid kubeconfig found (in-cluster: %v)", icErr)
}

func targetFromInCluster(cfg *rest.Config) (Target, error) {
	if cfg == nil {
		return Target{}, fmt.Errorf("empty in-cluster config")
	}
	ns := "default"
	if b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if n := strings.TrimSpace(string(b)); n != "" {
			ns = n
		}
	}
	return Target{
		ContextName:    "in-cluster",
		ClusterName:    "in-cluster",
		Server:         cfg.Host,
		Namespace:      ns,
		KubeconfigPath: "(in-cluster service account)",
	}, nil
}

func targetFromRaw(raw *api.Config) (Target, error) {
	if raw == nil {
		return Target{}, fmt.Errorf("empty kubeconfig")
	}
	ctxName := raw.CurrentContext
	if ctxName == "" {
		return Target{}, fmt.Errorf("kubeconfig has no current-context")
	}
	ctx, ok := raw.Contexts[ctxName]
	if !ok || ctx == nil {
		return Target{}, fmt.Errorf("context %q not found", ctxName)
	}
	cluster, ok := raw.Clusters[ctx.Cluster]
	if !ok || cluster == nil {
		return Target{}, fmt.Errorf("cluster %q for context %q not found", ctx.Cluster, ctxName)
	}
	ns := ctx.Namespace
	if ns == "" {
		ns = "default"
	}
	return Target{
		ContextName: ctxName,
		ClusterName: ctx.Cluster,
		Server:      cluster.Server,
		Namespace:   ns,
	}, nil
}

// Print writes a fixed block so operators see the real API target before any mutation.
func Print(w io.Writer, cfg *config.Config) error {
	t, err := ResolveFromConfig(cfg)
	if err != nil {
		return err
	}
	lines := []string{
		"Kubernetes target:",
		fmt.Sprintf("  started_at: %s", time.Now().Format(time.RFC3339)),
	}
	if cfg != nil && strings.TrimSpace(cfg.Client.ID) != "" {
		lines = append(lines, fmt.Sprintf("  client_id: %s", cfg.Client.ID))
	}
	lines = append(lines,
		fmt.Sprintf("  os_user: %s", operatorField(correlation.OperatorUser())),
		fmt.Sprintf("  os_uid: %s", operatorField(correlation.OperatorUID())),
	)
	lines = append(lines,
		fmt.Sprintf("  context: %s", t.ContextName),
		fmt.Sprintf("  cluster: %s", t.ClusterName),
		fmt.Sprintf("  namespace: %s", t.Namespace),
		fmt.Sprintf("  api_server: %s", t.Server),
		fmt.Sprintf("  kubeconfig: %s", t.KubeconfigPath),
	)
	if cfg != nil && (cfg.Cluster.Name != "" || cfg.Cluster.Environment != "") {
		lines = append(lines, fmt.Sprintf(
			"  config_metadata: name=%q environment=%q (YAML label only; not verified against the API)",
			cfg.Cluster.Name, cfg.Cluster.Environment,
		))
	}
	for _, line := range lines {
		if err := log.WriteLine(w, log.LevelInfo, line); err != nil {
			return err
		}
	}
	return nil
}

func operatorField(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(unknown)"
	}
	return v
}
