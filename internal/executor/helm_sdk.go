package executor

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SDKHelm runs release steps via helm.sh/helm/v3 (no host helm binary or .sh).
type SDKHelm struct {
	cfg      *config.Config
	settings *cli.EnvSettings
	loggedIn map[string]struct{}
	loginMu  sync.Mutex
}

// NewSDKHelm builds a Helm SDK executor from cfg run.kubeconfig.
func NewSDKHelm(cfg *config.Config) (*SDKHelm, error) {
	if cfg == nil {
		return nil, fmt.Errorf("helm sdk: nil config")
	}
	settings := cli.New()
	if k := strings.TrimSpace(cfg.Run.Kubeconfig); k != "" {
		settings.KubeConfig = k
	}
	return &SDKHelm{cfg: cfg, settings: settings, loggedIn: make(map[string]struct{})}, nil
}

func (h *SDKHelm) UsesSDK() bool { return true }

func (h *SDKHelm) Uninstall(ctx context.Context, step config.PipelineStep) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	actionConfig, err := h.actionConfig(step.Namespace)
	if err != nil {
		return err
	}
	client := action.NewUninstall(actionConfig)
	client.Wait = true
	client.IgnoreNotFound = true
	client.Timeout = h.opTimeout(step)
	_, err = client.Run(step.Name)
	if err != nil {
		return fmt.Errorf("helm sdk uninstall %s/%s: %w", step.Namespace, step.Name, err)
	}
	return nil
}

func (h *SDKHelm) UpgradeInstall(ctx context.Context, step config.PipelineStep) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	spec, err := ResolveChartSpec(h.cfg, step)
	if err != nil {
		return err
	}
	if spec.CreateNamespace {
		if err := ensureNamespace(ctx, h.cfg, step.Namespace); err != nil {
			return fmt.Errorf("helm sdk create namespace %s: %w", step.Namespace, err)
		}
	}
	actionConfig, err := h.actionConfig(step.Namespace)
	if err != nil {
		return err
	}

	client := action.NewUpgrade(actionConfig)
	client.Install = true
	client.Wait = spec.Wait
	client.Timeout = spec.Timeout
	client.Namespace = step.Namespace
	client.ChartPathOptions.Version = spec.Version

	chartRef := resolveChartRef(h.cfg, spec.Chart)
	if err := EnsureOCIRegistryAuth(h.cfg, chartRef, h.loggedIn, &h.loginMu); err != nil {
		return fmt.Errorf("helm sdk auth for %s: %w", step.Ref, err)
	}
	chartPath, err := client.LocateChart(chartRef, h.settings)
	if err != nil {
		return fmt.Errorf("helm sdk locate chart for %s: %w", step.Ref, err)
	}
	ch, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("helm sdk load chart for %s: %w", step.Ref, err)
	}

	valOpts := values.Options{ValueFiles: spec.ValuesFiles}
	vals, err := valOpts.MergeValues(nil)
	if err != nil {
		return fmt.Errorf("helm sdk merge values for %s: %w", step.Ref, err)
	}

	_, err = client.Run(step.Name, ch, vals)
	if err != nil {
		return fmt.Errorf("helm sdk upgrade --install %s/%s: %w", step.Namespace, step.Name, err)
	}
	return nil
}

func (h *SDKHelm) actionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	debug := func(format string, v ...interface{}) {
		_, _ = fmt.Fprintf(io.Discard, format, v...)
	}
	if err := actionConfig.Init(h.settings.RESTClientGetter(), namespace, "secret", debug); err != nil {
		return nil, fmt.Errorf("helm sdk init: %w", err)
	}
	return actionConfig, nil
}

func (h *SDKHelm) opTimeout(step config.PipelineStep) time.Duration {
	if step.Timeout > 0 {
		return step.Timeout
	}
	if h.cfg.Run.OperationTimeout > 0 {
		return h.cfg.Run.OperationTimeout
	}
	return 5 * time.Minute
}

func resolveChartRef(cfg *config.Config, chart string) string {
	chart = strings.TrimSpace(chart)
	if chart == "" || strings.HasPrefix(chart, "oci://") || filepath.IsAbs(chart) {
		return chart
	}
	ws := strings.TrimSpace(cfg.Helm.Workspace)
	if ws == "" {
		return chart
	}
	return filepath.Join(ws, chart)
}

func ensureNamespace(ctx context.Context, cfg *config.Config, name string) error {
	client, err := kubeClientForHelm(cfg.Run.Kubeconfig)
	if err != nil {
		return err
	}
	_, err = client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	_, err = client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}, metav1.CreateOptions{})
	return err
}

// kubeClientForHelm is swapped in tests.
var kubeClientForHelm = NewKubernetesClient
