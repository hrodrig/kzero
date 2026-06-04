package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

func TestDescribeStep_customAndRef(t *testing.T) {
	t.Parallel()

	custom := config.PipelineStep{Custom: "./hooks/x.sh"}
	if got := DescribeStep(custom); got != "custom: ./hooks/x.sh" {
		t.Fatalf("custom: got %q", got)
	}

	ref := config.PipelineStep{Ref: "deployment.ns/app", Type: "deployment", Namespace: "ns", Name: "app"}
	if got := DescribeStep(ref); got != "deployment.ns/app" {
		t.Fatalf("ref: got %q", got)
	}
}

func TestFormatStepPlanLine_releaseAndOptions(t *testing.T) {
	t.Parallel()

	three := 3
	step := config.PipelineStep{
		Ref:          "statefulset.db/pg",
		Type:         "statefulset",
		Namespace:    "db",
		Name:         "pg",
		PreStep:      "./hooks/pre.sh",
		Replicas:     &three,
		WaitForReady: true,
		Timeout:      2 * time.Minute,
	}
	got := FormatStepPlanLine(step, "./helm")
	if !strings.Contains(got, "statefulset.db/pg") {
		t.Fatalf("missing ref: %q", got)
	}
	if !strings.Contains(got, "pre: ./hooks/pre.sh") || !strings.Contains(got, "replicas: 3") {
		t.Fatalf("missing options: %q", got)
	}

	release := config.PipelineStep{
		Ref:       "release.mon/prom",
		Type:      "release",
		Namespace: "mon",
		Name:      "prom",
	}
	got = FormatStepPlanLine(release, "./helm-assets")
	if !strings.Contains(got, "script: helm-assets/prom.sh") {
		t.Fatalf("missing script path: %q", got)
	}
}
