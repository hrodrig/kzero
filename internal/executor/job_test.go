package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/hrodrig/kzero/internal/cluster"
)

func TestJob_Delete_removesAndIgnoresNotFound(t *testing.T) {
	t.Parallel()

	client := fake.NewClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "migrate", Namespace: "batch"},
	})
	j := &Job{client: client}
	if err := j.Delete(context.Background(), "batch", "migrate"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := client.BatchV1().Jobs("batch").Get(context.Background(), "migrate", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected job deleted")
	}
	if err := j.Delete(context.Background(), "batch", "missing"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestJob_CreateFromManifest_andWaitComplete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "job.yaml")
	body := `
apiVersion: batch/v1
kind: Job
metadata:
  name: from-file
  namespace: other
spec:
  template:
    spec:
      containers:
      - name: c
        image: busybox
      restartPolicy: Never
`
	if err := os.WriteFile(manifest, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	client := fake.NewClientset()
	client.PrependReactor("get", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		ga, ok := action.(ktesting.GetAction)
		if !ok {
			return false, nil, nil
		}
		return true, &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: ga.GetName(), Namespace: ga.GetNamespace()},
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
				}},
			},
		}, nil
	})

	j := &Job{client: client}
	if err := j.CreateFromManifest(context.Background(), "batch", "migrate-db", manifest); err != nil {
		t.Fatalf("CreateFromManifest: %v", err)
	}
	created, err := client.BatchV1().Jobs("batch").Get(context.Background(), "migrate-db", metav1.GetOptions{})
	if err != nil {
		// Get reactor overrides — check create action instead
		found := false
		for _, a := range client.Actions() {
			if a.GetVerb() == "create" && a.GetResource().Resource == "jobs" {
				found = true
				ca := a.(ktesting.CreateAction)
				job := ca.GetObject().(*batchv1.Job)
				if job.Name != "migrate-db" || job.Namespace != "batch" {
					t.Fatalf("created job meta: %s/%s", job.Namespace, job.Name)
				}
			}
		}
		if !found {
			t.Fatalf("create not found; get err=%v", err)
		}
	} else if created.Name != "migrate-db" {
		t.Fatalf("name=%s", created.Name)
	}

	if err := j.WaitComplete(context.Background(), "batch", "migrate-db", time.Second); err != nil {
		t.Fatalf("WaitComplete: %v", err)
	}
}

func TestJob_CreateFromManifest_dryRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "job.yaml")
	body := `
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
spec:
  template:
    spec:
      containers:
      - name: c
        image: busybox
      restartPolicy: Never
`
	if err := os.WriteFile(manifest, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	client := fake.NewClientset()
	j := &Job{client: client, dryRun: true}
	if err := j.CreateFromManifest(context.Background(), "batch", "migrate", manifest); err != nil {
		t.Fatalf("Create dry-run: %v", err)
	}
	if !createActionHasDryRunAll(client.Actions(), "jobs") {
		t.Fatal("expected Create with DryRun=All on jobs")
	}
}

func TestJob_CreateFromManifest_rejectsNonJob(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifest := filepath.Join(dir, "pod.yaml")
	body := `
apiVersion: v1
kind: Pod
metadata:
  name: x
spec:
  containers:
  - name: c
    image: busybox
`
	if err := os.WriteFile(manifest, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	j := &Job{client: fake.NewClientset()}
	err := j.CreateFromManifest(context.Background(), "batch", "x", manifest)
	if err == nil {
		t.Fatal("expected error for non-Job manifest")
	}
	if !strings.Contains(err.Error(), "Job") && !strings.Contains(err.Error(), "Pod") {
		t.Fatalf("expected kind-related error, got %v", err)
	}
}

func TestJob_WaitComplete_failsOnFailedCondition(t *testing.T) {
	t.Parallel()

	client := fake.NewClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "migrate", Namespace: "batch"},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:    batchv1.JobFailed,
				Status:  corev1.ConditionTrue,
				Message: "BackoffLimitExceeded",
			}},
		},
	})
	j := &Job{client: client}
	err := j.WaitComplete(context.Background(), "batch", "migrate", time.Second)
	if err == nil || !strings.Contains(err.Error(), "BackoffLimitExceeded") {
		t.Fatalf("expected failed job error, got %v", err)
	}
}

func TestJob_Delete_wrapsForbidden(t *testing.T) {
	t.Parallel()

	gr := schema.GroupResource{Group: "batch", Resource: "jobs"}
	client := fake.NewClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "migrate", Namespace: "batch"},
	})
	client.PrependReactor("delete", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(gr, "migrate", errors.New("denied"))
	})
	j := &Job{client: client}
	err := j.Delete(context.Background(), "batch", "migrate")
	if err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestFormatJobPlan(t *testing.T) {
	t.Parallel()
	if got := FormatJobPlan(true, "", true); !strings.Contains(got, "delete job") {
		t.Fatalf("down: %q", got)
	}
	if got := FormatJobPlan(false, "./m.yaml", true); !strings.Contains(got, "./m.yaml") || !strings.Contains(got, "wait_for_complete=true") {
		t.Fatalf("up wait: %q", got)
	}
	if got := FormatJobPlan(false, "", false); !strings.Contains(got, "wait_for_complete=false") {
		t.Fatalf("up no wait: %q", got)
	}
}

func TestNewJobForDryRun_resolvesKubeconfig(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	j, err := NewJobForDryRun(kc)
	if err != nil {
		t.Fatalf("NewJobForDryRun: %v", err)
	}
	if j == nil || !j.dryRun {
		t.Fatal("expected dryRun client")
	}
}

func TestDecodeJobManifest_rejectsEmptyJobList(t *testing.T) {
	t.Parallel()

	data := []byte(`
apiVersion: batch/v1
kind: JobList
items: []
`)
	_, err := decodeJobManifest(data)
	if err == nil || !strings.Contains(err.Error(), "exactly one Job") {
		t.Fatalf("expected JobList size error, got %v", err)
	}
}

func TestDecodeJobManifest_JobList(t *testing.T) {
	t.Parallel()

	data := []byte(`
apiVersion: batch/v1
kind: JobList
items:
- apiVersion: batch/v1
  kind: Job
  metadata:
    name: only
  spec:
    template:
      spec:
        containers:
        - name: c
          image: busybox
        restartPolicy: Never
`)
	job, err := decodeJobManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if job.Name != "only" {
		t.Fatalf("name=%s", job.Name)
	}
}

func createActionHasDryRunAll(actions []ktesting.Action, resource string) bool {
	for _, a := range actions {
		if a.GetVerb() != "create" || a.GetResource().Resource != resource {
			continue
		}
		ca, ok := a.(ktesting.CreateActionImpl)
		if !ok {
			continue
		}
		for _, d := range ca.GetCreateOptions().DryRun {
			if d == metav1.DryRunAll {
				return true
			}
		}
	}
	return false
}
