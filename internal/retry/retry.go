package retry

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/executor"
	"github.com/hrodrig/kzero/internal/log"
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

// Exponential returns the uncapped-then-capped exponential delay before the
// next try after failure number failedTry (1 = first failure), without jitter.
func Exponential(base time.Duration, failedTry int) time.Duration {
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

// Backoff returns a full-jitter wait in [0, Exponential(base, failedTry)]
// (inclusive). Used by the live engine between retries so concurrent
// operators do not retry in lockstep (ROADMAP #50).
func Backoff(base time.Duration, failedTry int) time.Duration {
	return fullJitter(Exponential(base, failedTry))
}

// fullJitter picks uniformly from [0, d] inclusive (AWS "full jitter").
func fullJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	// Int64N panics if n <= 0; d+1 is always >= 1 here.
	return time.Duration(rand.Int64N(int64(d) + 1))
}

// IsRetriable reports whether a pipeline step failure should be retried in live mode.
func IsRetriable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, executor.ErrConflict) ||
		errors.Is(err, executor.ErrTransient) {
		return true
	}
	if errors.Is(err, executor.ErrNotFound) || errors.Is(err, executor.ErrForbidden) {
		return false
	}
	if retriableAPIStatus(err) {
		return true
	}
	return retriableMessage(err.Error())
}

func retriableAPIStatus(err error) bool {
	return apierrors.IsServerTimeout(err) || apierrors.IsTimeout(err) ||
		apierrors.IsTooManyRequests(err) || apierrors.IsServiceUnavailable(err) ||
		apierrors.IsConflict(err)
}

func retriableMessage(msg string) bool {
	msg = strings.ToLower(msg)
	for _, sub := range []string{
		"connection refused",
		"connection reset",
		"connection lost",
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

// LogRetry writes a structured retry line when out is non-nil (legacy text format).
func LogRetry(out io.Writer, cfg *config.Config, phase string, index int, stepRef string, try, max int, wait time.Duration, err error) {
	if out == nil {
		return
	}
	log.New(out, log.FormatText).Retry(cfg, phase, index, stepRef, try, max, wait, err)
}
