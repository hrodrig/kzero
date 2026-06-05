package preflight

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheck_ok(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{}}
	factory := func(string) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
	if err := Check(context.Background(), cfg, factory); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestCheck_kubeconfigError(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{Kubeconfig: "/missing"}}
	factory := func(string) (kubernetes.Interface, error) { return nil, errors.New("load failed") }
	err := Check(context.Background(), cfg, factory)
	if err == nil || !strings.Contains(err.Error(), "cannot load kubeconfig") {
		t.Fatalf("expected kubeconfig error, got %v", err)
	}
}

func TestAnalyzeWarning_onFailure(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{}}
	factory := func(string) (kubernetes.Interface, error) { return nil, errors.New("unreachable") }
	w := AnalyzeWarning(context.Background(), cfg, factory)
	if w == "" || !strings.Contains(w, "preflight would fail") {
		t.Fatalf("expected warning, got %q", w)
	}
}

func TestAnalyzeWarning_ok(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Run: config.RunConfig{}}
	factory := func(string) (kubernetes.Interface, error) { return fake.NewSimpleClientset(), nil }
	if w := AnalyzeWarning(context.Background(), cfg, factory); w != "" {
		t.Fatalf("expected no warning, got %q", w)
	}
}
