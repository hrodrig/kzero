// Package doctor runs operator preflight checks without mutating the cluster
// (ROADMAP 0.9.x #49 — kzero doctor).
package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/preflight"
	"github.com/hrodrig/kzero/internal/validate"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Severity ranks a finding.
type Severity string

const (
	SeverityOK    Severity = "ok"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

// Finding is one check result.
type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

// Report is the aggregate doctor output.
type Report struct {
	OK       bool      `json:"ok"`
	Findings []Finding `json:"findings"`
}

// LookPathFunc resolves a binary on PATH (tests inject stubs).
type LookPathFunc func(file string) (string, error)

// Options configures Run.
type Options struct {
	Factory  validate.ClientFactory
	LookPath LookPathFunc
}

// Run executes doctor checks against cfg. factory may be nil.
func Run(ctx context.Context, cfg *config.Config, opts Options) Report {
	var r Report
	if cfg == nil {
		r.Findings = append(r.Findings, Finding{
			Check: "config", Severity: SeverityError, Message: "no config",
		})
		return finalize(r)
	}

	r.Findings = append(r.Findings, Finding{
		Check: "config", Severity: SeverityOK, Message: "config loaded",
	})

	look := opts.LookPath
	if look == nil {
		look = exec.LookPath
	}
	r.Findings = append(r.Findings, checkBinaries(cfg, look)...)

	factory := opts.Factory
	if factory == nil {
		factory = validate.ClientFactoryDefault()
	}

	if err := preflight.Check(ctx, cfg, factory); err != nil {
		r.Findings = append(r.Findings, Finding{
			Check: "kubernetes.api", Severity: SeverityError, Message: err.Error(),
		})
		return finalize(r)
	}
	r.Findings = append(r.Findings, Finding{
		Check: "kubernetes.api", Severity: SeverityOK, Message: "API reachable",
	})

	client, err := factory(cfg.Run.Kubeconfig)
	if err != nil {
		r.Findings = append(r.Findings, Finding{
			Check: "kubernetes.client", Severity: SeverityError, Message: err.Error(),
		})
		return finalize(r)
	}

	r.Findings = append(r.Findings, checkWorkloads(ctx, cfg, factory)...)
	r.Findings = append(r.Findings, checkRBAC(ctx, cfg, client)...)

	return finalize(r)
}

func finalize(r Report) Report {
	r.OK = true
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			r.OK = false
			break
		}
	}
	return r
}

func checkBinaries(cfg *config.Config, look LookPathFunc) []Finding {
	execMode := strings.TrimSpace(cfg.Run.Execution)
	if execMode == "" {
		execMode = "shell"
	}
	needKubectl := execMode == "shell" || execMode == "auto"
	if !needKubectl {
		return []Finding{{
			Check: "binaries.kubectl", Severity: SeverityOK,
			Message: fmt.Sprintf("skipped (run.execution=%s)", execMode),
		}}
	}

	var out []Finding
	kubectl := strings.TrimSpace(cfg.Command.Kubectl)
	if kubectl == "" {
		kubectl = "kubectl"
	}
	if path, err := look(kubectl); err != nil {
		out = append(out, Finding{
			Check: "binaries.kubectl", Severity: SeverityError,
			Message: fmt.Sprintf("%q not found on PATH: %v", kubectl, err),
		})
	} else {
		out = append(out, Finding{
			Check: "binaries.kubectl", Severity: SeverityOK, Message: path,
		})
	}

	if pipelineHasRelease(cfg) && (execMode == "shell" || execMode == "auto") {
		helm := strings.TrimSpace(cfg.Command.Helm)
		if helm == "" {
			helm = "helm"
		}
		if path, err := look(helm); err != nil {
			sev := SeverityWarn
			if execMode == "shell" {
				sev = SeverityError
			}
			out = append(out, Finding{
				Check: "binaries.helm", Severity: sev,
				Message: fmt.Sprintf("%q not found on PATH (release.* steps): %v", helm, err),
			})
		} else {
			out = append(out, Finding{
				Check: "binaries.helm", Severity: SeverityOK, Message: path,
			})
		}
	}
	return out
}

func pipelineHasRelease(cfg *config.Config) bool {
	for _, s := range cfg.Pipelines.Down {
		if s.Type == "release" {
			return true
		}
	}
	for _, s := range cfg.Pipelines.Up {
		if s.Type == "release" {
			return true
		}
	}
	return false
}

func checkWorkloads(ctx context.Context, cfg *config.Config, factory validate.ClientFactory) []Finding {
	lines, skipped, err := validate.CheckPipelineWorkloads(ctx, cfg, factory)
	if skipped != "" {
		return []Finding{{
			Check: "kubernetes.workloads", Severity: SeverityWarn,
			Message: "skipped: " + skipped,
		}}
	}
	if len(lines) == 0 {
		return []Finding{{
			Check: "kubernetes.workloads", Severity: SeverityOK,
			Message: "no deployment/statefulset/pvc/exec refs to check",
		}}
	}
	var out []Finding
	fail := 0
	for _, line := range lines {
		sev := SeverityOK
		msg := line.Detail
		if !line.OK {
			fail++
			sev = SeverityError
			if strings.Contains(strings.ToLower(line.Detail), "forbidden") {
				sev = SeverityError
				msg = "RBAC or missing object: " + line.Detail
			}
		}
		out = append(out, Finding{
			Check: "kubernetes.workloads." + line.Ref, Severity: sev, Message: msg,
		})
	}
	if fail == 0 {
		out = append([]Finding{{
			Check: "kubernetes.workloads", Severity: SeverityOK,
			Message: fmt.Sprintf("%d refs OK", len(lines)),
		}}, out...)
	} else if err != nil {
		out = append([]Finding{{
			Check: "kubernetes.workloads", Severity: SeverityError, Message: err.Error(),
		}}, out...)
	}
	return out
}

type rbacNeed struct {
	group, resource, verb, namespace string
}

func checkRBAC(ctx context.Context, cfg *config.Config, client kubernetes.Interface) []Finding {
	if client == nil {
		return nil
	}
	needs := collectRBACNeeds(cfg)
	if len(needs) == 0 {
		return []Finding{{
			Check: "kubernetes.rbac", Severity: SeverityOK,
			Message: "no scalable workload refs for RBAC hints",
		}}
	}

	var out []Finding
	denied := 0
	for _, n := range needs {
		ok, msg, err := selfSubjectAllowed(ctx, client, n)
		check := fmt.Sprintf("kubernetes.rbac.%s/%s.%s.%s", n.namespace, n.resource, n.verb, n.group)
		if err != nil {
			out = append(out, Finding{Check: check, Severity: SeverityWarn, Message: err.Error()})
			continue
		}
		if !ok {
			denied++
			out = append(out, Finding{Check: check, Severity: SeverityError, Message: msg})
			continue
		}
		out = append(out, Finding{Check: check, Severity: SeverityOK, Message: "allowed"})
	}
	summary := Finding{
		Check: "kubernetes.rbac", Severity: SeverityOK,
		Message: fmt.Sprintf("%d access reviews", len(needs)),
	}
	if denied > 0 {
		summary.Severity = SeverityError
		summary.Message = fmt.Sprintf("%d of %d access reviews denied", denied, len(needs))
	}
	return append([]Finding{summary}, out...)
}

func collectRBACNeeds(cfg *config.Config) []rbacNeed {
	seen := make(map[string]struct{})
	var out []rbacNeed
	add := func(s config.PipelineStep) {
		var resource, group string
		var verbs []string
		switch s.Type {
		case "deployment":
			resource, group = "deployments", "apps"
			verbs = []string{"get", "update", "patch"}
		case "statefulset":
			resource, group = "statefulsets", "apps"
			verbs = []string{"get", "update", "patch"}
		case "pvc":
			resource, group = "persistentvolumeclaims", ""
			verbs = []string{"get", "delete"}
		default:
			return
		}
		if s.Namespace == "" {
			return
		}
		for _, verb := range verbs {
			key := group + "|" + resource + "|" + verb + "|" + s.Namespace
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, rbacNeed{group: group, resource: resource, verb: verb, namespace: s.Namespace})
		}
	}
	for _, s := range cfg.Pipelines.Down {
		add(s)
	}
	for _, s := range cfg.Pipelines.Up {
		add(s)
	}
	return out
}

func selfSubjectAllowed(ctx context.Context, client kubernetes.Interface, n rbacNeed) (bool, string, error) {
	sar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: n.namespace,
				Verb:      n.verb,
				Group:     n.group,
				Resource:  n.resource,
			},
		},
	}
	res, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, "", err
	}
	if !res.Status.Allowed {
		reason := res.Status.Reason
		if reason == "" {
			reason = "not allowed"
		}
		return false, reason, nil
	}
	return true, "", nil
}
