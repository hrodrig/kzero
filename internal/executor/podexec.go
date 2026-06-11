package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// PodExec runs a command in a pod container via the Kubernetes exec subresource.
type PodExec interface {
	Run(ctx context.Context, step config.PipelineStep) (stdout, stderr []byte, err error)
}

// RemotePodExec uses client-go remotecommand (SPDY).
type RemotePodExec struct {
	client kubernetes.Interface
	config *rest.Config
}

// NewPodExec builds an API-backed pod exec runner from kubeconfig (empty = default rules).
func NewPodExec(kubeconfig string) (*RemotePodExec, error) {
	cc, err := cluster.LoadRESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(rest.CopyConfig(cc))
	if err != nil {
		return nil, err
	}
	return &RemotePodExec{client: client, config: cc}, nil
}

// Run executes command in the named pod/container.
func (r *RemotePodExec) Run(ctx context.Context, step config.PipelineStep) (stdout, stderr []byte, err error) {
	if r == nil || r.client == nil || r.config == nil {
		return nil, nil, fmt.Errorf("pod exec: nil runner")
	}
	if strings.TrimSpace(step.Container) == "" || len(step.Command) == 0 {
		return nil, nil, fmt.Errorf("pod exec %s: container and command required", step.Ref)
	}

	var stdin io.Reader
	if step.Stdin != "" {
		stdin = strings.NewReader(step.Stdin)
	}

	req := r.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(step.Name).
		Namespace(step.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: step.Container,
			Command:   step.Command,
			Stdin:     step.Stdin != "",
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(r.config, "POST", req.URL())
	if err != nil {
		return nil, nil, fmt.Errorf("pod exec %s/%s: %w", step.Namespace, step.Name, err)
	}

	var outBuf, errBuf bytes.Buffer
	streamErr := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &outBuf,
		Stderr: &errBuf,
		Tty:    false,
	})
	stdout, stderr = outBuf.Bytes(), errBuf.Bytes()
	if streamErr != nil {
		return stdout, stderr, WrapPodExec(step, stdout, stderr, streamErr)
	}
	return stdout, stderr, nil
}

// FormatExecPlan returns a short analyze/dry-run label for an exec step.
func FormatExecPlan(step config.PipelineStep) string {
	cmd := strings.Join(step.Command, " ")
	if len(cmd) > 80 {
		cmd = cmd[:77] + "..."
	}
	return fmt.Sprintf("exec %s/%s container=%s: %s", step.Namespace, step.Name, step.Container, cmd)
}

// PodExecError carries pod exec context and combined output.
type PodExecError struct {
	Ref      string
	ExitCode int
	Output   string
	Err      error
}

func (e *PodExecError) Error() string {
	if e.ExitCode >= 0 {
		return fmt.Sprintf("%s failed (exit %d): %s", e.Ref, e.ExitCode, trimOutput(e.Output))
	}
	return fmt.Sprintf("%s failed: %v", e.Ref, e.Err)
}

func (e *PodExecError) Unwrap() error { return e.Err }

// WrapPodExec maps remote exec failures to stable errors where possible.
func WrapPodExec(step config.PipelineStep, stdout, stderr []byte, err error) error {
	if err == nil {
		return nil
	}
	ref := FormatExecPlan(step)
	out := string(stdout) + string(stderr)
	base := &PodExecError{
		Ref:      ref,
		ExitCode: exitCodeFrom(err),
		Output:   out,
		Err:      err,
	}
	msg := strings.ToLower(out + "\n" + err.Error())
	switch {
	case strings.Contains(msg, "not found"), strings.Contains(msg, "notfound"):
		return fmt.Errorf("%s: %w", ref, joinSentinel(ErrNotFound, base))
	case strings.Contains(msg, "forbidden"):
		return fmt.Errorf("%s: %w", ref, joinSentinel(ErrForbidden, base))
	default:
		return fmt.Errorf("%s: %w", ref, base)
	}
}
