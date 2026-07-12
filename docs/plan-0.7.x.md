# Plan 0.7.x — native cluster ops, Helm SDK, and operator safety

**Status:** **Band closed** — **`v0.7.2`** on **`main`** (core **0.7.x** deliverables). **`v0.7.3`** patch: text log levels, Slack notify attachments, **`KZERO_NOTIFY_*`**, Helm SDK OCI login hardening.

This document is **historical planning context** for the **0.7.x** semver band. For current shipped behavior, see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md). Open work: **1.0.0** (**#32–#34**, **#42**); **#29** (`job`/`cronjob`) deferred **post-1.x**.

### Internal merge order (2026-06-11) — completed

Close **0.6.x #17** first (small, operator-facing, unblocks safer logs before larger executors). Then ship **in-cluster** as a tagged patch, then **Helm SDK** (largest slice), then **`pvc` / `exec` / probe native** for develop2-style distroless pipelines.

1. **Secret redaction + env passthrough** — finish **#17** deferred from [plan-0.6.0.md](plan-0.6.0.md)  
2. **In-cluster auth** — release polish for merged `LoadRESTConfig` / target in-cluster (tag **0.7.1**)  
3. **Helm SDK** — **#25**; closes distroless gap for **`release.*`**  
4. **`pvc` delete** — **#24**; data-reset pipelines without host `kubectl`  
5. **`exec` step type** — **#23**; replaces many shell hooks in-cluster  
6. **Infra probe (native)** — **#26**; probe without shell scripts in the GHCR image  
7. **Scheduling / affinity sanity** — **#27**; optional verify/probe extension  
8. **Extended step types + ergonomics** — **#29–#31** as pilot demand; tag when band criteria met  

**Also before closing 0.7.x:** update **ROADMAP** “Last reviewed”, **selfhosted** in-cluster docs when RBAC/rules change for Helm/`exec`/`pvc`.

---

## Why 0.7.x

After **0.6.x** (notify, verify, probe, preflight, structured logs), operators need:

1. **Distroless / in-cluster** runs without bastion kubeconfig mounts  
2. **`release.*`** without host **`helm`** + **`.sh`** scripts (**Helm SDK**)  
3. **Data reset and maintenance** primitives (**`pvc`**, **`exec`**) without shell in the image  
4. **Safer logs and hooks** — **#17** shipped in **v0.7.1** (redaction + **`--no-env-passthrough`**)

**1.0.0** items (**default native**, PVC strategy doc, product-repo kind CI) stay in [ROADMAP.md](../ROADMAP.md) **1.0.0**.

---

## #17 — shipped in v0.7.1

| Area | Shipped (**v0.7.1**) |
|------|----------------------|
| Notify HTTP **errors** | Webhook URLs and routing keys redacted in error strings |
| Notify **payloads** | `Meta.Error` and sensitive fields scrubbed before POST |
| Engine **logs** (`text` / `json`) | Common secret patterns scrubbed; timestamped **`[DBG|INF|WRN|ERR]`** lines since **v0.7.3** |
| Hook / subprocess **environment** | **`--no-env-passthrough`** / **`run.no_env_passthrough: true`** → only `KZERO_*`, optional `KUBECONFIG`, correlation fields |

See [ROADMAP.md](../ROADMAP.md) **0.6.x #17** and **CHANGELOG** **[0.7.1]**.

---

## Success criteria (0.7.x band) — all met except optional #29

| # | Criterion | Roadmap | Status |
|---|-----------|---------|--------|
| 1 | Empty **`run.kubeconfig`** in a Pod uses in-cluster SA; **`Kubernetes target:`** shows `in-cluster` block | — | **Done** (**v0.7.1**) |
| 2 | **`--no-env-passthrough`** / config flag documented; hooks do not inherit host secrets when enabled | **#17** | **Done** (**v0.7.1**) |
| 3 | Notify payloads and engine error logs scrub webhook URLs, routing keys, and common secret patterns | **#17** | **Done** (**v0.7.1**) |
| 4 | **`release.*`** live **up**/**down** via **helm.sh/helm/v3**; analyze shows SDK plan | **#25** | **Done** (**v0.7.2**) |
| 5 | **`pvc`** step deletes named PVCs via API | **#24** | **Done** (**v0.7.2**) |
| 6 | **`exec`** step runs command in pod/container via remotecommand | **#23** | **Done** (**v0.7.2**) |
| 7 | **`infra_probe`** checks runnable with native step types (no probe `.sh` required) | **#26** | **Done** (**v0.7.2**) |
| 8 | Pods **Pending** due to selectors/taints surfaced in verify or probe | **#27** | **Done** (**v0.7.2**) |
| 9 | Total coverage ≥ 80%; `make release-check` green | — | **Done** (band-close tags) |

**In-cluster PO readiness (scale-only):** criteria **1** + [kzero-selfhosted `run/in-cluster/`](https://github.com/hrodrig/kzero-selfhosted/tree/develop/run/in-cluster) — met after **0.7.1** tag.

**develop2-full in-cluster (distroless):** criteria **4–7** minimum.

---

## Implementation order (PR-sized slices)

Merge to **`develop`** in this order. Each PR: `make lint`, `make test`, `make cover-check`.

| PR | Item | Roadmap | Notes |
|----|------|---------|--------|
| PR1 | Secret redaction + env passthrough | **#17** | Close 0.6.x carry-over |
| PR2 | In-cluster release (**0.7.1**) | — | **Done** — CHANGELOG, SPEC, ROADMAP, tag **`v0.7.1`** |
| PR3 | Helm SDK MVP | **#25** | **Done** — commit on develop |
| PR4 | **`pvc`** step | **#24** | **Done** on develop |
| PR5 | **`exec`** step | **#23** | **Done** on develop |
| PR6 | Infra probe native | **#26** | **Done** — native Redis sample + docs + selfhosted RBAC |
| PR7 | Scheduling sanity + **#29–#31** (incl. OCI registry login **#31**) + band close | **#27**, **#29–#31** | **Single tag** **`v0.7.2`** |

---

### PR1 — Secret redaction and env passthrough (#17)

**Scope:** shared redaction helper; no new step types.

**New / changed:**

- `internal/redact` (or extend `internal/notify`): `RedactString(s string) string`, `RedactURL`, patterns for `Bearer …`, basic `key=value` secret env echoes  
- **Notify:** redact `Meta.Error` and routing keys in logged failures before `buildPayload` / `joinErrors`  
- **Engine / slog:** run subprocess hook output and structured error fields through redactor when writing logs  
- **CLI:** global flag **`--no-env-passthrough`** on `down`, `up`, `reset`; config **`run.no_env_passthrough: true`**  
- **`LiveRunner.envFor`:** when passthrough off, start from empty slice + `correlation.AppendEnv` + optional `KUBECONFIG` only  

**SPEC:**

- Document flag and config; list what hooks still receive (`KZERO_*` table unchanged)  
- Warn that passthrough off may break hooks that relied on host `PATH` / cloud creds — intentional  

**Acceptance:**

- Unit tests: redact webhook URL, `Bearer abc`, `SLACK_WEBHOOK_URL=https://…` in log lines  
- Hook test: with passthrough off, subprocess env lacks `HOME`/`USER` but has `KZERO_PHASE`  
- `make cover-check` green  

---

### PR2 — In-cluster auth release (0.7.1)

**Scope:** release-only for code already on `develop` (`LoadRESTConfig`, `targetFromInCluster`, selfhosted smoke).

- **CHANGELOG** `[0.7.1]`: InClusterConfig when `run.kubeconfig` empty  
- **SPEC:** in-cluster Job note (empty kubeconfig, SA RBAC per pipeline namespace)  
- **ROADMAP:** move in-cluster enabler to Shipped or note under **0.7.1**  
- **VERSION** `0.7.1`; merge `develop` → `main`; tag **`v0.7.1`**; refresh GHCR tag in selfhosted manifests  

**Acceptance:** `make test-kind-in-cluster` passes against **`ghcr.io/hrodrig/kzero:v0.7.1`** (pull, not local build).

---

### PR3 — Helm SDK executor (#25)

**Scope:** largest PR; replaces shell **`helm`** for built-in **`release.*`** when `run.execution: native` (or new `run.helm: sdk|shell` if needed — prefer extending **`run.execution`** or explicit **`run.helm_executor`** documented in SPEC).

**MVP:**

- **Down:** `helm uninstall` equivalent with wait (matches today’s live down)  
- **Up:** `upgrade --install` with wait; values from config (minimal: chart path or OCI ref — align with develop2 pilot needs)  
- **Analyze:** show SDK operations instead of `script: …sh` when SDK path selected  
- **Tests:** helm action mocks / fake kube client patterns; no cluster required for unit gate  

**Defer within #25 (follow-up PR or 0.7.x patch):**

- OCI registry login from config (**#31** overlap)  
- Non-flat **`helm.workspace`** paths  

**Acceptance:** kind or envtest optional integration; develop2 subset runs **`release.*`** without `/bin/sh` or host `helm` in Job image.

**Operator:** extend [kzero-selfhosted `run/in-cluster/`](https://github.com/hrodrig/kzero-selfhosted) RBAC (Helm release secrets, configmaps) and sample pipeline with one probe release.

---

### PR4 — `pvc` step (#24)

**Schema:**

```yaml
pipelines:
  down:
    - pvc.database/data-postgresql-0
    - pvc.database/data-postgresql-1
```

- API delete with propagation policy documented  
- Analyze: list PVC refs; cluster validation optional Get  
- RBAC: `persistentvolumeclaims` delete in target namespaces  

**Acceptance:** fake clientset delete; kind smoke optional.

---

### PR5 — `exec` step (#23)

**Schema:**

```yaml
pipelines:
  down:
    - exec.database/postgresql-0:
        container: postgres
        command: ["psql", "-c", "TRUNCATE …"]
```

- `remotecommand` executor; timeout from step or `run.operation_timeout`  
- Analyze: pod/container ref validation when client available  

**Acceptance:** fake remotecommand test; replaces develop2 postgres truncate hook for in-cluster path.

---

### PR6 — Infra probe native (#26)

- Reuse **0.6.x #22** gate; implement checks with **`pvc`**, **`release.*` (SDK)**, **`exec`** instead of operator `.sh`  
- Selfhosted: native probe sample under `run/in-cluster/` or `run/examples/`  

**Acceptance:** `kzero probe` succeeds in Job without shell; distroless image.

---

### PR7 — Scheduling sanity (#27) + band close

- **#27:** after `up`, optional verify check or probe check for pods Pending (affinity/taints)  
- **#29–#31:** only if develop2 or pilot requires (`job`/`cronjob` suspend, `custom:` parity, helm path ergonomics)  
- **ROADMAP:** tick **#17**, **#23–#27**, **#29–#31** as shipped; **Last reviewed** date  
- **Band-close tag** (e.g. **`v0.7.2`**) bundles **#25–#27**, **#29–#31**, and develop2-full criteria **4–7** — no intermediate tags after **0.7.1**

---

## Release sequencing

| Tag | Contents |
|-----|----------|
| **`v0.7.0`** | **Done** — Cosign + SBOM (**#28**) |
| **`v0.7.1`** | **Done** — **#17** + InClusterConfig (PO / in-cluster smoke track) |
| **`v0.7.2`** | **Done** — band close: **#25–#27**, **#30–#31** (OCI registry login, **`pods_schedulable`**, **`custom:`** parity, **`script:`** paths); **#29** deferred **post-1.x** |
| **`v0.7.3`** | **Done** — text log levels (`--log-level`), Slack rich attachments + **`KZERO_NOTIFY_*`**, Helm SDK private OCI login hardening |
| **`v0.8.0` or `1.0.0`** | **1.0.0** contract items if band splits |

**Cadence (2026-06):** **PR3–PR7** merged on **`develop`**; **`v0.7.2`** closed the band. **`v0.7.3`** is a operator-facing patch (logs + notify + OCI) without new step types.

---

## Release checklist (band-close tag only)

Same gate as [plan-0.6.0.md](plan-0.6.0.md) § Release 0.6.0:

1. PR merged on **`develop`**; `make release-check` green  
2. **VERSION**, **CHANGELOG**, README badge, **ROADMAP** shipped rows  
3. BSD port sync if applicable  
4. VHS demo if CLI surface changed materially  
5. Merge **`develop` → `main`**; annotated tag **`vX.Y.Z`**; push  
6. Verify GHCR image; update selfhosted manifest pin  

---

## Test strategy

| Area | Approach |
|------|----------|
| **#17** redact | Table-driven string fixtures; hook env with/without passthrough |
| In-cluster | Product unit tests + selfhosted **`make test-kind-in-cluster`** |
| Helm SDK | helm `action` package mocks; optional kind + chart fixture |
| **`pvc` / `exec`** | fake clientset + fake remotecommand |
| Probe native | engine integration with stub executors |
| develop2 pilot | Private config repo; not in product CI gate initially |

---

## Diagram (target: in-cluster live `reset` with 0.7.x features)

```mermaid
sequenceDiagram
  participant Job as kzero Job (Pod)
  participant API as Kubernetes API
  participant H as Helm SDK
  participant N as notify

  Job->>API: InClusterConfig + preflight
  Job->>N: pipeline.start (redacted logs)
  Job->>API: infra_probe native
  Job->>H: release down (SDK)
  Job->>API: scale + pvc delete
  Job->>H: release up (SDK)
  Job->>API: exec / verify
  Job->>N: pipeline.success
```

---

## Post-band patch — v0.7.3 (2026-06-12)

Not part of the original PR1–PR7 slices; shipped after pilot validation:

| Item | Notes |
|------|--------|
| **Text log levels** | Global **`--log-level`**, timestamped **`YYYY/MM/DD HH:MM:SS: kzero - [LEVEL] -`** prefix; documented in SPEC § log levels |
| **Slack notify UX** | Colored attachments, fixed **`kzero {action}`** titles, footer **`kzero vX.Y.Z`** from build metadata |
| **`KZERO_NOTIFY_*`** | Env binding for all notify channel keys |
| **OCI login hardening** | Registry login before **`LocateChart`** on private **`oci://`** charts (pilot-tested) |

---

## References

- [ROADMAP.md](../ROADMAP.md) — **0.7.x** / **1.0.0** bands  
- [plan-0.6.0.md](plan-0.6.0.md) — **#17** explicitly deferred  
- [SPECIFICATIONS.md](../SPECIFICATIONS.md) — update per PR  
- [kzero-selfhosted run/in-cluster/](https://github.com/hrodrig/kzero-selfhosted/tree/develop/run/in-cluster) — Job/RBAC smoke  
- Handoff topic **`kzero/in-cluster-handoff`** — PO track after 0.7.1 + develop2 subset  
