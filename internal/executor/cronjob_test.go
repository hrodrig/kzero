package executor

import (
	"context"
	"errors"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/hrodrig/kzero/internal/cluster"
)

func TestCronJob_Suspend_setsTrueAndFalse(t *testing.T) {
	t.Parallel()

	suspend := false
	client := fake.NewClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "batch"},
		Spec:       batchv1.CronJobSpec{Suspend: &suspend},
	})
	c := &CronJob{client: client}

	if err := c.Suspend(context.Background(), "batch", "nightly", true); err != nil {
		t.Fatalf("Suspend true: %v", err)
	}
	got, err := client.BatchV1().CronJobs("batch").Get(context.Background(), "nightly", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Suspend == nil || !*got.Spec.Suspend {
		t.Fatalf("suspend=%v, want true", got.Spec.Suspend)
	}

	if err := c.Suspend(context.Background(), "batch", "nightly", false); err != nil {
		t.Fatalf("Suspend false: %v", err)
	}
	got, err = client.BatchV1().CronJobs("batch").Get(context.Background(), "nightly", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Suspend == nil || *got.Spec.Suspend {
		t.Fatalf("suspend=%v, want false", got.Spec.Suspend)
	}
}

func TestCronJob_Suspend_dryRunAll(t *testing.T) {
	t.Parallel()

	suspend := false
	client := fake.NewClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "batch"},
		Spec:       batchv1.CronJobSpec{Suspend: &suspend},
	})
	c := &CronJob{client: client, dryRun: true}
	if err := c.Suspend(context.Background(), "batch", "nightly", true); err != nil {
		t.Fatalf("Suspend dry-run: %v", err)
	}
	if !updateActionHasDryRunAll(client.Actions(), "cronjobs") {
		t.Fatal("expected Update with DryRun=All on cronjobs")
	}
}

func TestCronJob_Suspend_wrapsNotFound(t *testing.T) {
	t.Parallel()

	c := &CronJob{client: fake.NewClientset()}
	err := c.Suspend(context.Background(), "batch", "missing", true)
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCronJob_Suspend_wrapsForbidden(t *testing.T) {
	t.Parallel()

	gr := schema.GroupResource{Group: "batch", Resource: "cronjobs"}
	client := fake.NewClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "batch"},
	})
	client.PrependReactor("update", "cronjobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(gr, "nightly", errors.New("denied"))
	})
	c := &CronJob{client: client}
	err := c.Suspend(context.Background(), "batch", "nightly", true)
	if err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "cronjob batch/nightly") {
		t.Fatalf("expected resource in error, got %v", err)
	}
}

func TestFormatCronJobPlan(t *testing.T) {
	t.Parallel()
	if got := FormatCronJobPlan(true); !strings.Contains(got, "suspend=true") {
		t.Fatalf("got %q", got)
	}
	if got := FormatCronJobPlan(false); !strings.Contains(got, "suspend=false") {
		t.Fatalf("got %q", got)
	}
}

func TestNewCronJobForDryRun_resolvesKubeconfig(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	c, err := NewCronJobForDryRun(kc)
	if err != nil {
		t.Fatalf("NewCronJobForDryRun: %v", err)
	}
	if c == nil || !c.dryRun {
		t.Fatal("expected dryRun client")
	}
}
