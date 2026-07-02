package cluster

import "testing"

func TestSanitizeForFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"develop-cluster", "develop-cluster"},
		{"Develop Cluster", "develop-cluster"},
		{"foo/bar:baz", "foo-bar-baz"},
		{"QA.Cluster_01", "qa-cluster-01"},
		{"", "unknown"},
		{"   ", "unknown"},
		{"---", "unknown"},
		{"///", "unknown"},
		{"in-cluster", "in-cluster"},
	}
	for _, tc := range tests {
		got := SanitizeForFilename(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeForFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
