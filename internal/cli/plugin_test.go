package cli

import (
	"os"
	"testing"
)

func TestIsPluginInvocation_basenameMatch(t *testing.T) {
	t.Setenv("KZERO_FORCE_KUBECTL_PLUGIN", "")
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"/usr/local/bin/kzero"}
	if IsPluginInvocation() {
		t.Fatal("basename kzero must not be flagged as plugin invocation")
	}

	os.Args = []string{"./kzero"}
	if IsPluginInvocation() {
		t.Fatal("./kzero basename must not be flagged as plugin invocation")
	}

	os.Args = []string{"/opt/krew/bin/kubectl-kzero"}
	if !IsPluginInvocation() {
		t.Fatal("/opt/krew/bin/kubectl-kzero must be flagged as plugin invocation")
	}

	os.Args = []string{"/tmp/kubectl-kzero.exe"}
	if !IsPluginInvocation() {
		t.Fatal("kubectl-kzero.exe must be flagged as plugin invocation")
	}
}

func TestIsPluginInvocation_envOverride(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"/usr/local/bin/kzero"}

	t.Setenv("KZERO_FORCE_KUBECTL_PLUGIN", "")
	if IsPluginInvocation() {
		t.Fatal("empty override must NOT force plugin mode")
	}

	cases := []struct {
		value string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"Yes", true},
		{"yes", true},
		{"0", false},
		{"false", false},
		{"  ", false},
		{"on", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("KZERO_FORCE_KUBECTL_PLUGIN", tc.value)
			if got := IsPluginInvocation(); got != tc.want {
				t.Fatalf("KZERO_FORCE_KUBECTL_PLUGIN=%q: got=%v want=%v", tc.value, got, tc.want)
			}
		})
	}
}

func TestInvocationLabel(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	t.Setenv("KZERO_FORCE_KUBECTL_PLUGIN", "")

	os.Args = []string{"/opt/local/bin/kzero"}
	if got := InvocationLabel(); got != "kzero" {
		t.Fatalf("standalone basename: got %q want kzero", got)
	}

	os.Args = []string{"/opt/krew/bin/kubectl-kzero"}
	if got := InvocationLabel(); got != "kubectl-kzero" {
		t.Fatalf("plugin invocation: got %q want kubectl-kzero", got)
	}

	os.Args = []string{"/very/long/path/mysymlink"}
	if got := InvocationLabel(); got != "mysymlink" {
		t.Fatalf("custom basename: got %q want mysymlink", got)
	}
}
