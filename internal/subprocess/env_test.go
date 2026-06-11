package subprocess

import (
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestEnv_passthroughDefault(t *testing.T) {
	t.Setenv("KZERO_SUBPROCESS_ENV_TEST", "visible")

	env := Env(&config.Config{Run: config.RunConfig{Kubeconfig: "/tmp/kc"}})
	if !envHas(env, "KUBECONFIG=/tmp/kc") {
		t.Fatalf("missing kubeconfig: %v", env)
	}
	if !envHas(env, "KZERO_SUBPROCESS_ENV_TEST=visible") {
		t.Fatalf("expected host env passthrough, got %v", env)
	}
}

func TestEnv_noPassthrough(t *testing.T) {
	t.Setenv("HOME", "/tmp/home-test")
	t.Setenv("USER", "operator")

	env := Env(&config.Config{Run: config.RunConfig{NoEnvPassthrough: true, Kubeconfig: "/tmp/kc"}})
	if envHas(env, "HOME=/tmp/home-test") || envHas(env, "USER=operator") {
		t.Fatalf("host env should not pass through: %v", env)
	}
	if !envHas(env, "KUBECONFIG=/tmp/kc") {
		t.Fatalf("missing kubeconfig: %v", env)
	}
}

func envHas(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}
