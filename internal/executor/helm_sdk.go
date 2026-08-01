package executor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"helm.sh/helm/v4/pkg/action"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/cli/values"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/registry"
	helmrelease "helm.sh/helm/v4/pkg/release"
	releasecommon "helm.sh/helm/v4/pkg/release/common"
	"helm.sh/helm/v4/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SDKHelm runs release steps via helm.sh/helm/v4 (no host helm binary or .sh).
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
	client.WaitStrategy = kube.StatusWatcherStrategy
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
	client.WaitStrategy = waitStrategy(spec.Wait)
	client.Timeout = spec.Timeout
	client.Namespace = step.Namespace
	client.ChartPathOptions.Version = spec.Version

	chartRef := resolveChartRef(h.cfg, spec.Chart)
	regClient, err := h.prepareOCIRegistry(client, chartRef, step.Ref)
	if err != nil {
		return err
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

	return h.runInstallOrUpgrade(ctx, actionConfig, client, step, spec, ch, vals, regClient)
}

func (h *SDKHelm) prepareOCIRegistry(client *action.Upgrade, chartRef, stepRef string) (*registry.Client, error) {
	if !strings.HasPrefix(chartRef, "oci://") {
		return nil, nil
	}
	regClient, err := NewHelmRegistryClient()
	if err != nil {
		return nil, fmt.Errorf("helm sdk registry client for %s: %w", stepRef, err)
	}
	client.SetRegistryClient(regClient)
	if err := EnsureOCIRegistryAuth(h.cfg, chartRef, regClient, h.loggedIn, &h.loginMu); err != nil {
		return nil, fmt.Errorf("helm sdk auth for %s: %w", stepRef, err)
	}
	return regClient, nil
}

func (h *SDKHelm) runInstallOrUpgrade(
	ctx context.Context,
	actionConfig *action.Configuration,
	client *action.Upgrade,
	step config.PipelineStep,
	spec ChartSpec,
	ch *chart.Chart,
	vals map[string]any,
	regClient *registry.Client,
) error {
	needsInstall, err := releaseNeedsInstall(actionConfig, step.Name)
	if err != nil {
		return fmt.Errorf("helm sdk history %s/%s: %w", step.Namespace, step.Name, err)
	}
	if needsInstall {
		return h.helmInstall(ctx, actionConfig, step, spec, ch, vals, regClient, client.ChartPathOptions)
	}
	_, err = client.RunWithContext(ctx, step.Name, ch, vals)
	if err != nil {
		return fmt.Errorf("helm sdk upgrade --install %s/%s: %w", step.Namespace, step.Name, err)
	}
	return nil
}

func (h *SDKHelm) helmInstall(
	ctx context.Context,
	actionConfig *action.Configuration,
	step config.PipelineStep,
	spec ChartSpec,
	ch *chart.Chart,
	vals map[string]any,
	regClient *registry.Client,
	chartPathOpts action.ChartPathOptions,
) error {
	inst := action.NewInstall(actionConfig)
	inst.ReleaseName = step.Name
	inst.CreateNamespace = spec.CreateNamespace
	inst.WaitStrategy = waitStrategy(spec.Wait)
	inst.Timeout = spec.Timeout
	inst.Namespace = step.Namespace
	inst.ChartPathOptions = chartPathOpts
	if regClient != nil {
		inst.SetRegistryClient(regClient)
	}
	if _, err := inst.RunWithContext(ctx, ch, vals); err != nil {
		return fmt.Errorf("helm sdk install %s/%s: %w", step.Namespace, step.Name, err)
	}
	return nil
}

func releaseNeedsInstall(actionConfig *action.Configuration, name string) (bool, error) {
	hist := action.NewHistory(actionConfig)
	hist.Max = 1
	versions, err := hist.Run(name)
	if errors.Is(err, driver.ErrReleaseNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if len(versions) == 0 {
		return true, nil
	}
	acc, err := helmrelease.NewAccessor(versions[0])
	if err != nil {
		return false, err
	}
	return acc.Status() == releasecommon.StatusUninstalled.String(), nil
}

func waitStrategy(wait bool) kube.WaitStrategy {
	if wait {
		return kube.StatusWatcherStrategy
	}
	// Helm v4 requires an explicit strategy; HookOnly skips waiting on chart resources.
	return kube.HookOnlyStrategy
}

func (h *SDKHelm) actionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), namespace, "secret"); err != nil {
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
