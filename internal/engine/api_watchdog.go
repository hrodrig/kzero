package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	"k8s.io/client-go/rest"
)

func (e *Engine) warnAPIWatchdogDisabled(msg string, err error) {
	if e == nil || e.Log == nil || err == nil {
		return
	}
	e.Log.Emit(log.Entry{
		Kind:  log.KindLive,
		Level: log.LevelWarn,
		Msg:   msg,
		Err:   err.Error(),
	})
}

func newAPIWatchdogProbe(kubeconfig string) (*http.Client, string, error) {
	restCfg, err := cluster.LoadRESTConfig(kubeconfig)
	if err != nil {
		return nil, "", fmt.Errorf("load REST config: %w", err)
	}
	client, err := rest.HTTPClientFor(restCfg)
	if err != nil {
		return nil, "", fmt.Errorf("create HTTP client: %w", err)
	}
	baseURL, _, err := rest.DefaultServerUrlFor(restCfg)
	if err != nil {
		return nil, "", fmt.Errorf("resolve API server URL: %w", err)
	}
	healthzURL := strings.TrimRight(baseURL.String(), "/") + "/healthz"
	return client, healthzURL, nil
}

func probeKubernetesHealthz(ctx context.Context, client *http.Client, healthzURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthzURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz status %d", resp.StatusCode)
	}
	return nil
}

func apiWatchdogTimings(wd *config.APIWatchdogConfig) (interval, failAfter time.Duration) {
	interval = wd.Interval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	failAfter = wd.FailAfter
	if failAfter <= 0 {
		failAfter = 5 * time.Minute
	}
	return interval, failAfter
}
