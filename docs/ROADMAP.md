# kzero roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, schema, and what the engine does **today**
- **[CHANGELOG.md](../CHANGELOG.md)** — what shipped in each release

When a roadmap item ships, update **CHANGELOG** and tick or remove the item here (or move it to a “Completed” subsection with the release tag).

**Last reviewed:** 2026-06-03

### Versioning note

The first **public** releases are **0.2.0** onward (there was no prior `1.0.x` line). Section headings below (**0.3.x**, **0.4.x**, …) are **planned semver bands** for grouping work—not labels for releases that already existed under other numbers.

### Strategic direction

The v1 engine today is a **shell-backed** orchestrator (`kubectl` subprocesses, `/bin/sh` for hooks, releases, and custom steps). That was the right MVP.

The **next strategic priority** is a **native Go path** via **`k8s.io/client-go`** (and related modules): typed errors, testability without a live cluster, server-side dry-run, and room for richer pipeline steps (patch, suspend, delete) without growing the shell surface.

**Helm** stays on **workspace scripts** in the near term; a **Helm SDK** executor is optional later. **`release.*`**, **`custom:`**, and phase hooks may keep using shell even after workloads move to client-go.

---

## Shipped

| Release | Highlights |
|---------|------------|
| **0.2.0** | Initial published release: packaging, CI, core CLI/engine, declarative YAML pipelines. |
| **0.2.1** | Parse-time allow-list for compact step kinds; **DaemonSet** removed from built-in scalable workloads (documented `custom:` workaround); see [supported workload kinds](SPECIFICATIONS.md#supported-workload-kinds). |
| **0.2.2** | **`docs/ROADMAP.md`** published and linked; roadmap milestone bands aligned with **0.2.x** semver; **CHANGELOG** structure repaired so **[0.2.1]** release notes are under the correct heading again. CLI **deferred-feature warnings** on `analyze` / `down` / `up` / `reset` (stderr). |
| **0.2.3** | **Richer `kzero analyze`**: normalized **`[down]`** / **`[up]`** plans on stdout, **Deferred** summary, phase hooks and step metadata; SPEC/README contract. Completes **0.3.x** operator-honesty band. |
| **0.4.0** | **`run.execution`** (`shell` / `native` / `auto`) and **`internal/executor`**: client-go scale + rollout wait for `deployment` / `statefulset`; fake-clientset tests. Completes core **0.4.x** native-client band (items 5–9). |
| **0.4.1** | **`analyze` cluster validation**: API **Get** checks for pipeline `deployment` / `statefulset` refs when kubeconfig loads (roadmap **0.4.x** #10). |

---

## 0.3.x — operator honesty (complete in 0.2.3)

Close the gap between **schema** and **engine** before larger execution changes.

| # | Item | Status |
|---|------|--------|
| 1 | **CLI warnings** for config the engine does not honor: `run.worker_concurrency > 1`, `retry.attempts > 1`, `notify.{slack,discord}.enabled`. | **Done** (0.2.2) |
| 2 | **Explicit allow-list** for compact pipeline step kinds at parse time. | **Done** (0.2.1) |
| 3 | **Richer `analyze` output**: list normalized steps and summarize deferred schema fields. | **Done** (0.2.3) |
| 4 | **DaemonSet**: not a built-in scalable kind; document `custom:` workaround. | **Done** (0.2.1) |

---

## 0.4.x — native Kubernetes client (priority)

Introduce an **`Executor`** abstraction and implement workload steps against the API instead of fork/exec `kubectl` where practical.

| # | Item | Status |
|---|------|--------|
| 5 | **`internal/executor` contract**: `Shell` (kubectl) and `Native` (client-go). Config: `run.execution: shell \| native \| auto` (default `shell`). Document in SPEC. | **Done** (0.4.0) |
| 6 | **Native scale** for `deployment` and `statefulset` (API update; same down→0 / up→N semantics). | **Done** (0.4.0) |
| 7 | **Native rollout wait** for `wait_for_ready` on up (poll deployment/statefulset status). | **Done** (0.4.0) |
| 8 | **Typed API errors** (`NotFound`, `Forbidden`, conflict) via `errors.Is` on wrapped sentinels. | **Done** (0.4.0) |
| 9 | **Tests without a cluster**: fake clientset for scale + wait. | **Done** (0.4.0) |
| 10 | **`analyze` + API (optional)**: when kubeconfig is reachable, validate that referenced workloads exist and support scale (fail in analyze, not only in live). | **Done** (0.4.1) |
| 11 | **Stronger dry-run on native path**: server-side dry-run (`DryRun: All`) for scale/patch operations where applicable. | Pending |

**Out of scope for 0.4.x:** replacing `release.*` shell scripts with Helm SDK; node drain/cordon; PVC wipe primitives (see **0.7.x** / **1.0.0**).

---

## 0.5.x — execution engine (retries and throughput)

Applies to **both** executors where relevant; subprocess classification still matters for hooks and scripts.

| # | Item | Status |
|---|------|--------|
| 12 | **Retry** with exponential backoff for transient failures, wired to `cfg.Retry`. | Pending |
| 13 | **Concurrency** via bounded worker pool from `run.worker_concurrency`, preserving strict YAML order unless a future opt-in per-step parallelism is defined. | Pending |
| 14 | **Propagate `client.id`** into structured logs and hook environment (e.g. `KZERO_CLIENT_ID`). | Pending |
| 15 | **Subprocess error taxonomy** for shell path (exit codes, common stderr patterns) when native path is not used. | Pending |

---

## 0.6.x — observability and notifications

| # | Item | Status |
|---|------|--------|
| 16 | **`log/slog`** with `--log-format json|text`. | Pending |
| 17 | **Secret redaction** in logs and optional `--no-env-passthrough` for hooks. | Pending |
| 18 | **Implement `notify`** (Slack and Discord webhooks as promised by schema). | Pending |
| 19 | **`verify` mode** after `up`: structured readiness report (e.g. JSON). | Pending |

---

## 0.7.x — supply chain and extended pipeline steps

Broader **flush** operations beyond scale-to-zero, building on the native client from **0.4.x**.

| # | Item | Status |
|---|------|--------|
| 20 | **Cosign signing** and **SBOM** (e.g. Syft) in the GoReleaser pipeline. | Pending |
| 21 | **Raise coverage target** (e.g. 85%+) and fold **`make cover-check`** into **`make release-check`** when sustainable. | Pending |
| 22 | **Additional step types**: `job`, `cronjob` (suspend), safe generic **patch** / scale patterns for CRDs. Prefer native executor; shell fallback where needed. | Pending |
| 23 | **`custom:` parity**: pass `KZERO_PHASE` and step metadata to the main custom script (same as per-step hooks / release scripts). | Pending |
| 24 | **Release script ergonomics**: optional non-flat paths under `helm.workspace` (e.g. `monitoring/kube-prometheus-stack.sh`) without breaking flat `name.sh` convention. | Pending |

---

## 1.0.0 (future) — stable contract and platform depth

Major when YAML **`schema_version`**, executor behavior, and step types are stable enough for long-term compatibility promises.

| # | Item | Status |
|---|------|--------|
| 25 | **Helm SDK executor** (optional): `helm upgrade` / uninstall without maintainer `.sh` wrappers, behind explicit config. | Pending |
| 26 | **Default native execution** for workload steps when `run.execution` is omitted (shell opt-in). | Pending |
| 27 | **PVC / StatefulSet data strategy** documented as pipeline patterns (snapshot, wipe, init-job) — not necessarily core primitives on day one. | Pending |
| 28 | **Integration tests** with **kind** or envtest in CI, with documented flake policy and runtime budget. | Pending |

---

## Maintenance notes

- **GoReleaser**: address `nfpms` deprecation warnings (`maintainer`, `builds` → `ids`) on the next housekeeping release.
- **client-go version**: pin `k8s.io/*` modules to a supported Kubernetes minor; document minimum cluster version in README.
- **Integration tests**: **0.4.x** adds fake-client coverage; **1.0.0** targets optional **kind** / envtest in CI once flake policy is agreed (see SPEC testing baseline).
