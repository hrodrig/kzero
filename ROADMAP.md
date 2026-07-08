# kzero roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, schema, and what the engine does **today**
- **[CHANGELOG.md](CHANGELOG.md)** — what shipped in each release

When a roadmap item ships, update **CHANGELOG** and tick or remove the item here (or move it to a “Completed” subsection with the release tag).

**Last reviewed:** 2026-07-08 (**v0.9.0** on `develop`; **0.9.x** **#43–#47** shipped; **#48** partial; **#49–#53** → **0.9.1**; **1.0.0 #42** exit codes)

### Versioning note

The first **public** releases are **0.2.0** onward (there was no prior `1.0.x` line). Section headings below (**0.3.x**, **0.4.x**, …) are **planned semver bands** for grouping work—not labels for releases that already existed under other numbers.

### Strategic direction

The v1 engine runs **`deployment` / `statefulset`** steps via **`run.execution`**: `shell` (default), **`native`** (client-go scale + rollout wait), or **`auto`** (native with shell fallback).

**Deployment model (operator posture):** kzero **orchestrates** the cluster from **outside** it. The **recommended** production path is **out-of-band** — bastion, management VM, or cron on a host with **kubeconfig** — especially for destructive **`down`**, **`reset`**, and recovery when API or network reliability is uncertain. **In-cluster Job/CronJob** is **supported** (empty **`run.kubeconfig`**, distroless image, **`run.execution: native`**) as **optional packaging** for non-destructive or CI/smoke work, **not** as the target architecture for platform resets. Rationale and tables: **[docs/deployment-models.md](docs/deployment-models.md)**.

**Executor (single binary):** the **native** path covers **`deployment` / `statefulset`** scale and rollout wait, **`release.*`** via **Helm SDK**, **`pvc` delete**, and **`exec` in pod** — on a **bastion** this avoids host **`kubectl`** / **`helm`** while staying out-of-band. Set **`run.execution: native`** (or **`auto`**) for that path. Phase hooks and **`custom:`** shell scripts remain valid on bastions with **`/bin/sh`**; in-cluster Jobs should prefer declarative **`pvc`**, **`exec`**, and SDK **`release.*`** over host-only scripts when used at all.

**Remaining gap before 1.0.0:** default **`run.execution: native`** when omitted (**#32**), documented PVC/data-reset patterns (**#33**), and product-repo **kind**/envtest CI (**#34**). **`release.*`** on the **shell** path still requires **`<helm.workspace>/<name>.sh`** and external **`helm`** on **`PATH`**.

**Log capture** before or after pipelines is **out of scope** for the engine—invoke external tools via phase hooks when operators need archives. **Local stdout/stderr** (and wrapper tee to disk on the **management host**) is the audit trail when notify and API are both unavailable; see [docs/examples/pipeline-network-loss.md](docs/examples/pipeline-network-loss.md).

**Completed bands:** **0.3.x** (operator honesty), **0.4.x** (native client + analyze validation + server-side dry-run on native). **0.5.x** retry, **`client.id`**, live audit logs, and sequential-only contract shipped through **v0.5.6**. **0.6.x** notify, slog, verify, infra probe, preflight, OS audit, Helm workspace SPEC through **v0.6.0**. **0.7.x** Helm SDK, PVC/exec/schedulable primitives through **v0.7.2** (secret redaction in **v0.7.1**, text log levels in **v0.7.3**, sample-config in **v0.7.4**). **0.8.x** API watchdog, notify delivery visibility, reset phase-boundary preflight, progress logs, stalled event through **v0.8.0**. **0.9.0** bastion-first hardening: graceful shutdown, **`require_delivery`**, E2E smoke CI, watchdog tests, SPEC contract index (**#43–#47**).

**Current focus (planned work):**

| Band | Open items |
|------|------------|
| **0.5.x** | **Closed** (last item **#15** in **v0.5.7**) |
| **0.6.x** | **Closed** in **v0.6.0** |
| **0.7.x** | **Closed** — **#23–#31** in **v0.7.2**; **#29** `job`/`cronjob` still open. **v0.7.3**: text log levels. **v0.7.4**: `--print-sample-config`. |
| **0.8.x** | **Closed** — API watchdog, notify delivery, stalled event (**#35–#41**) in **v0.8.0**. |
| **0.9.x** | **v0.9.0 shipped** (**#43–#47**); **#48** docs partial; stretch **#49–#53** in **0.9.1** — see [docs/plan-0.9.x.md](docs/plan-0.9.x.md). |
| **1.0.0** | default **native** when `run.execution` omitted, PVC/data patterns doc, **kind**/envtest CI (**#32–#34**); **#29** optional in band or pre-1.0; **#42** exit codes. |

**Shell path:** **`run.execution: shell`** (default) still uses **`kubectl`** subprocesses and **`<helm.workspace>/<name>.sh`** for **`release.*` up**. **Native/auto** uses the Helm SDK and API primitives above.

---

## Shipped

| Release | Highlights |
|---------|------------|
| **0.2.0** | Initial published release: packaging, CI, core CLI/engine, declarative YAML pipelines. |
| **0.2.1** | Parse-time allow-list for compact step kinds; **DaemonSet** removed from built-in scalable workloads (documented `custom:` workaround); see [supported workload kinds](SPECIFICATIONS.md#supported-workload-kinds). |
| **0.2.2** | **`docs/ROADMAP.md`** published and linked; roadmap milestone bands aligned with **0.2.x** semver; **CHANGELOG** structure repaired so **[0.2.1]** release notes are under the correct heading again. CLI **deferred-feature warnings** on `analyze` / `down` / `up` / `reset` (stderr). |
| **0.2.3** | **Richer `kzero analyze`**: normalized **`[down]`** / **`[up]`** plans on stdout, **Deferred** summary, phase hooks and step metadata; SPEC/README contract. Completes **0.3.x** operator-honesty band. |
| **0.4.0** | **`run.execution`** (`shell` / `native` / `auto`) and **`internal/executor`**: client-go scale + rollout wait for `deployment` / `statefulset`; fake-clientset tests (roadmap **0.4.x** #5–9). |
| **0.4.1** | **`analyze` cluster validation**: API **Get** checks for pipeline `deployment` / `statefulset` refs when kubeconfig loads (roadmap **0.4.x** #10). **0.4.x** band closed after **0.5.1** (#11). |
| **0.5.0** | **Operator safety for pilots**: **`Kubernetes target:`** on pipeline commands, **`kzero target`**, **`KZERO_*` env overrides** on load, **elapsed time** summary; suitable for scripted **`release.*`** down/up with external Helm wrappers. |
| **0.5.1** | **`run.color`** for timing-line ANSI styling; **server-side dry-run** on native/auto scale steps (roadmap **0.4.x** #11). |
| **0.5.2** | **Per-step retry** with exponential backoff on transient errors (roadmap **0.5.x** #12). |
| **0.5.3** | **`run.worker_concurrency` removed** from contract; pipeline execution **strictly sequential** (roadmap **0.5.x** #13 closed). |
| **0.5.5** | **`[live]`** action logs, **`started_at`** / **`client_id`** in **`Kubernetes target:`**, release hook env fix, pipeline wait docs and reference hooks. |
| **0.5.6** | Pipeline command factorization, coverage housekeeping, BSD port sync. |
| **0.5.7** | **0.5.x band closed:** subprocess error taxonomy (`WrapSubprocess`, `ErrTransient`) for kubectl/helm/hooks on the shell path. |
| **0.6.0** | **0.6.x band closed:** multi-channel **`notify`**, **`--log-format`**, **`kzero verify`**, **`kzero probe`** / **`infra_probe`**, live **preflight**, OS audit fields, Helm workspace SPEC. |
| **0.7.0** | **Cosign** keyless signing and **SPDX/CycloneDX SBOM** in GoReleaser (roadmap **0.7.x #28**). |
| **0.7.1** | **In-cluster auth** (empty **`run.kubeconfig`** → service account token); **#17** secret redaction and **`--no-env-passthrough`**. |
| **0.7.2** | **0.7.x band close:** Helm SDK (**#25**), **`pvc`** / **`exec`**, probe native (**#26**), **`pods_schedulable`** (**#27**), OCI **`helm.registries`**, **`script:`** paths (**#31**), **`custom:`** env parity (**#30**). |
| **0.7.3** | **Text log levels** (`--log-level`, timestamped `[DBG|INF|WRN|ERR]` lines); **Slack notify** rich attachments + **`KZERO_NOTIFY_*`** env; Helm SDK **OCI login** hardening (private registry pilot). |
| **0.7.4** | **`kzero --print-sample-config`** (stdout sample YAML for Homebrew / binary-only installs); **0.8.x** planning docs; Docker build includes **`configs/`**. |
| **0.8.0** | **0.8.x band close:** API watchdog (`run.api_watchdog`) with throttle and cumulative trip; `[ERR]` log on notify dispatch failure (#35); `notify.require_delivery` schema (#39); reset phase-boundary preflight (#37); throttled progress logs on long waits (#38); `pipeline.stalled` event + `notify test --event stalled` (#41). |
| **0.9.0** | **0.9.x core:** graceful shutdown (#44); **`notify.require_delivery`** engine (#43); **`target --output slug`**; E2E smoke CI (#45); watchdog mid-wait tests (#46); SPEC contract index (#47); API watchdog `/healthz` HTTP probe fix; [deployment-models.md](docs/deployment-models.md). |

---

## 0.3.x — operator honesty (complete in 0.2.3)

Close the gap between **schema** and **engine** before larger execution changes.

| # | Item | Status |
|---|------|--------|
| 1 | **CLI warnings** for config the engine does not honor (formerly `worker_concurrency`; removed from contract in 0.5.3). **`notify.*`** channels are implemented since **0.6.0**; deferred warnings apply only to schema keys still without engine support. **`retry`** is implemented since **0.5.2**. | **Done** (0.2.2; notify warnings removed 0.6.0) |
| 2 | **Explicit allow-list** for compact pipeline step kinds at parse time. | **Done** (0.2.1) |
| 3 | **Richer `analyze` output**: list normalized steps and summarize deferred schema fields. | **Done** (0.2.3) |
| 4 | **DaemonSet**: not a built-in scalable kind; document `custom:` workaround. | **Done** (0.2.1) |

---

## 0.4.x — native Kubernetes client (complete)

Band **closed** (items **#5–#11**). Last deliverable: server-side dry-run on the native path (**v0.5.1**, item #11).

Introduced an **`Executor`** abstraction and workload steps against the API instead of fork/exec `kubectl` where practical.

| # | Item | Status |
|---|------|--------|
| 5 | **`internal/executor` contract**: `Shell` (kubectl) and `Native` (client-go). Config: `run.execution: shell \| native \| auto` (default `shell`). Document in SPEC. | **Done** (0.4.0) |
| 6 | **Native scale** for `deployment` and `statefulset` (API update; same down→0 / up→N semantics). | **Done** (0.4.0) |
| 7 | **Native rollout wait** for `wait_for_ready` on up (poll deployment/statefulset status). | **Done** (0.4.0) |
| 8 | **Typed API errors** (`NotFound`, `Forbidden`, conflict) via `errors.Is` on wrapped sentinels. | **Done** (0.4.0) |
| 9 | **Tests without a cluster**: fake clientset for scale + wait. | **Done** (0.4.0) |
| 10 | **`analyze` + API (optional)**: when kubeconfig is reachable, validate that referenced workloads exist and support scale (fail in analyze, not only in live). | **Done** (0.4.1) |
| 11 | **Stronger dry-run on native path**: server-side dry-run (`DryRun: All`) for scale/patch operations where applicable. | **Done** (0.5.1) |

**Out of scope for 0.4.x:** replacing `release.*` shell scripts with Helm SDK; node drain/cordon; PVC wipe primitives (see **0.7.x** / **1.0.0**).

---

## 0.5.x — execution engine (retries; sequential contract)

Band **closed** in **v0.5.7** (item **#15**). Applies to kubectl/helm/hook subprocesses on the shell path; native API errors continue to use **`WrapAPIError`**.

| # | Item | Status |
|---|------|--------|
| 12 | **Retry** with exponential backoff for transient failures, wired to `cfg.Retry`. | **Done** (0.5.2) |
| 13 | **Pipeline parallelism** (`run.worker_concurrency`, worker pools, parallel waves). | **Removed from contract** (0.5.3) — engine stays **strictly sequential**; use step order and `custom:` for operator-controlled batching. |
| 14 | **Propagate `client.id`** into structured logs and hook environment (e.g. `KZERO_CLIENT_ID`). | **Done** (0.5.4) |
| 15 | **Subprocess error taxonomy** for shell path (exit codes, common stderr patterns) when native path is not used. | **Done** (0.5.7) |

---

## 0.6.x — observability, notifications, and preflight

**Implementation plan:** [docs/plan-0.6.0.md](docs/plan-0.6.0.md) (target **`v0.6.0`**). Merge order: **PR1** audit → **notify** → **slog** → **verify** → **infra probe** → **preflight** + SPEC + tag.

| # | Item | Status |
|---|------|--------|
| 16 | **`log/slog`** with `--log-format json|text`. | Done (PR3, develop) |
| 17 | **Secret redaction** in logs and optional `--no-env-passthrough` for hooks. | **Done** (**v0.7.1**) |
| 18 | **`notify`**: implement common outbound channels—**Slack**, **Microsoft Teams**, **PagerDuty**, and a **generic webhook** (plus **Discord** already in schema). Fire on pipeline start/end and optionally on error; redact secrets in payloads. **`kzero notify test`** verifies channels without a pipeline. | Done (PR2, develop) |
| 19 | **Preflight connectivity**: before mutating resources, verify API reachability (e.g. list nodes or equivalent) and fail fast with a clear message. | **Done** (develop, 0.6.x PR6) |
| 20 | **Operator audit**: include **OS username** and **UID** in the **`Kubernetes target:`** block and expose **`KZERO_OS_USER`** / **`KZERO_OS_UID`** (or equivalent) to hooks and subprocesses. Complements **`client.id`**. | Done (PR1, develop) |
| 21 | **`verify` mode** after `up`: structured readiness report (e.g. JSON). | Done (PR4, develop) |
| 22 | **Infra probe** before destructive **`down`** / **`reset`**: optional **`infra_probe`** config and **`kzero probe`** runs a **declarative mini-pipeline** (operator-maintained dummy **`release.*`** + PVC) to confirm storage/Helm can provision before wiping real PVCs and core releases. Fail-fast; optional result cache TTL. | **Done** (develop, 0.6.x PR5) |
| 22bis | **Helm workspace contract in SPEC**: document flat **`<helm.workspace>/<release>.sh`** naming, env vars, and analyze/live resolution **before** **0.7.x #25** (Helm SDK) extends paths/OCI. | **Done** (develop, 0.6.x PR6) |

---

## 0.7.x — native cluster operations and Helm SDK

**Implementation plan:** [docs/plan-0.7.x.md](docs/plan-0.7.x.md) (**band closed** at **`v0.7.2`**; **`v0.7.3`** patch documented there).

Broader pipeline primitives via **client-go** and **helm.sh/helm/v3**, keeping a **single distroless image** without fork/exec to external `kubectl` / `helm` for built-in step types when **`run.execution: native`** or **`auto`**.

| # | Item | Status |
|---|------|--------|
| 23 | **`exec` step type**: run a command (and optional stdin) inside a named pod/container via **remotecommand**—covers SQL, admin CLIs, and other in-cluster maintenance without a one-off truncate primitive. | **Done** (develop, PR5) |
| 24 | **`pvc` step type**: delete named PVCs (or labeled sets) via the API for data-reset pipelines. | **Done** (develop, PR4) |
| 25 | **Helm SDK executor** for **`release.*`**: `upgrade --install` / uninstall with wait; **OCI registry login** from **`helm.registries`** (no separate operator image). | **Done** (**0.7.2**) |
| 26 | **Infra probe (native checks)**: PVC **Bound** wait, optional in-volume write/read, and probe teardown using built-in step types (extends **0.6.x #22** for distroless/single-binary runs). | **Done** (develop, PR6 — native Redis sample + docs) |
| 27 | **Scheduling / affinity sanity** (optional probe or **`verify`** check): detect pods **Pending** due to node selectors, affinity, or taints after maintenance—separate from storage probe. | **Done** (**0.7.2** — **`pods_schedulable`**) |
| 28 | **Cosign signing** and **SBOM** (e.g. Syft) in the GoReleaser pipeline. | **Done** (**v0.7.0**) |
| 29 | **Additional step types**: `job`, `cronjob` (suspend), safe generic **patch** / scale patterns for CRDs. Prefer native executor; shell fallback where needed. | Pending |
| 30 | **`custom:` parity**: pass `KZERO_PHASE` and step metadata to the main custom script (same as per-step hooks / release scripts). | **Done** (**0.7.2**) |
| 31 | **Release script ergonomics**: optional non-flat paths under `helm.workspace` (e.g. `monitoring/kube-prometheus-stack.sh`) without breaking flat `name.sh` convention. | **Done** (**0.7.2** — step **`script:`**) |

---

## 0.8.x — pipeline resilience (network loss and notify delivery)

**Implementation plan:** [docs/plan-0.8.x.md](docs/plan-0.8.x.md) (target **`v0.8.0`**).

Motivation: live **`reset`** on a bastion lost API connectivity mid-run; process detected unreachable control plane after ~15 minutes but **did not alert**; ~30 minutes later **total network loss**; recovery hours later with **logs as sole evidence**.

| # | Item | Status |
|---|------|--------|
| 35 | **Notify delivery visibility**: log **`[ERR]`** when outbound notify POST fails (redacted); optional **`notify.require_delivery`** to fail pipeline if error notify cannot be sent. | Done (v0.8.0, #35) |
| 36 | **API watchdog** during live **`down`/`up`/`reset`**: periodic API reachability check between steps and during long waits; fail-fast + **`pipeline.error`** when threshold exceeded. | Done (v0.8.0, #36) |
| 37 | **Preflight between `reset` phases**: re-check API after **`down`**, before **`up`**. | Done (v0.8.0, #37) |
| 38 | **Throttled progress logs** on long rollout/Helm waits (step ref, elapsed, last API OK). | Done (v0.8.0, #38) |
| 39 | **Config `run.api_watchdog`** (+ **`KZERO_RUN_API_WATCHDOG_*`** env binding). | Done (v0.8.0, #39) |
| 40 | **Operator docs**: [docs/examples/pipeline-network-loss.md](docs/examples/pipeline-network-loss.md); selfhosted automation cross-link. | Done (v0.8.0, #40) |
| 41 | *(Optional)* **`pipeline.stalled`** notify event + **`kzero notify test --event stalled`**. | Done (v0.8.0, #41) |

**Operator mitigations before 0.8 ships:** short **`run.operation_timeout`**, **`on-error`** hooks, external watchdog, wrapper log files on the **bastion** — documented in [pipeline-network-loss.md](docs/examples/pipeline-network-loss.md).

---

## 0.9.x — bastion-first hardening (v0.9.0 shipped; stretch in 0.9.1)

**Implementation plan:** [docs/plan-0.9.x.md](docs/plan-0.9.x.md).

Motivation: close **0.8.x** deferred contract gaps and operator posture after external audits — **out-of-band** control as the default story ([deployment-models.md](docs/deployment-models.md)), not in-cluster **`reset`**.

| # | Item | Status |
|---|------|--------|
| 43 | **`notify.require_delivery`** — engine fail-fast when error-notify POST fails (finish **#35** deferred) | Done (develop) |
| 44 | **Graceful shutdown** — SIGTERM/SIGINT cancel pipeline context; log last step (bastion/cron) | Done (develop) |
| 45 | **E2E smoke in CI** — kind or **kzero-selfhosted** minimal pipeline (not in-cluster production reset) | Done (v0.9.0) |
| 46 | **Watchdog tests** — API unreachable mid-wait scenarios | Done (v0.9.0) |
| 47 | **SPEC: contract vs experimental/deferred** — single operator-facing index | Done (v0.9.0) |
| 48 | **Docs** — [deployment-models.md](docs/deployment-models.md) kickoff **done**; [scope-and-alternatives.md](docs/scope-and-alternatives.md) linked; What's new **0.9.x** in CHANGELOG/README; **`cosign verify`** in README (v0.7.0+); README trim deferred to **0.9.1** | Partial (**v0.9.0**) |
| 49 | *(Stretch / 0.9.1)* **`kzero validate --strict`** / **`doctor`** — config + connectivity + RBAC hints | Pending |
| 50 | *(Stretch)* **Retry jitter** on existing backoff | Pending |
| 51 | *(Stretch)* **JSON Schema** for editor autocomplete | Pending |
| 52 | *(Stretch / 0.9.1)* **`kubectl-kzero` plugin** — `kubectl kzero …` via **`kubectl-kzero`** on PATH (same CLI as **`kzero`**; bastion DX; GoReleaser) | Pending |
| 53 | *(Stretch / 0.9.1)* **Shell completion** — `kzero completion <bash\|zsh\|fish\|powershell>` with strict validation, tests, README ([groot #80](https://github.com/hrodrig/groot/blob/main/pkg/cmd/completion.go) pattern); **`kubectl kzero completion`** when **#52** ships | Pending |

**Merge order (see plan):** PR1 docs → **#44** graceful shutdown → **#43** `require_delivery` → E2E/watchdog/SPEC → **v0.9.0** → **#53** / **#52** / **#49** in **0.9.1**.

**Non-goals for 0.9.x:** promoting in-cluster Job as the primary **`reset`** path; Prometheus/OTel; chaos-mesh; breaking in-cluster auth without migration note.

---

## 1.0.0 (future) — stable contract and platform depth

Major when YAML **`schema_version`**, executor behavior, and step types are stable enough for long-term compatibility promises.

| # | Item | Status |
|---|------|--------|
| 32 | **Default native execution** for workload steps when `run.execution` is omitted (shell opt-in). | Pending |
| 33 | **PVC / StatefulSet data strategy** documented as pipeline patterns (snapshot, wipe, init-job) beyond core delete primitives. | Pending |
| 34 | **Integration tests** with **kind** or envtest in CI, with documented flake policy and runtime budget. | Pending |
| 42 | **Documented exit code taxonomy** for CLI scripts/wrappers: today all non-zero returns collapse to `1` (`cmd/kzero/main.go:14`); map subsystem failures to stable codes (config, Kubernetes client/API, executor aborted, notify delivery, partial failures) following the same pattern adopted for [groot 0.9.x #82](https://github.com/hrodrig/groot/blob/main/pkg/cmd/exitcode.go). Implementation note: not breaking for existing wrappers — codes beyond config error are only emitted where the underlying failure category is unambiguous. | Pending |
| 55 | *(Optional)* **Post-pipeline log upload** — after a run, push **`run.log_file`** (or wrapper tee output) to S3/GCS/SFTP (env creds, `continue_on_error`, `--no-upload`); hooks/selfhosted patterns remain the default; not a groot-style archive bundle. | Pending |

---

## Maintenance notes

- **Release cadence:** closing semver bands (0.5.x → 0.6.x) justifies frequent tags; once adoption catches up, prefer fewer tags that each earn changelog, port-sync, and demo refresh—avoid maintaining versions nobody runs.
- **Coverage artifact:** `coverage.out` is deny-all gitignored; `make clean` removes it locally so clones do not accumulate stale artifacts (`git add -f` risk).
- **GoReleaser**: address `nfpms` deprecation warnings (`maintainer`, `builds` → `ids`) on the next housekeeping release.
- **client-go version**: pin `k8s.io/*` modules to a supported Kubernetes minor; document minimum cluster version in README.
- **Integration tests**: **0.4.x** delivered fake-client coverage (band complete). **1.0.0 #34** targets optional **kind** / envtest in CI once flake policy is agreed (see SPEC testing baseline).
