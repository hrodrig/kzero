package executor

import (
	"context"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var jobScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(batchv1.AddToScheme(jobScheme))
}

// JobRunner deletes Jobs and creates them from manifests via the Kubernetes API.
type JobRunner interface {
	Delete(ctx context.Context, namespace, name string) error
	CreateFromManifest(ctx context.Context, namespace, name, manifestPath string) error
	WaitComplete(ctx context.Context, namespace, name string, timeout time.Duration) error
}

// Job manages batch/v1 Jobs. When dryRun is true, Create uses DryRun=All (no wait).
type Job struct {
	client kubernetes.Interface
	dryRun bool
}

// NewJob builds an API-backed Job runner from kubeconfig (empty = default rules).
func NewJob(kubeconfig string) (*Job, error) {
	client, err := NewKubernetesClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &Job{client: client}, nil
}

// NewJobForDryRun builds a runner that validates Create with DryRun=All.
func NewJobForDryRun(kubeconfig string) (*Job, error) {
	client, err := NewKubernetesClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &Job{client: client, dryRun: true}, nil
}

// Delete removes the Job with background propagation; missing Jobs are no-ops.
func (j *Job) Delete(ctx context.Context, namespace, name string) error {
	if j == nil || j.client == nil {
		return fmt.Errorf("job delete: nil client")
	}
	propagation := metav1.DeletePropagationBackground
	err := j.client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return WrapAPIError(err, fmt.Sprintf("job %s/%s", namespace, name))
	}
	return nil
}

// CreateFromManifest decodes a single batch/v1 Job (or JobList with one Job) and creates it.
// Step namespace/name override metadata when set.
func (j *Job) CreateFromManifest(ctx context.Context, namespace, name, manifestPath string) error {
	if j == nil || j.client == nil {
		return fmt.Errorf("job create: nil client")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("job create: read manifest %q: %w", manifestPath, err)
	}
	job, err := decodeJobManifest(data)
	if err != nil {
		return fmt.Errorf("job create: %w", err)
	}
	if namespace != "" {
		job.Namespace = namespace
	}
	if name != "" {
		job.Name = name
	}
	if job.Namespace == "" || job.Name == "" {
		return fmt.Errorf("job create: namespace and name required (step or manifest metadata)")
	}
	clearJobIdentity(job)

	opts := metav1.CreateOptions{}
	if j.dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}
	_, err = j.client.BatchV1().Jobs(job.Namespace).Create(ctx, job, opts)
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("job %s/%s", job.Namespace, job.Name))
	}
	return nil
}

// WaitComplete polls until Job Complete=True, Failed=True, or timeout/ctx cancel.
func (j *Job) WaitComplete(ctx context.Context, namespace, name string, timeout time.Duration) error {
	if j == nil || j.client == nil {
		return fmt.Errorf("job wait: nil client")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	interval := 2 * time.Second
	return wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		job, err := j.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, WrapAPIError(err, fmt.Sprintf("job %s/%s", namespace, name))
		}
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
				return true, nil
			}
			if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
				msg := c.Message
				if msg == "" {
					msg = "job failed"
				}
				return false, fmt.Errorf("job %s/%s: %s", namespace, name, msg)
			}
		}
		return false, nil
	})
}

func decodeJobManifest(data []byte) (*batchv1.Job, error) {
	dec := serializer.NewCodecFactory(jobScheme).UniversalDeserializer()
	obj, gvk, err := dec.Decode(data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	switch j := obj.(type) {
	case *batchv1.Job:
		return j.DeepCopy(), nil
	case *batchv1.JobList:
		if len(j.Items) != 1 {
			return nil, fmt.Errorf("manifest JobList must contain exactly one Job, got %d", len(j.Items))
		}
		return j.Items[0].DeepCopy(), nil
	default:
		kind := "unknown"
		if gvk != nil {
			kind = gvk.GroupVersion().String() + " " + gvk.Kind
		}
		return nil, fmt.Errorf("manifest must be batch/v1 Job (got %s)", kind)
	}
}

func clearJobIdentity(job *batchv1.Job) {
	job.ResourceVersion = ""
	job.UID = ""
	job.Generation = 0
	job.CreationTimestamp = metav1.Time{}
	job.ManagedFields = nil
	job.Status = batchv1.JobStatus{}
}

// FormatJobPlan returns a short analyze/dry-run label for a job step.
func FormatJobPlan(down bool, manifest string, waitComplete bool) string {
	if down {
		return "delete job (background propagation, ignore-not-found)"
	}
	wait := "wait_for_complete=true"
	if !waitComplete {
		wait = "wait_for_complete=false"
	}
	if manifest == "" {
		return "create job from manifest (" + wait + ")"
	}
	return fmt.Sprintf("create job from manifest %s (%s)", manifest, wait)
}
