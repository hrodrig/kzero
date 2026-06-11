package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hrodrig/kzero/internal/redact"
)

func postJSON(ctx context.Context, client HTTPDoer, url string, headers map[string]string, v any) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("notify: empty webhook URL")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("notify: encode body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("notify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notify: POST %s: %w", redact.URL(url), err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notify: POST %s: HTTP %d", redact.URL(url), resp.StatusCode)
	}
	return nil
}

func postRawJSON(ctx context.Context, client HTTPDoer, url string, headers map[string]string, data []byte) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("notify: empty webhook URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("notify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notify: POST %s: %w", redact.URL(url), err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notify: POST %s: HTTP %d", redact.URL(url), resp.StatusCode)
	}
	return nil
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Error()
	}
	return fmt.Errorf("notify: %s", strings.Join(parts, "; "))
}
