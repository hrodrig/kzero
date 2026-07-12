package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/validate"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestRun_configAndBinariesShell(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run", Execution: "shell"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	})
	allowAllSAR(client)

	report := Run(context.Background(), cfg, Options{
		Factory: func(string) (kubernetes.Interface, error) { return client, nil },
		LookPath: func(file string) (string, error) {
			if file == "kubectl" {
				return "/usr/bin/kubectl", nil
			}
			return "", errors.New("not found")
		},
	})
	if !report.OK {
		t.Fatalf("expected OK, findings=%+v", report.Findings)
	}
	assertHas(t, report, "binaries.kubectl", SeverityOK)
	assertHas(t, report, "kubernetes.api", SeverityOK)
}

func TestRun_missingKubectlFails(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run", Execution: "shell"},
	}
	client := fake.NewSimpleClientset()
	allowAllSAR(client)

	rep := Run(context.Background(), cfg, Options{
		Factory:  func(string) (kubernetes.Interface, error) { return client, nil },
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
	})
	if rep.OK {
		t.Fatal("expected failure when kubectl missing")
	}
	assertHas(t, rep, "binaries.kubectl", SeverityError)
}

func TestRun_nativeSkipsKubectl(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run", Execution: "native"},
	}
	client := fake.NewSimpleClientset()
	allowAllSAR(client)

	rep := Run(context.Background(), cfg, Options{
		Factory:  func(string) (kubernetes.Interface, error) { return client, nil },
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
	})
	if !rep.OK {
		t.Fatalf("native should skip kubectl binary check, got %+v", rep.Findings)
	}
	assertHas(t, rep, "binaries.kubectl", SeverityOK)
}

func TestRun_apiUnreachable(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Run: config.RunConfig{Mode: "live", Execution: "native"}}
	rep := Run(context.Background(), cfg, Options{
		Factory: func(string) (kubernetes.Interface, error) {
			return nil, errors.New("no kubeconfig")
		},
		LookPath: func(string) (string, error) { return "/bin/x", nil },
	})
	if rep.OK {
		t.Fatal("expected API failure")
	}
	assertHas(t, rep, "kubernetes.api", SeverityError)
}

func TestRun_rbacDenied(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Run: config.RunConfig{Mode: "dry-run", Execution: "native"},
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{
				{Type: "deployment", Namespace: "ns", Name: "app"},
			},
		},
	}
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	})
	client.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{Allowed: false, Reason: "denied in test"},
		}, nil
	})

	rep := Run(context.Background(), cfg, Options{
		Factory:  func(string) (kubernetes.Interface, error) { return client, nil },
		LookPath: func(string) (string, error) { return "/bin/x", nil },
	})
	if rep.OK {
		t.Fatal("expected RBAC denial to fail report")
	}
	assertHas(t, rep, "kubernetes.rbac", SeverityError)
}

func TestCollectRBACNeeds(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Pipelines: config.PipelinesConfig{
			Down: []config.PipelineStep{{Type: "deployment", Namespace: "a", Name: "x"}},
			Up:   []config.PipelineStep{{Type: "pvc", Namespace: "b", Name: "data"}},
		},
	}
	needs := collectRBACNeeds(cfg)
	if len(needs) < 4 {
		t.Fatalf("expected multiple RBAC needs, got %d", len(needs))
	}
}

func allowAllSAR(client *fake.Clientset) {
	client.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})
}

func assertHas(t *testing.T, rep Report, check string, sev Severity) {
	t.Helper()
	for _, f := range rep.Findings {
		if f.Check == check || strings.HasPrefix(f.Check, check+".") {
			if f.Check == check && f.Severity == sev {
				return
			}
			if f.Check == check {
				t.Fatalf("check %s: severity %s want %s (%s)", check, f.Severity, sev, f.Message)
			}
		}
	}
	for _, f := range rep.Findings {
		if f.Check == check {
			if f.Severity != sev {
				t.Fatalf("check %s: severity %s want %s (%s)", check, f.Severity, sev, f.Message)
			}
			return
		}
	}
	t.Fatalf("missing check %s in %+v", check, rep.Findings)
}

// Ensure validate.ClientFactory type stays wired.
var _ validate.ClientFactory = func(string) (kubernetes.Interface, error) { return nil, nil }
