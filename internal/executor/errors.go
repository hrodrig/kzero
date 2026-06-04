package executor

import (
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Sentinel errors for typed API failures (use errors.Is in hooks or tests).
var (
	ErrNotFound    = errors.New("kubernetes resource not found")
	ErrForbidden   = errors.New("kubernetes api forbidden")
	ErrConflict    = errors.New("kubernetes api conflict")
	ErrUnsupported = errors.New("unsupported workload kind for native execution")
)

// WrapAPIError maps client-go errors to stable sentinels while preserving the cause chain.
func WrapAPIError(err error, resource string) error {
	if err == nil {
		return nil
	}
	switch {
	case apierrors.IsNotFound(err):
		return fmt.Errorf("%s: %w", resource, joinSentinel(ErrNotFound, err))
	case apierrors.IsForbidden(err):
		return fmt.Errorf("%s: %w", resource, joinSentinel(ErrForbidden, err))
	case apierrors.IsConflict(err):
		return fmt.Errorf("%s: %w", resource, joinSentinel(ErrConflict, err))
	default:
		return fmt.Errorf("%s: %w", resource, err)
	}
}

func joinSentinel(sentinel, cause error) error {
	return fmt.Errorf("%w: %v", sentinel, cause)
}
