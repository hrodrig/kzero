package executor

import (
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestWrapAPIError_sentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want error
	}{
		{"not found", apierrors.NewNotFound(schema.GroupResource{Resource: "deployments"}, "x"), ErrNotFound},
		{"forbidden", apierrors.NewForbidden(schema.GroupResource{Resource: "deployments"}, "x", errors.New("denied")), ErrForbidden},
		{"conflict", apierrors.NewConflict(schema.GroupResource{Resource: "deployments"}, "x", errors.New("conflict")), ErrConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := WrapAPIError(tc.err, "deployment ns/x")
			if !errors.Is(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestWrapAPIError_nilAndGeneric(t *testing.T) {
	t.Parallel()

	if WrapAPIError(nil, "r") != nil {
		t.Fatal("nil")
	}
	err := WrapAPIError(errors.New("boom"), "deployment ns/app")
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("generic: %v", err)
	}
}
