package executor

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
)

// RunFunc runs a subprocess; same contract as engine.LiveExec.
type RunFunc func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error)

// ShellDeps supplies kubectl invocation for Shell workload steps.
type ShellDeps struct {
	Kubectl    string
	Kubeconfig string
	Run        RunFunc
	WriteOut   func([]byte)
}

// Shell scales and waits via kubectl subprocesses.
type Shell struct {
	deps ShellDeps
}

// NewShell returns a kubectl-backed Workload executor.
func NewShell(deps ShellDeps) *Shell {
	return &Shell{deps: deps}
}

func (s *Shell) Scale(ctx context.Context, kind, namespace, name string, replicas int32) error {
	if _, ok := scalableKinds[kind]; !ok {
		return fmt.Errorf("shell: unsupported resource kind %q for scale", kind)
	}
	bin := s.deps.Kubectl
	if strings.TrimSpace(bin) == "" {
		bin = "kubectl"
	}
	args := []string{
		"scale",
		fmt.Sprintf("%s/%s", kind, name),
		"-n", namespace,
		"--replicas", strconv.FormatInt(int64(replicas), 10),
	}
	out, err := s.deps.Run(ctx, bin, args, s.env(), ".")
	s.write(out)
	if err != nil {
		return fmt.Errorf("kubectl scale %s/%s in %s: %w", kind, name, namespace, WrapSubprocess(bin, args, out, err))
	}
	return nil
}

func (s *Shell) WaitRollout(ctx context.Context, kind, namespace, name string, timeout time.Duration) error {
	bin := s.deps.Kubectl
	if strings.TrimSpace(bin) == "" {
		bin = "kubectl"
	}
	args := []string{
		"rollout", "status",
		fmt.Sprintf("%s/%s", kind, name),
		"-n", namespace,
		"--timeout", timeout.String(),
	}
	out, err := s.deps.Run(ctx, bin, args, s.env(), ".")
	s.write(out)
	if err != nil {
		return fmt.Errorf("kubectl rollout status %s/%s in %s: %w", kind, name, namespace, WrapSubprocess(bin, args, out, err))
	}
	return nil
}

func (s *Shell) env() []string {
	env := os.Environ()
	if k := strings.TrimSpace(s.deps.Kubeconfig); k != "" {
		env = append(env, "KUBECONFIG="+k)
	}
	return env
}

func (s *Shell) write(out []byte) {
	if len(out) > 0 && s.deps.WriteOut != nil {
		s.deps.WriteOut(out)
	}
}

// ShellFromConfig builds Shell deps from a loaded config and run hook.
func ShellFromConfig(cfg *config.Config, run RunFunc, writeOut func([]byte)) *Shell {
	kubectl := "kubectl"
	if cfg != nil && strings.TrimSpace(cfg.Command.Kubectl) != "" {
		kubectl = cfg.Command.Kubectl
	}
	kc := ""
	if cfg != nil {
		kc = cfg.Run.Kubeconfig
	}
	return NewShell(ShellDeps{
		Kubectl:    kubectl,
		Kubeconfig: kc,
		Run:        run,
		WriteOut:   writeOut,
	})
}

var scalableKinds = map[string]struct{}{
	"deployment":  {},
	"statefulset": {},
}
