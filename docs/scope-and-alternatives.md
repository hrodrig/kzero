# Scope and alternatives

This page compares **kzero** to tools operators often consider for similar-sounding problems. Comparisons are **strictly by primary use case and scope** — not “which is better overall.”

**kzero in one line:** declarative **`down` / `up` / `reset`** pipelines for **workloads already on a cluster**, run **out-of-band** from a bastion (recommended), with ordered steps, hooks, fail-fast, and maintenance-oriented safety (preflight, infra probe, API watchdog).

Full contract: [SPECIFICATIONS.md](../SPECIFICATIONS.md). Where to run: [deployment-models.md](deployment-models.md).

---

## When kzero is the right tool

| You need | kzero provides |
|----------|----------------|
| A **repeatable platform maintenance** playbook (scale down → wipe data → Helm uninstall → bring back up) | Phased **`pipelines.down` / `pipelines.up`**, **`reset`** |
| **Explicit ordering** across Deployments, StatefulSets, Helm releases, PVC delete, in-pod exec | Sequential steps in YAML; no embedded product playbooks |
| **Dry-run / analyze** before a live maintenance window | `kzero analyze`, `run.mode: dry-run` |
| **Bastion / cron** execution with audit logs and optional notify on API loss | [deployment-models.md](deployment-models.md), **0.8.x** watchdog + **`pipeline.stalled`** |
| Operator-owned hooks and scripts at phase or step boundaries | `hooks.*`, per-step `pre` / `post`, `custom:` |

**kzero is not:** continuous GitOps, cluster provisioning, node drain/cordon, cloud control planes, multi-cluster IDP control planes, or backup/DR as a product.

---

## Close cousin: [k0rdent](https://k0rdent.io/)

If you have seen **kzero** and **k0rdent** side by side, the naming is not accidental — both live in the “k0*” operator space — but the **primary job** is different.

| | **k0rdent** | **kzero** |
|---|-------------|-----------|
| **What it is** | Kubernetes-native **DCME** (“super control plane”) from the [Mirantis / k0s](https://k0rdent.io/) ecosystem | **Bastion CLI** for ordered maintenance on workloads **already** on a cluster |
| **Primary use case** | **Design, provision, and continuously manage** IDPs: clusters (Cluster API) + platform services (Helm/Kustomize templates) at **multi-cluster** scale | **Bounded** `down` / `up` / `reset` playbooks during maintenance windows or lab recovery |
| **Where it runs** | On a **management Kubernetes cluster** (controllers reconcile CRDs) | **Out-of-band** on bastion / cron host ([deployment-models.md](deployment-models.md)) |
| **How work is expressed** | Immutable `ClusterTemplate` / `ServiceTemplate` / chains; GitOps-style desired state | Operator YAML with explicit **step order**, hooks, `analyze`, dry-run |
| **Lifecycle model** | **Continuous reconciliation**, self-healing, template-driven upgrades | **Imperative pipeline** with fail-fast; not a day-to-day sync controller |
| **Infra scope** | Can **create clusters** (AWS, Azure, vSphere, …) and **install the platform stack into** managed clusters | **Never** provisions clusters; orchestrates scale, Helm, PVC, exec on an existing kubeconfig target |

**Rule of thumb:** k0rdent answers “how do we **stand up and keep** a platform / IDP (clusters + services) from a control cluster?” kzero answers “how do we **safely tear down and bring back** this known workload stack from a bastion during maintenance?”

They can coexist: k0rdent (or GitOps) may own steady-state platform composition; kzero runs a **reviewed reset playbook** when you need ordered teardown, data wipe, and phased bring-up outside continuous reconciliation.

Docs: [k0rdent documentation](https://docs.k0rdent.io/latest/).

---

## Lighter overlap: [szero](https://github.com/jadolg/szero)

[szero](https://github.com/jadolg/szero) shares the **`down` / `up`** verbs and namespace-scale mental model, but only **scales Deployments/StatefulSets/DaemonSets to zero and restores prior replica counts**. It does not install platform infra, Helm releases, PVC lifecycle, hooks, or a full **`reset`** contract. Useful for quick namespace hibernation; not a platform maintenance engine.

---

## Comparison matrix (by primary use case)

| Approach | Primary use case | Overlap with kzero | Typical gap vs kzero |
|----------|------------------|--------------------|----------------------|
| **kzero** | Ordered **maintenance / reset** pipelines on an **existing** cluster | — | Does not reconcile desired state continuously; does not create clusters or IDPs |
| **[k0rdent](https://k0rdent.io/)** | **Provision and manage** clusters + platform services (templates, multi-cluster DCME) | Declarative, Helm/K8s-native | Continuous control plane, not bastion cron **`reset`**; different failure model for destructive maintenance |
| **`helm` CLI** | **Per-chart** install, upgrade, uninstall | `release.*` steps (Helm SDK or wrapper scripts) | No cross-release **down→up** contract, phase hooks, or platform-wide ordering in one config |
| **`kubectl` + shell** | Ad hoc or bespoke bash/Make playbooks | Scale, delete PVC, exec — same underlying APIs | No shared schema, `analyze`, deferred-feature honesty, or engine-level fail-fast/retry/watchdog |
| **Argo CD / Flux (GitOps)** | **Continuous** sync of declared manifests to cluster state | Both are declarative | GitOps reconciles **desired state**, not a one-shot **maintenance reset** with ordered teardown; not bastion cron playbooks |
| **Ansible / general automation** | Multi-host and multi-system configuration | Can invoke `kubectl` / `helm` in tasks | No kzero YAML contract, native K8s executor, or maintenance-specific steps (`infra_probe`, reset phase preflight) |
| **Terraform / Crossplane / Cluster API** | **Provision** clusters, cloud infra, or platform CRDs | Indirect (cluster must exist) | Out of scope for **in-cluster workload** maintenance after the cluster is live |
| **Backup tools (e.g. Velero)** | **Backup and restore** for DR | May accompany a reset strategy | DR timeline and semantics differ from ordered scale-down + data wipe + Helm lifecycle |
| **Cost / scale-down utilities** (e.g. kube-downscaler patterns) | **Scale to zero** on schedule or label | `deployment` / `statefulset` down steps | No **`up`** pipeline, Helm releases, hooks, or full **reset** contract |
| **In-cluster Job + scripts** | Run automation **inside** the cluster | Can call the same APIs | Shared fate with the cluster under maintenance; kzero recommends **out-of-band** for destructive **`reset`** ([deployment-models.md](deployment-models.md)) |

---

## Common patterns (not either/or)

| Pattern | Roles |
|---------|--------|
| **GitOps day-to-day + kzero maintenance window** | Flux/Argo keep normal desired state; kzero runs a **bounded** down/up or reset during upgrades or lab recovery, then GitOps resumes |
| **Helm per app + kzero orchestration** | Helm (or kzero `release.*`) still performs each release; kzero defines **order** and **phases** across many releases and workloads |
| **Terraform / k0rdent cluster + kzero workloads** | Provisioning tool brings cluster (and optionally platform services); kzero operates **after** kubeconfig works |
| **Velero + kzero** | Velero for backup/restore policy; kzero for **controlled** teardown and bring-up sequences you want scripted and reviewed in YAML |

---

## What kzero deliberately does not compete on

Aligned with [SPECIFICATIONS.md §2 Out of scope](../SPECIFICATIONS.md):

- Node **drain**, **cordon**, removal
- **Cloud CLIs** (`az`, `aws`, `gcloud`) and OS-level cluster rebuild (Talos/k3s reset flows)
- **Automatic node pool scaling**
- Replacing a **GitOps** controller or **k0rdent-style** multi-cluster control plane as the source of truth for everyday deployments

If your main problem is “compose and continuously manage IDPs / clusters at scale,” look at **[k0rdent](https://k0rdent.io/)** (or GitOps for app sync). If your main problem is “run a known maintenance reset safely from a bastion on Tuesday at 02:00,” that is kzero’s lane.

---

## Related docs

| Doc | Topic |
|-----|--------|
| [deployment-models.md](deployment-models.md) | Bastion-first vs in-cluster |
| [examples/pipeline-network-loss.md](examples/pipeline-network-loss.md) | Long live reset, API/bastion network loss |
| [kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted) | Full platform reset example profiles |

**Last reviewed:** 2026-06-30
