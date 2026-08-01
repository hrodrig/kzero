package engine

import (
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDesiredReplicas(t *testing.T) {
	t.Parallel()

	n := 3
	step := config.PipelineStep{Replicas: &n}
	if got := DesiredReplicas(PhaseDown, step); got != 0 {
		t.Fatalf("down=%d", got)
	}
	if got := DesiredReplicas(PhaseUp, step); got != 3 {
		t.Fatalf("up with replicas=%d", got)
	}
	if got := DesiredReplicas(PhaseUp, config.PipelineStep{}); got != 1 {
		t.Fatalf("up default=%d", got)
	}
}
