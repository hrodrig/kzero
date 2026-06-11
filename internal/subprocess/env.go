package subprocess

import (
	"os"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/correlation"
)

// Env builds hook, custom, release, and kubectl subprocess environment.
// When run.no_env_passthrough is true, only KZERO_* correlation vars and optional KUBECONFIG are set.
func Env(cfg *config.Config) []string {
	var env []string
	if cfg == nil || !cfg.Run.NoEnvPassthrough {
		env = append(env, os.Environ()...)
	}
	if cfg != nil {
		if k := strings.TrimSpace(cfg.Run.Kubeconfig); k != "" {
			env = append(env, "KUBECONFIG="+k)
		}
	}
	return correlation.AppendEnv(cfg, env)
}
