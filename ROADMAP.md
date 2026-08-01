# kzero roadmap

<a id="top"></a>

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, schema, and what the engine does **today**
- **[CHANGELOG.md](CHANGELOG.md)** — what shipped in each release

When a roadmap item ships, update **CHANGELOG** and tick or remove the item here (or move it to a “Completed” subsection with the release tag).

**Last reviewed:** 2026-08-01 (**1.1.0** queue: **#59** Helm v4 → **#29** → **#58**; **#57** deferred; **#55** parked; **#56** shipped in **v1.0.2**)

### Versioning note

The first **public** releases are **0.2.0** onward (there was no prior `1.0.x` line). Section headings below (**0.3.x**, **0.4.x**, …) are **planned semver bands** for grouping work—not labels for releases that already existed under other numbers.

### Strategic direction

The v1 engine runs **`deployment` / `statefulset`** steps via **`run.execution`**: **`native`** (default when omitted — client-go scale + rollout wait), **`shell`** (opt-in kubectl), or **`auto`** (native with shell fallback).

**Deployment model (operator posture):** kzero **orchestrates** the cluster from **outside** it. The **recommended** production path is **out-of-band** — bastion, management VM, or cron on a host with **kubeconfig** — especially for destructive **`down`**, **`reset`**, and recovery when API or network reliability is uncertain. **In-cluster Job/CronJob** is **supported** (empty **`run.kubeconfig`**, distroless image, **`run.execution: native`**) as **optional packaging** for non-destructive or CI/smoke work, **not** as the target architecture for platform resets. Rationale and tables: **[docs/deployment-models.md](docs/deployment-models.md)**.

**Executor (single binary):** the **native** path covers **`deployment` / `statefulset`** scale and rollout wait, **`release.*`** via **Helm SDK**, **`pvc` delete**, and **`exec` in pod** — on a **bastion** this avoids host **`kubectl`** / **`helm`** while staying out-of-band. Set **`run.execution: native`** (or **`auto`**) for that path. Phase hooks and **`custom:`** shell scripts remain valid on bastions with **`/bin/sh`**; in-cluster Jobs should prefer declarative **`pvc`**, **`exec`**, and SDK **`release.*`** over host-only scripts when used at all.

**1.0.0 (shipped):** PVC patterns (**#33**), exit codes (**#42**), kind CI (**#34**), default native (**#32**). **`release.*`** on the **shell** path still requires **`<helm.workspace>/<name>.sh`** and external **`helm`** on **`PATH`**.

**Log capture** before or after pipelines is **out of scope** for the engine—invoke external tools via phase hooks when operators need archives. **Local stdout/stderr** (and wrapper tee to disk on the **management host**) is the audit trail when notify and API are both unavailable; see [docs/examples/pipeline-network-loss.md](docs/examples/pipeline-network-loss.md).

**Completed bands:** **0.3.x** (operator honesty), **0.4.x** (native client + analyze validation + server-side dry-run on native). **0.5.x** retry, **`client.id`**, live audit logs, and sequential-only contract shipped through **v0.5.6**. **0.6.x** notify, slog, verify, infra probe, preflight, OS audit, Helm workspace SPEC through **v0.6.0**. **0.7.x** Helm SDK, PVC/exec/schedulable primitives through **v0.7.2** (secret redaction in **v0.7.1**, text log levels in **v0.7.3**, sample-config in **v0.7.4**). **0.8.x** API watchdog, notify delivery visibility, reset phase-boundary preflight, progress logs, stalled event through **v0.8.0**. **0.9.0** bastion-first hardening: graceful shutdown, **`require_delivery`**, E2E smoke CI, watchdog tests, SPEC contract index (**#43–#47**).

**Current focus (planned work):**

| Band | Open items |
|------|------------|
| **0.5.x** | **Closed** (last item **#15** in **v0.5.7**) |
| **0.6.x** | **Closed** in **v0.6.0** |
| **0.7.x** | **Closed** — **#23–#28**, **#30–#31** in **v0.7.2** (+ **v0.7.3**/**v0.7.4** patches). **#29** deferred **post-1.x**. |
| **0.8.x** | **Closed** — API watchdog, notify delivery, stalled event (**#35–#41**) in **v0.8.0**. |
| **0.9.x** | **Closed** stretch — **v0.9.0** core (**#43–#47**); **v0.9.1** security; **v0.9.2** doctor/completion/plugin/jitter/JSON Schema/docs (**#48–#53**). |
| **1.0.0** | **Closed** — default **native** (**#32**), PVC cookbook (**#33**), kind CI (**#34**), exit codes **0–4** (**#42**) in **v1.0.0**. |
| **1.1.0** | **#59** Helm SDK **v4** (done on develop) → **#29** job/cronjob/CRD → **#58** diff; **#57** deferred; **#55** parked — [plan-1.1.0.md](docs/plan-1.1.0.md). (**#56** shipped in **v1.0.2**.) |

**Shell path:** **`run.execution: shell`** (opt-in) still uses **`kubectl`** subprocesses and **`<helm.workspace>/<name>.sh`** for **`release.*` up**. **Native** (default) / **auto** use the Helm SDK and API primitives above.

---

[↑ Back to top](#top)

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
| **0.9.1** | **Security patch:** Go **1.26.5** (stdlib GO-2026-4970, GO-2026-5856); **`oras.land/oras-go/v2` v2.6.2** (CVE-2026-50163 / Dependabot #6); distroless **`static-debian13:nonroot`**; Grype ignore removed. |
| **0.9.2** | **0.9.x stretch:** shell completion (#53); **`kubectl-kzero`** (#52); **`kzero doctor`** (#49); retry full jitter (#50); JSON Schema (#51); docs Cosign/README (#48). |
| **1.0.0** | **Stable contract:** default **`run.execution: native`** (**#32**); exit codes **0–4** (**#42**); product kind CI (**#34**); PVC/StatefulSet cookbook (**#33**); README out-of-band hero; POSIX `/bin/sh` hook contract. |
| **1.0.1** | **Retry:** classify **`connection lost`** / **`http2: client connection lost`** as transient for live step retry and shell **`ErrTransient`**. |
| **1.0.2** | **`command.shell`** (#56) opt-in hook/script interpreter; pin **`golang.org/x/crypto` v0.54.0** + Grype ignore hygiene (GO-2026-5932 until Helm v4 #59); README badge/docs hygiene. |

---

[↑ Back to top](#top)

## 0.3.x — operator honesty (complete in 0.2.3)

Close the gap between **schema** and **engine** before larger execution changes.

| # | Item | Status |
|---|------|--------|
| 1 | **CLI warnings** for config the engine does not honor (formerly `worker_concurrency`; removed from contract in 0.5.3). **`notify.*`** channels are implemented since **0.6.0**; deferred warnings apply only to schema keys still without engine support. **`retry`** is implemented since **0.5.2**. | **Done** (0.2.2; notify warnings removed 0.6.0) |
| 2 | **Explicit allow-list** for compact pipeline step kinds at parse time. | **Done** (0.2.1) |
| 3 | **Richer `analyze` output**: list normalized steps and summarize deferred schema fields. | **Done** (0.2.3) |
| 4 | **DaemonSet**: not a built-in scalable kind; document `custom:` workaround. | **Done** (0.2.1) |

---

[↑ Back to top](#top)

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

[↑ Back to top](#top)

## 0.5.x — execution engine (retries; sequential contract)

Band **closed** in **v0.5.7** (item **#15**). Applies to kubectl/helm/hook subprocesses on the shell path; native API errors continue to use **`WrapAPIError`**.

| # | Item | Status |
|---|------|--------|
| 12 | **Retry** with exponential backoff for transient failures, wired to `cfg.Retry`. | **Done** (0.5.2) |
| 13 | **Pipeline parallelism** (`run.worker_concurrency`, worker pools, parallel waves). | **Removed from contract** (0.5.3) — engine stays **strictly sequential**; use step order and `custom:` for operator-controlled batching. |
| 14 | **Propagate `client.id`** into structured logs and hook environment (e.g. `KZERO_CLIENT_ID`). | **Done** (0.5.4) |
| 15 | **Subprocess error taxonomy** for shell path (exit codes, common stderr patterns) when native path is not used. | **Done** (0.5.7) |

---

[↑ Back to top](#top)

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

[↑ Back to top](#top)

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
| 29 | **Additional step types**: `job`, `cronjob` (suspend), safe generic **patch** / scale patterns for CRDs. Prefer native executor; shell fallback where needed. Until then use **`custom:`**. | Deferred → **1.1.0** ([plan-1.1.0.md](docs/plan-1.1.0.md)) |
| 30 | **`custom:` parity**: pass `KZERO_PHASE` and step metadata to the main custom script (same as per-step hooks / release scripts). | **Done** (**0.7.2**) |
| 31 | **Release script ergonomics**: optional non-flat paths under `helm.workspace` (e.g. `monitoring/kube-prometheus-stack.sh`) without breaking flat `name.sh` convention. | **Done** (**0.7.2** — step **`script:`**) |

---

[↑ Back to top](#top)

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

[↑ Back to top](#top)

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
| 48 | **Docs** — deployment-models + scope linked; Cosign verify examples; README trim; What's new / stretch notes | Done (**v0.9.2**) |
| 49 | *(Stretch / 0.9.1)* **`kzero doctor`** — config + connectivity + binaries + RBAC hints | Done (**v0.9.2**) |
| 50 | *(Stretch)* **Retry jitter** on existing backoff | Done (**v0.9.2**) |
| 51 | *(Stretch)* **JSON Schema** for editor autocomplete (`configs/kzero.schema.json`) | Done (**v0.9.2**) |
| 52 | *(Stretch / 0.9.1)* **`kubectl-kzero` plugin** — `kubectl kzero …` via **`kubectl-kzero`** on PATH (same CLI as **`kzero`**; bastion DX; GoReleaser) | Done (**v0.9.2**) |
| 53 | *(Stretch / 0.9.1)* **Shell completion** — `kzero completion <bash\|zsh\|fish\|powershell>` with strict validation, tests, README ([groot #80](https://github.com/hrodrig/groot/blob/main/pkg/cmd/completion.go) pattern); **`kubectl kzero completion`** when **#52** ships | Done (**v0.9.2**) |

**Merge order (historical):** PR1 docs kickoff → **#44** graceful shutdown → **#43** `require_delivery` → E2E/watchdog/SPEC → **v0.9.0** → stretch **#53** / **#52** / **#49** / **#50** / **#51** + **#48** close → **v0.9.2**.

**Non-goals for 0.9.x:** promoting in-cluster Job as the primary **`reset`** path; Prometheus/OTel; chaos-mesh; breaking in-cluster auth without migration note.

---

[↑ Back to top](#top)

## 1.0.0 — stable contract (shipped in v1.0.0)

**Implementation plan:** [docs/plan-1.0.0.md](docs/plan-1.0.0.md) (**Done**).

| # | Item | Status |
|---|------|--------|
| 32 | **Default native execution** for workload steps when `run.execution` is omitted (shell opt-in). | **Done** (v1.0.0) |
| 33 | **PVC / StatefulSet data strategy** documented as pipeline patterns (snapshot, wipe, init-job) beyond core delete primitives. | **Done** (v1.0.0, [pvc-statefulset-data-strategy.md](docs/examples/pvc-statefulset-data-strategy.md)) |
| 34 | **Integration tests** with **kind** or envtest in CI, with documented flake policy and runtime budget. | **Done** (v1.0.0, `testing/kind/` + job **`integration-kind`**) |
| 42 | **Documented exit code taxonomy** for CLI scripts/wrappers: stable codes **0–4** (`internal/exitcode`, groot-style `ExitError`); config / Kubernetes / executor / notify. Plain errors default to **1**. | **Done** (v1.0.0) |
| 55 | *(Optional)* **Post-pipeline log upload** — after a run, push **`run.log_file`** (or wrapper tee output) to S3/GCS/SFTP (env creds, `continue_on_error`, `--no-upload`); hooks/selfhosted patterns remain the default; not a groot-style archive bundle. | Deferred (optional / **1.1** stretch) |

---

[↑ Back to top](#top)

## 1.1.0 (future) — post-1.0 ergonomics (bounded)

**Implementation plan:** [docs/plan-1.1.0.md](docs/plan-1.1.0.md). Starts after **v1.0.0**.

| # | Item | Status |
|---|------|--------|
| 56 | **Configurable hook interpreter** — default **`/bin/sh`**; opt-in **`command.shell`** for hooks / **`custom:`** / shell release scripts (no magic shebang). | **Done** (**v1.0.2**) |
| 59 | **Helm SDK v4.2.3+** — drop **GO-2026-5932**; bump `k8s.io/*` with Helm; see plan spike. | **Done** (on develop; Unreleased → **1.1.0**) |
| 29 | **`job` / `cronjob`** (suspend) and safe generic **patch** / scale for CRDs — prefer native; until then **`custom:`**. | Pending (next) |
| 58 | **`kzero diff`** — live plan vs cluster. | Pending (after **#29**) |
| 57 | **Resume / restart from step** — Phase A: restart from index N; Phase B optional state file. | **Deferred** (complexity) |
| 55 | **Post-pipeline log upload** — wrappers/selfhosted remain default. | **Parked** |

**Parked (not 1.1):** parallel waves, webhook/schedule daemon, SSH bastion tunnel, multi-cluster YAML, approval gates, secret-manager plugins, OTel/Prometheus productization.

---

[↑ Back to top](#top)

## Maintenance notes

- **Release cadence:** closing semver bands (0.5.x → 0.6.x) justifies frequent tags; once adoption catches up, prefer fewer tags that each earn changelog, port-sync, and demo refresh—avoid maintaining versions nobody runs.
- **Coverage artifact:** `coverage.out` is deny-all gitignored; `make clean` removes it locally so clones do not accumulate stale artifacts (`git add -f` risk).
- **GoReleaser**: address `nfpms` deprecation warnings (`maintainer`, `builds` → `ids`) on the next housekeeping release.
- **client-go version**: pin `k8s.io/*` modules to a supported Kubernetes minor; document minimum cluster version in README.
- **Integration tests**: **0.4.x** fake-client coverage; **1.0.0 #34** product kind CI (`testing/kind/`, job **`integration-kind`**). Full lab remains in **kzero-selfhosted**.

[↑ Back to top](#top)
