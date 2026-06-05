package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
	"github.com/hrodrig/kzero/internal/executor"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const maxBackoff = 2 * time.Minute

// Attempts returns total tries for a pipeline step (minimum 1).
func Attempts(cfg *config.Config) int {
	if cfg == nil || cfg.Retry.Attempts < 1 {
		return 1
	}
	return cfg.Retry.Attempts
}

// Backoff returns wait time before the next try after failure number failedTry (1 = first failure).
func Backoff(base time.Duration, failedTry int) time.Duration {
	if base <= 0 {
		base = 5 * time.Second
	}
	if failedTry < 1 {
		failedTry = 1
	}
	d := base
	for i := 1; i < failedTry; i++ {
		d *= 2
		if d >= maxBackoff {
			return maxBackoff
		}
	}
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// IsRetriable reports whether a pipeline step failure should be retried in live mode.
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, executor.ErrConflict) {
		return true
	}
	if errors.Is(err, executor.ErrNotFound) || errors.Is(err, executor.ErrForbidden) {
		return false
	}
	if apierrors.IsServerTimeout(err) || apierrors.IsTimeout(err) ||
		apierrors.IsTooManyRequests(err) || apierrors.IsServiceUnavailable(err) ||
		apierrors.IsConflict(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, sub := range []string{
		"connection refused",
		"connection reset",
		"i/o timeout",
		"tls handshake timeout",
		"deadline exceeded",
		"too many requests",
		"temporarily unavailable",
		"server timeout",
		"timeout",
		"429",
		"503",
		"504",
	} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}

// LogRetry writes a structured retry line when out is non-nil.
func LogRetry(out io.Writer, cfg *config.Config, phase string, index int, stepRef string, try, max int, wait time.Duration, err error) {
	if out == nil {
		return
	}
	ref := stepRef
	if ref == "" {
		ref = fmt.Sprintf("index %d", index)
	}
	_, _ = fmt.Fprintf(out, "[retry] %spipeline %s step %s attempt %d/%d failed (%v); retrying in %s\n",
		correlation.LogPrefix(cfg), phase, ref, try, max, err, wait.Round(time.Millisecond))
}
