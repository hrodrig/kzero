package executor

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CronJobSuspender sets CronJob spec.suspend via the Kubernetes API.
type CronJobSuspender interface {
	Suspend(ctx context.Context, namespace, name string, suspend bool) error
}

// CronJob updates batch/v1 CronJob suspend. When dryRun is true, updates use DryRun=All.
type CronJob struct {
	client kubernetes.Interface
	dryRun bool
}

// NewCronJob builds an API-backed CronJob suspender from kubeconfig (empty = default rules).
func NewCronJob(kubeconfig string) (*CronJob, error) {
	client, err := NewKubernetesClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &CronJob{client: client}, nil
}

// NewCronJobForDryRun builds a suspender that validates updates with DryRun=All.
func NewCronJobForDryRun(kubeconfig string) (*CronJob, error) {
	client, err := NewKubernetesClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return &CronJob{client: client, dryRun: true}, nil
}

// Suspend sets CronJob.spec.suspend. Get+Update (typed batch/v1).
func (c *CronJob) Suspend(ctx context.Context, namespace, name string, suspend bool) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cronjob suspend: nil client")
	}
	cj, err := c.client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("cronjob %s/%s", namespace, name))
	}
	cj.Spec.Suspend = &suspend
	opts := metav1.UpdateOptions{}
	if c.dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}
	_, err = c.client.BatchV1().CronJobs(namespace).Update(ctx, cj, opts)
	if err != nil {
		return WrapAPIError(err, fmt.Sprintf("cronjob %s/%s", namespace, name))
	}
	return nil
}

// FormatCronJobPlan returns a short analyze/dry-run label for a cronjob step.
func FormatCronJobPlan(suspend bool) string {
	if suspend {
		return "patch cronjob spec.suspend=true"
	}
	return "patch cronjob spec.suspend=false"
}
