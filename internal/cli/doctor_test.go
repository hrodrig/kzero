package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/doctor"
	"github.com/hrodrig/kzero/internal/validate"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestDoctor_okJSON(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	})
	client.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})
	old := validate.SwapDefaultClientFactory(func(string) (kubernetes.Interface, error) { return client, nil })
	t.Cleanup(func() { validate.SwapDefaultClientFactory(old) })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: dry-run
  execution: native
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--config", cfgPath, "--output", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	var rep doctor.Report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("json: %v body=%q", err, stdout.String())
	}
	if !rep.OK {
		t.Fatalf("expected ok report: %+v", rep)
	}
}

func TestDoctor_textOutput(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	})
	client.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})
	old := validate.SwapDefaultClientFactory(func(string) (kubernetes.Interface, error) { return client, nil })
	t.Cleanup(func() { validate.SwapDefaultClientFactory(old) })

	cfgPath := filepath.Join(t.TempDir(), "kzero.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - deployment.ns/app
run:
  mode: dry-run
  execution: native
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"doctor", "--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor: %v\n%s", err, stdout.String())
	}
	out := stdout.String()
	for _, want := range []string{"Doctor: OK", "[OK   ]", "config"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestDoctorSeverityLabel(t *testing.T) {
	t.Parallel()
	if got := doctorSeverityLabel(doctor.SeverityError); got != "ERROR" {
		t.Fatalf("got %q", got)
	}
	if got := doctorSeverityLabel(doctor.SeverityWarn); got != "WARN " {
		t.Fatalf("got %q", got)
	}
	if got := doctorSeverityLabel(doctor.SeverityOK); got != "OK   " {
		t.Fatalf("got %q", got)
	}
	if got := doctorSeverityLabel(doctor.Severity("x")); got != "????" {
		t.Fatalf("got %q", got)
	}
}

func TestDoctor_badConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
schema_version: "1.0"
pipelines:
  down:
    - "not-valid"
run:
  mode: dry-run
  execution: shell
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"doctor", "--config", cfgPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected config error")
	}
	if !strings.Contains(err.Error(), "doctor config") && !strings.Contains(err.Error(), "pipeline") {
		t.Fatalf("unexpected error: %v", err)
	}
}
