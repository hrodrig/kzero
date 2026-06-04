package executor

import (
	"context"
	"time"
)

// Workload performs scale and rollout-wait for deployment and statefulset steps.
type Workload interface {
	Scale(ctx context.Context, kind, namespace, name string, replicas int32) error
	WaitRollout(ctx context.Context, kind, namespace, name string, timeout time.Duration) error
}
