package executor

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/hrodrig/kzero/internal/cluster"
)

func TestPVC_Delete_removesClaim(t *testing.T) {
	t.Parallel()

	client := fake.NewClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-0", Namespace: "db"},
	})
	p := &PVC{client: client}

	if err := p.Delete(context.Background(), "db", "data-0"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := client.CoreV1().PersistentVolumeClaims("db").Get(context.Background(), "data-0", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected pvc to be deleted")
	}
}

func TestPVC_Delete_ignoreNotFound(t *testing.T) {
	t.Parallel()

	p := &PVC{client: fake.NewClientset()}
	if err := p.Delete(context.Background(), "db", "missing"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestPVC_Delete_wrapsForbidden(t *testing.T) {
	t.Parallel()

	gr := schema.GroupResource{Group: "", Resource: "persistentvolumeclaims"}
	client := fake.NewClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-0", Namespace: "db"},
	})
	client.PrependReactor("delete", "persistentvolumeclaims", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewForbidden(gr, "data-0", errors.New("denied"))
	})
	p := &PVC{client: client}
	err := p.Delete(context.Background(), "db", "data-0")
	if err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if !strings.Contains(err.Error(), "pvc db/data-0") {
		t.Fatalf("expected resource in error, got %v", err)
	}
}

func TestNewPVCDeleter_resolvesKubeconfig(t *testing.T) {
	t.Parallel()

	kc := cluster.TestKubeconfigPath(t)
	p, err := NewPVCDeleter(kc)
	if err != nil {
		t.Fatalf("NewPVCDeleter: %v", err)
	}
	if p == nil || p.client == nil {
		t.Fatal("expected client")
	}
}
