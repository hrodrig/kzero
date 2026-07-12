package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/validate"
	"github.com/hrodrig/kzero/internal/verify"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RunChecks validates configured probe checks. dryRun logs planned checks without API calls.
func RunChecks(ctx context.Context, cfg *config.Config, factory validate.ClientFactory, dryRun bool, upOK bool) error {
	if cfg == nil || len(cfg.InfraProbe.Checks) == 0 {
		return nil
	}
	if factory == nil {
		factory = validate.ClientFactoryDefault()
	}
	var client kubernetes.Interface
	if !dryRun {
		var err error
		client, err = factory(cfg.Run.Kubeconfig)
		if err != nil {
			return fmt.Errorf("infra probe checks: cannot load kubeconfig: %w", err)
		}
	}
	for i, check := range cfg.InfraProbe.Checks {
		if err := runOneCheck(ctx, cfg, client, check, dryRun, upOK); err != nil {
			return fmt.Errorf("infra probe checks[%d]: %w", i, err)
		}
	}
	return nil
}

func runOneCheck(ctx context.Context, cfg *config.Config, client kubernetes.Interface, check config.ProbeCheck, dryRun, upOK bool) error {
	if check.PVCBound != "" {
		ns, name, err := parsePVCRef(check.PVCBound)
		if err != nil {
			return err
		}
		if dryRun {
			return nil
		}
		return checkPVCBound(ctx, client, ns, name)
	}
	if check.ReleaseReady {
		if dryRun {
			return nil
		}
		if !upOK {
			return fmt.Errorf("release_ready: probe up did not complete successfully")
		}
	}
	if check.PodsSchedulable {
		if dryRun {
			return nil
		}
		return checkProbePodsSchedulable(ctx, client, cfg)
	}
	return nil
}

func checkProbePodsSchedulable(ctx context.Context, client kubernetes.Interface, cfg *config.Config) error {
	namespaces := verify.NamespacesFromSteps(cfg.InfraProbe.Pipeline.Up)
	if len(namespaces) == 0 {
		return fmt.Errorf("pods_schedulable: no namespaces in infra_probe.pipeline.up")
	}
	items, err := verify.FindUnschedulablePods(ctx, client, namespaces)
	if err != nil {
		return fmt.Errorf("pods_schedulable: %w", err)
	}
	for _, item := range items {
		if !item.OK {
			return fmt.Errorf("pods_schedulable %s: %s", item.Ref, item.Detail)
		}
	}
	return nil
}

func checkPVCBound(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("pvc_bound %s/%s: not found", namespace, name)
		}
		return fmt.Errorf("pvc_bound %s/%s: %w", namespace, name, err)
	}
	if pvc.Status.Phase != corev1.ClaimBound {
		return fmt.Errorf("pvc_bound %s/%s: phase %s (want Bound)", namespace, name, pvc.Status.Phase)
	}
	return nil
}

func parsePVCRef(ref string) (namespace, name string, err error) {
	ref = strings.TrimSpace(ref)
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("pvc_bound must be namespace/name, got %q", ref)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
