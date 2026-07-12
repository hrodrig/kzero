// Package preflight verifies Kubernetes API reachability before live mutations.
package preflight

import (
	"context"
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/validate"
	"k8s.io/client-go/kubernetes"
)

// Check confirms the API server is reachable (ServerVersion). factory may be nil.
func Check(ctx context.Context, cfg *config.Config, factory validate.ClientFactory) error {
	if cfg == nil {
		return fmt.Errorf("preflight: no config")
	}
	if factory == nil {
		factory = validate.ClientFactoryDefault()
	}
	client, err := factory(cfg.Run.Kubeconfig)
	if err != nil {
		return fmt.Errorf("preflight: cannot load kubeconfig: %w", err)
	}
	if err := serverReachable(ctx, client); err != nil {
		return fmt.Errorf("preflight: cannot reach Kubernetes API: %w", err)
	}
	return nil
}

func serverReachable(ctx context.Context, client kubernetes.Interface) error {
	if client == nil {
		return fmt.Errorf("no client")
	}
	_, err := client.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	return nil
}

// DryRunLine is the planned preflight message for dry-run engine output.
const DryRunLine = "preflight: would verify Kubernetes API reachability"

// AnalyzeWarning returns a non-fatal warning for analyze stderr when live preflight would fail.
func AnalyzeWarning(ctx context.Context, cfg *config.Config, factory validate.ClientFactory) string {
	if cfg == nil {
		return ""
	}
	if err := Check(ctx, cfg, factory); err != nil {
		return fmt.Sprintf("preflight would fail in live mode (%s)", err.Error())
	}
	return ""
}
