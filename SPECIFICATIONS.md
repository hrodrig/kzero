# kzero v1 Specifications (TDD Baseline)

<a id="top"></a>

## 1. Purpose

`kzero` is a Go CLI that executes declarative, ordered Kubernetes workload pipelines.
Version 1 focuses on workload orchestration only (Deployments, StatefulSets, Helm release steps, and custom scripts).

This document is the source of truth for behavior and test expectations.

Visual overviews (Mermaid): **[docs/diagrams.md](docs/diagrams.md)**.

[↑ Back to top](#top)

## 2. Scope (v1)

### In scope
- Declarative config from YAML (`kzero.yaml` or `--config` path).
- Ordered phase execution: `down`, `up`, and `reset` (`down` then `up`).
- Phase lifecycle hooks: `pre-down`, `post-down`, `pre-up`, `post-up`, `on-error`.
- Optional **per-step** hooks: `pre` and `post` (shell scripts) scoped to a single pipeline step, executed immediately before and after that step’s main action.
- Pipeline step types:
  - compact resource refs (`<kind>.<namespace>/<name>`) for **`deployment`**, **`statefulset`**, **`release`**, **`pvc`**, **`exec`**, **`job`**, **`cronjob`**
  - Helm release refs (`release.<namespace>/<name>`) — shell script (**`run.execution: shell`**) or Helm SDK chart manifest (**`native`** / **`auto`**)
  - custom script steps (`{ custom: ./path/script.sh }`, optionally with `pre` / `post` in the same mapping)
- Run modes: `dry-run` and `live` (live can start as minimal implementation).
- **`retry` block** in YAML (`retry.attempts`, `retry.delay`): per-pipeline-step retries in **`live`** mode with exponential backoff **and full jitter** (see [Current engine: sequencing, retry, and concurrency](#current-engine-sequencing-retry-and-concurrency)).

### Out of scope
- Node lifecycle operations (`drain`, `cordon`, node deletion).
- Cloud/provider orchestration (`az`, `aws`, `gcloud`, Talos/k3s reset flows).
- Automatic nodepool scaling.

### Contract index (operator-facing)

Use this table with **`kzero analyze`** (stdout plan + optional **Deferred** summary) to see what the **1.0.0** contract engine honors today.

| Area | Status | Notes |
|------|--------|-------|
| **YAML schema `1.0`** (`schema_version`, `pipelines`, `run`) | **Implemented** | Required keys; unknown keys ignored with loader warnings. |
| **Phases** `down` / `up` / `reset` | **Implemented** | `reset` = `down` then phase-boundary preflight then `up`. |
| **Phase hooks** `pre-down`, `post-down`, `pre-up`, `post-up`, `on-error` | **Implemented** | Default **`/bin/sh`**; opt-in **`command.shell`** (see § Hook and script interpreter). |
| **Per-step hooks** `pre` / `post` on map steps | **Implemented** | Scoped to one pipeline step; same interpreter rules as phase hooks. |
| **Step types** `deployment`, `statefulset`, `release`, `pvc`, `exec`, `job`, `cronjob`, `custom` | **Implemented** | Compact refs + Helm workspace scripts / Helm SDK (`run.execution`). |
| **`run.mode`** `dry-run` / `live` | **Implemented** | Dry-run skips cluster mutations and notify POSTs. |
| **`run.execution`** `shell` / `native` / `auto` | **Implemented** | Default **`native`** when omitted (**#32**); `pvc` / `exec` / `job` / `cronjob` always native. |
| **`retry`** block | **Implemented** | Live mode only; exponential backoff; transient errors only. |
| **`notify.*`** channels + **`require_delivery`** | **Implemented** | Events: `pipeline.start`, `success`, `error`, `stalled`; `kzero notify test`. |
| **`run.api_watchdog`** | **Implemented** | `/healthz` probes; trips → `pipeline.stalled` (not user interrupt). |
| **`verify`**, **`infra_probe`**, **`kzero probe`** | **Implemented** | Post-up readiness and pre-destructive probe cache. |
| **`kzero target --output slug`** | **Implemented** | Filesystem-safe cluster slug for wrapper logs. |
| **`kzero doctor`** | **Implemented** | Config + binaries + API + workload refs + RBAC hints; no mutations (see § **`kzero doctor`**). |
| **`kzero completion`** / **`kubectl-kzero`** | **Implemented** | Shell completion scripts; optional kubectl plugin binary on **`PATH`**. |
| **Graceful shutdown** (SIGINT/SIGTERM) | **Implemented** | Cancels pipeline context; distinct from watchdog stall. |
| **Pipeline parallelism** (`run.worker_concurrency`) | **Out of scope** | Removed from contract **0.5.3**; use step order + `custom:`. |
| **Node / cloud lifecycle** | **Out of scope** | See §2 Out of scope. |
| **Deferred schema keys** | **None (1.0.0)** | `config.DeferredFeatureWarnings` returns empty; analyze **Deferred** section omitted when empty. |
| **Experimental** | **None declared** | No feature flags; stretch items live on [ROADMAP.md](ROADMAP.md) until promoted. |

**CI contract checks:**

| Gate | Where | Budget / flakes |
|------|-------|-----------------|
| Binary smoke (**#45**) | `testing/smoke/smoke.sh` → job **`e2e-smoke`** | No cluster; fast |
| Kind integration (**#34**) | `testing/kind/kind-e2e.sh` → job **`integration-kind`** | **`timeout-minutes: 20`**; one retry on kind create only — [testing/kind/README.md](testing/kind/README.md) |
| Unit / fake-client | `go test` incl. watchdog mid-wait | Default CI job **`test`** |

Operator-scale kind lab (multi-workload / PVC): [kzero-selfhosted `testing/kind`](https://github.com/hrodrig/kzero-selfhosted/tree/main/testing/kind) — not a product PR gate.

### Engine design principles

kzero stays **generic** and **configuration-driven**: the engine interprets validated YAML; it does not embed environment-specific playbooks.

**Implementation patterns to prefer (target architecture):**
- Phased workflows with explicit ordering (`pre-*`, pipeline steps, `post-*`).
- Readiness waits and per-step / global timeouts.
- **Strict sequential** pipeline execution in YAML order (parallel steps are out of scope for v1; use `custom:` scripts if you need controlled parallelism).
- Safe notifications: optional channels, redact or mask secrets in logs, include run mode and correlation metadata (e.g. `client.id`, cluster name).
- kubectl and Helm execution with explicit timeouts, structured logs, and **per-step retry** with exponential backoff for transient failures in **live** mode (see [Current engine: sequencing, retry, and concurrency](#current-engine-sequencing-retry-and-concurrency)).

**Avoid:**
- Hardcoding product- or tenant-specific resource lists or branching logic in Go; express that in YAML and hooks instead.

**Contract:** This document and the versioned YAML schema (`schema_version`) define observable behavior.

[↑ Back to top](#top)

## 3. Configuration Contract

[↑ Back to top](#top)

## Required top-level keys
- `schema_version` (string; expected `"1.0"` in v1)
- `pipelines`
- `run`

[↑ Back to top](#top)

## JSON Schema (editor aid)

Optional **[configs/kzero.schema.json](configs/kzero.schema.json)** for editor autocomplete (YAML Language Server `$schema`). Point a profile at it with:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/hrodrig/kzero/main/configs/kzero.schema.json
```

or a relative path when editing beside the file (see [configs/kzero.sample.yml](configs/kzero.sample.yml)). **Runtime validation is the Go loader** (`internal/config`); the JSON Schema is not consulted by `kzero` at run time.

[↑ Back to top](#top)

## Supported sections
- `cluster` (metadata only)
- `helm.workspace`
- `client.id`
- `command.helm`, `command.kubectl`, `command.shell` (script interpreter; default `/bin/sh`)
- `hooks.pre-down`, `hooks.post-down`, `hooks.pre-up`, `hooks.post-up`, `hooks.on-error`
- `pipelines.down` / `pipelines.up` list items; map-valued steps may include `pre` / `post` (per-step hook script paths), `replicas`, `wait_for_ready`, `timeout` where documented in §3
- `notify` (optional; outbound HTTP in **live** mode — see § notify)
- `verify` (optional; post-up readiness — see § **`kzero verify`**)
- `infra_probe` (optional; pre-destructive probe — see § **`kzero probe`**)
- `retry.attempts`, `retry.delay` (loaded; engine behavior: see subsection below)
- `run.kubeconfig`, `run.mode`, `run.execution`, `run.color`, `run.timeout`, `run.operation_timeout`, `run.no_env_passthrough`, `run.verify`, `run.probe_cache_dir`, `run.api_watchdog.{enabled,interval,fail_after}`

### `run.kubeconfig` and in-cluster auth

**Deployment posture:** Production **`down`**, **`reset`**, and recovery pipelines should run **out-of-band** (bastion or management host with kubeconfig). See **[docs/deployment-models.md](docs/deployment-models.md)**. In-cluster execution is **optional** and **not recommended** when API reliability is uncertain.

When **`run.kubeconfig`** is empty or omitted, the engine loads API credentials in order: default kubeconfig discovery (`KUBECONFIG`, `~/.kube/config`, in-cluster mount at `/var/run/secrets/kubernetes.io`), then **`rest.InClusterConfig()`** (Pod service account token). This supports **Job/CronJob** runs without mounting a kubeconfig Secret — use only for **non-destructive** or **CI/smoke** scenarios unless you accept shared fate with the cluster.

- Leave **`run.kubeconfig`** empty in in-cluster manifests; mount pipeline YAML via ConfigMap or Secret.
- The Job **ServiceAccount** needs RBAC in **each namespace referenced by pipeline steps** (`deployment.ns/name`, etc.), not only the Pod namespace.
- **`Kubernetes target:`** prints an **`in-cluster`** block (API server, service-account namespace) for audit; step namespaces come from compact refs.
- With **`run.execution: native`** or **`auto`**, **`release.*`** uses the **Helm SDK** (chart manifest under **`helm.workspace`**) instead of host **`helm`** / **`.sh`**. Operator Job examples (optional): [kzero-selfhosted `run/in-cluster/`](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/in-cluster).

**Removed from contract (schema 1.0):** `run.worker_concurrency` is **not** supported. Legacy configs that still set it are ignored (unknown key under `run`). Pipeline parallelism is intentionally out of scope; express ordering and optional batching in YAML step order and `custom:` scripts.

<a id="current-engine-sequencing-retry-and-concurrency"></a>
### Current engine: sequencing, retry, and concurrency

This subsection documents **observable behavior in the codebase today** (strictly sequential pipeline steps; **`cfg.Retry`** honored in **live** mode). It overrides informal “in scope” wording elsewhere when there is a conflict.

1. **Sequential pipeline steps:** For `kzero down` / `kzero up` / `kzero reset`, each entry in `pipelines.down` or `pipelines.up` runs **after** the previous step completes successfully. Steps do **not** run in parallel. Fail-fast: the first failing hook or step aborts the phase (see §5). On **down**, `deployment` / `statefulset` steps set replicas without waiting for pods to terminate unless a step defines its own wait semantics via hooks or future fields.
2. **`retry.attempts` and `retry.delay`:** In **`run.mode: live`**, each **pipeline step** (pre-hook, main step, post-hook as one unit) may be retried up to **`retry.attempts`** times. After failure *n*, the engine waits a **full-jitter** delay uniformly chosen in **`[0, retry.delay × 2^(n−1)]`** (exponential ceiling capped at **2m**) before the next try — so concurrent operators do not retry in lockstep (ROADMAP **#50**). Retries apply only to **transient** errors (API timeout/conflict/429/503, `context.DeadlineExceeded`, common connection/timeout strings including **`connection lost`** and **`http2: client connection lost`**). **`ErrNotFound`**, **`ErrForbidden`**, and **`context.Canceled`** are not retried. **`dry-run`** does not retry. A line `[retry] pipeline …` is written to the command output stream when a retry occurs (wait duration is the jittered value actually used).
3. **`notify`:** When **`run.mode: live`** and at least one channel is **`enabled`**, the CLI sends **`pipeline.start`** (after the **`Kubernetes target:`** block), **`pipeline.success`** on completion, **`pipeline.stalled`** when the API watchdog trips, and **`pipeline.error`** on fail-fast **before** the **`on-error`** hook. Channels: **`slack`** (colored attachment with fields such as `Cluster`, `Client`, `Context`, `User`, `Mode`, `Duration`; footer **`kzero vX.Y.Z`** from build metadata — see [docs/examples/notifications.md](docs/examples/notifications.md)), **`discord`**, **`teams`**, **`pagerduty`** (Events API v2), and **`webhook`** (generic JSON payload including **`kube_context`** when available). **`notify.on_error`** defaults to **true** when any channel is enabled. **`notify.require_delivery`** (default `false`; when true a failed **`pipeline.error`** or **`pipeline.stalled`** POST fails the pipeline with exit **4** — see [docs/examples/notifications.md](docs/examples/notifications.md) and §5). Override channel keys via **`KZERO_NOTIFY_*`** env vars (same binding rules as other **`KZERO_*`** keys). **`dry-run`** does not send pipeline notifications. Use **`kzero notify test`** to POST a test event without running a pipeline (see § **`kzero notify test`**). Webhook URLs, bearer tokens, and common `*_TOKEN` / `*_KEY` / `*_SECRET` patterns are redacted in notify payloads and engine logs.
4. **Secret redaction and hook environment:** Engine logs (text and JSON), notify **`pipeline.error`** payloads, and subprocess stdout/stderr written to the output stream scrub common secret patterns. Set **`run.no_env_passthrough: true`** or pass **`--no-env-passthrough`** on **`down`** / **`up`** / **`reset`** to omit the host **`os.Environ()`** from hook, **`custom:`**, **`release`**, and **`kubectl`** subprocesses — only **`KZERO_*`**, optional **`KUBECONFIG`**, and correlation fields remain. Hooks that depend on host **`PATH`**, cloud SDK env vars, or other inherited credentials must not use this flag.
5. **CLI warnings:** Deferred-feature warnings are reserved for schema fields not yet implemented; **`notify.*`** channels no longer emit deferred warnings once enabled.

### Workload execution backend (`run.execution`)

When `run.mode` is `live`, `deployment` and `statefulset` steps use a **Workload** executor selected by `run.execution` (default **`native`** if omitted — **1.0.0 #32**; set **`shell`** explicitly for kubectl/helm subprocesses):

| Value | Behavior |
|-------|----------|
| `shell` | `kubectl scale` and `kubectl rollout status` (subprocess; honors `command.kubectl` and `KUBECONFIG` from `run.kubeconfig`). **Opt-in** after **1.0.0**. |
| `native` | `k8s.io/client-go`: update workload replica count and poll readiness (no `kubectl` for scale/wait). Requires a valid kubeconfig / in-cluster config. **`release.*`** steps use **Helm SDK** (`helm.sh/helm/v4`) instead of shell **`helm`** / **`.sh`** scripts. **`pvc.*`** steps delete claims via the API (always native; ignores `run.execution`). **`exec.*`** steps run commands in a pod/container via **remotecommand** (always native). **`job.*`** / **`cronjob.*`** steps are always native (Job create/delete/wait; CronJob suspend). **Default** when `run.execution` is omitted. |
| `auto` | Try **native** (workloads + Helm SDK for releases); on client init failure, fall back to **shell** and print a one-line notice on the run output stream. |

Hooks, `custom:` steps, and per-step `pre`/`post` use **`command.shell`** (default **`/bin/sh`**) regardless of `run.execution`. Shell-path **`release.*`** install scripts use the same interpreter.

### Hook and script interpreter (`command.shell`)

Phase hooks, per-step **`pre`/`post`**, **`custom:`** scripts, and **`release.*`** install scripts on the **shell** path are invoked as:

```text
<command.shell| /bin/sh> <script-path> [args…]
```

(not as an executable that follows a shebang). A leading `#!/bin/bash` (or similar) in the file is **ignored**.

| Key | Default | Description |
|-----|---------|-------------|
| **`command.shell`** | **`/bin/sh`** when omitted or empty | Absolute path or `PATH` name of the interpreter (e.g. **`/bin/bash`**, **`bash`**). Env override: **`KZERO_COMMAND_SHELL`**. |

**Operator contract (default `/bin/sh`):** scripts must be valid for the host’s **`/bin/sh`** (typically **POSIX**). On many Linux distros (notably **Ubuntu** / **Debian**) `/bin/sh` is **dash**, which rejects common bashisms such as:

- `set -o pipefail`
- `[[ … ]]` tests
- bash arrays / `${name[0]}`
- process substitution

Prefer `set -eu` (without `pipefail`) and POSIX constructs, **or** set **`command.shell: /bin/bash`** (or another interpreter) when operators intentionally use bashisms. Validate with `<interpreter> -n <script>` on the target OS before a live pipeline.

Shebang honor (running the file as an executable) remains **out of scope**.

### API watchdog (`run.api_watchdog`)

When `run.mode` is `live` and `run.api_watchdog.enabled` is `true`, the engine starts a **watchdog goroutine** that periodically probes the Kubernetes API `/healthz` endpoint. The watchdog runs for the duration of `down`/`up`/`reset`.

Config keys (all under `run.api_watchdog`):

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Activate the watchdog during live runs. |
| `interval` | Go duration | `60s` | Period between reachability probes. |
| `fail_after` | Go duration | `5m` | Cumulative unreachability deadline. When the API has been unreachable for at least this long, the watchdog **trips** — the current pipeline step is cancelled and the pipeline exits with `pipeline.stalled`. |

Env overrides: `KZERO_RUN_API_WATCHDOG_ENABLED`, `KZERO_RUN_API_WATCHDOG_INTERVAL`, `KZERO_RUN_API_WATCHDOG_FAIL_AFTER`.

Trips fire exactly once per pipeline run. Recovery resets the failure timer — a transient blip shorter than `fail_after` does not trip the watchdog.

API errors on the native path are wrapped with stable sentinels (`ErrNotFound`, `ErrForbidden`, `ErrConflict` in `internal/executor`) for `errors.Is` in tests or `on-error` hooks.

Subprocess failures on the **shell** path (`kubectl`, `helm`, hook scripts) are classified with **`WrapSubprocess`**: exit codes and common stderr/stdout patterns map to the same sentinels plus **`ErrTransient`** for likely temporary network/API errors. **`retry`** honors **`ErrTransient`** the same way as native transient API errors.

### Supported workload kinds

Compact step references (`<kind>.<namespace>/<name>`) in `pipelines.down` and `pipelines.up` are validated against an explicit allow-list at config load time. Unsupported kinds are rejected by `kzero analyze` before any live execution.

| Kind | Down action | Up action |
|------|-------------|-----------|
| `deployment` | scale to 0 (`shell`: `kubectl scale`; `native`: API update) | scale to N (default 1; optional `wait_for_ready` → rollout wait) |
| `statefulset` | scale to 0 | scale to N (default 1; optional `wait_for_ready` → rollout wait) |
| `release` | **Shell:** `helm uninstall` (live); dry-run logs the same. **Native/auto:** Helm SDK uninstall (wait, ignore-not-found). | **Shell:** `<helm.workspace>/<name>.sh` with arg `up`. **Native/auto:** Helm SDK `upgrade --install` from `<helm.workspace>/<name>.yaml` (or step overrides) |
| `pvc` | delete claim via API (`DeletePropagationBackground`, ignore-not-found) | same (delete only; typically used on **`down`** after scale-down) |
| `exec` | run `command` in pod container via API exec subresource | same |
| `cronjob` | patch `spec.suspend: true` | patch `spec.suspend: false` |
| `job` | delete Job (`DeletePropagationBackground`, ignore-not-found) | create from **`manifest`** YAML (single `batch/v1 Job`) + optional wait for Complete |

**Job options** (job steps only):

| Option | Required | Default | Notes |
|--------|----------|---------|-------|
| `manifest` | on **`pipelines.up`** (and infra_probe up) | — | Path to Job YAML (cwd-relative or absolute). Step namespace/name override metadata when set. |
| `wait_for_complete` | no | `true` | On up: poll until `Complete=True`; fail on `Failed=True` or timeout. Dry-run: plan + optional API DryRun create; no wait. |

**PVC / StatefulSet data reset** is composed from these primitives (scale → wait → `pvc.*`, in-place `exec`, snapshot/`custom:`, init Job via **`job.*`**). Patterns: [docs/examples/pvc-statefulset-data-strategy.md](docs/examples/pvc-statefulset-data-strategy.md). Generic CRD patch/scale is **not** a built-in kind yet (use **`custom:`**; follow-up **#29b**).

`daemonset` is **not** a built-in kind in v1 because the Kubernetes API server does not expose a `/scale` subresource for DaemonSet, so `kubectl scale daemonset/...` returns `Error from server (NotFound): the server could not find the requested resource`. Configs that reference `daemonset.<ns>/<name>` are rejected at parse time.

To drain DaemonSet pods as part of a pipeline, use a `custom:` step that patches a `nodeSelector` no node satisfies, and reverses it on `up`:

```yaml
pipelines:
  down:
    - custom: ./hooks/daemonset-disable.sh
  up:
    - custom: ./hooks/daemonset-enable.sh
```

Where `daemonset-disable.sh` runs something like:

```sh
kubectl -n kube-system patch daemonset fluent-bit --type=strategic \
  -p '{"spec":{"template":{"spec":{"nodeSelector":{"kzero.io/disabled":"true"}}}}}'
```

and `daemonset-enable.sh` removes that nodeSelector key. A future minor release may add a first-class `daemonset` step type built on top of this pattern.

[↑ Back to top](#top)

## Pipeline syntax
- String step: `<kind>.<namespace>/<name>`
  - Examples:
    - `deployment.argocd/argocd-server`
    - `statefulset.database/postgresql`
    - `release.monitoring/kube-prometheus-stack`
    - `pvc.database/data-postgresql-0`
    - `exec.database/postgresql-0`:
        container: postgres
        command: ["psql", "-c", "TRUNCATE …"]
- Map step with one key (resource or `custom`):
  - `custom: ./hooks/example-custom.sh`
  - `custom` mapping may include **only** these additional keys: `pre`, `post` (each a non-empty string path to a shell script).
  - Resource map value (object) may include:
    - Up / scale options: `replicas`, `wait_for_ready`, `timeout`
    - Per-step hooks: `pre`, `post` (non-empty string paths; see below)
    - **`exec.*`** options: `container` (required), `command` (required string list), optional `stdin`, `timeout`
    - **`job.*`** options: `manifest` (required on up), optional `wait_for_complete` (default true), `timeout`
  - Example (per-step hooks on a StatefulSet):
    - `statefulset.database/postgresql: { pre: ./hooks/before-pg.sh, post: ./hooks/after-pg.sh }`
  - Example (custom step with hooks):
    - `custom: ./hooks/main.sh` plus sibling YAML keys `pre:` and `post:` under the same list item.

### Ordered steps vs full termination on `down`

YAML list order only ensures step *i+1* starts after step *i* succeeds. For `deployment` / `statefulset` on **`down`**, the main action is replica count **0**; the engine does **not** wait for pods to terminate unless you add per-step hooks (or a future first-class wait field).

**Example (downstream workload must not scale until upstream pods are gone):**

```yaml
pipelines:
  down:
    - deployment.app/consumer:
        post: ./hooks/wait-deployment-scale-down.sh
    - deployment.app/producer
```

The `post` script typically runs `kubectl rollout status deployment/consumer` (see [kzero-selfhosted reference hook](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/hooks/wait-deployment-scale-down.sh)). If `post` fails, `producer` is never scaled.

**`wait_for_ready`** applies on **`up`** after scale-up, not for pod drain on **`down`**.

**Waiting between steps on `up`** (Helm `--wait` in release scripts, `post` on `release.*`, `wait_for_ready` on workloads): [docs/examples/waiting-between-pipeline-steps.md](docs/examples/waiting-between-pipeline-steps.md).

More examples (StatefulSet `pre` before scale, assert scripts): [docs/examples/pipeline-order-and-integrity.md](docs/examples/pipeline-order-and-integrity.md).

### Per-step `pre` / `post` behavior (live mode)

For each pipeline step, when `run.mode` is `live`:

1. If `pre` is set, run **`command.shell`** (default `/bin/sh`) on `<pre>` before the step’s main action (same interpreter rules as [Hook and script interpreter](#hook-and-script-interpreter-commandshell)).
2. Run the main action (workload scale / rollout wait per `run.execution`, or release script / custom script).
3. If `post` is set, run the same interpreter on `<post>` **only if** the main action succeeded.

If `pre` fails, the main action and `post` for that step do not run; the phase fails and `hooks.on-error` applies per the global failure policy.

**`custom:`** main scripts receive the same **`KZERO_*`** step metadata as per-step hooks (`KZERO_PHASE`, `KZERO_PIPELINE_STEP_INDEX`, `KZERO_STEP_HOOK=main`, `KZERO_STEP_CUSTOM`, …).

**Environment variables** passed to per-step hook scripts (in addition to the process environment when **`run.no_env_passthrough`** is false, and optional **`KUBECONFIG`** from **`run.kubeconfig`**):

| Variable | Meaning |
|----------|---------|
| `KZERO_CLIENT_ID` | Set when `client.id` is configured (same value as YAML / `KZERO_CLIENT_ID` env override) |
| `KZERO_OS_USER` | OS username of the kzero process (`(unknown)` in the target block when unavailable) |
| `KZERO_OS_UID` | Numeric UID of the kzero process when resolvable |
| `KZERO_PHASE` | `down` or `up` |
| `KZERO_PIPELINE_STEP_INDEX` | Zero-based index of this step in the phase’s pipeline list |
| `KZERO_STEP_HOOK` | `pre`, `post`, or **`main`** (custom pipeline step body) |
| `KZERO_STEP_REF` | Set when the step has a compact ref (e.g. `deployment.ns/app`) |
| `KZERO_STEP_CUSTOM` | Set when the step is a `custom` script path |
| `KZERO_STEP_TYPE`, `KZERO_STEP_NAMESPACE`, `KZERO_STEP_NAME` | Set for `deployment`, `statefulset`, `release`, `pvc`, `exec`, `job`, and `cronjob` steps |
| `KZERO_RELEASE_NAME`, `KZERO_RELEASE_NAMESPACE` | Set for **`release`** steps (per-step `pre`/`post` and release `.sh` scripts) |

Release `.sh` scripts also receive `KZERO_PHASE` on install (see engine implementation).

[↑ Back to top](#top)

## Helm workspace contract (`helm.workspace`)

**0.6.x** documents the flat script layout used today; **0.7.x** adds optional non-flat **`script:`** paths, OCI registry login for the Helm SDK, and chart manifests under **`helm.workspace`**.

### Script resolution

| Pipeline step | Live **up** (shell) | Live **up** (native/auto SDK) | Live **down** |
|---------------|---------------------|-------------------------------|---------------|
| `release.<namespace>/<name>` | `/bin/sh <helm.workspace>/<name>.sh` with first arg `up` (override with step **`script:`**) | **Helm SDK** `upgrade --install` from `<helm.workspace>/<name>.yaml` (or step `chart:` / `version:` overrides) | `helm uninstall` (shell) or **Helm SDK** uninstall with wait (native/auto) |

Rules:

- **Basename = release name** — `release.monitoring/kube-prometheus-stack` → `<helm.workspace>/kube-prometheus-stack.sh` by default (namespace in the step ref is **not** part of the filename).
- **Non-flat shell path** — optional step map **`script: monitoring/kube-prometheus-stack.sh`** (relative to **`helm.workspace`**) overrides the default **`<name>.sh`** layout (**0.7.2 #31**).
- **Required when** any `pipelines` or `infra_probe.pipeline` step uses `release.*` (`helm.workspace` validation error if missing).
- **Path** may be relative to the process working directory or absolute; operators often set an absolute path in CI/cron.
- **Analyze** prints `script: <helm.workspace>/<name>.sh` for each `release.*` **up** step when **`run.execution: shell`**; with **`native`** / **`auto`**, prints **`helm upgrade --install (sdk, …)`** from the chart manifest or step overrides.

### Helm SDK chart manifest (`<helm.workspace>/<release>.yaml`)

When **`run.execution`** is **`native`** or **`auto`**, each **`release.*`** **up** step loads chart metadata from **`<helm.workspace>/<release-name>.yaml`** (same basename convention as **`.sh`** scripts):

```yaml
chart: oci://registry-1.docker.io/bitnamicharts/redis   # or path relative to helm.workspace
version: "25.5.2"
values_files:
  - kzero-probe-redis-values.yaml
create_namespace: true
wait: true
timeout: 10m
```

Optional **step map** overrides (same release step): `script` (shell path), `chart`, `version`, `values_files`, `create_namespace`, `timeout` (see pipeline step options).

### OCI registry authentication (`helm.registries`)

When **`run.execution`** is **`native`** or **`auto`** and a chart ref uses **`oci://`**, the Helm SDK logs into matching registries before **`LocateChart`**:

```yaml
helm:
  workspace: ./charts
  registries:
    - host: ghcr.io
      username: myuser
      password_env: HELM_REGISTRY_PASSWORD   # preferred over inline password
```

- **`host`** must match the OCI registry host in the chart URL (e.g. `oci://ghcr.io/org/chart` → `ghcr.io`).
- **`username`** is required per entry; supply **`password`** or **`password_env`** (env var read at runtime).
- Public OCI charts need no **`registries`** entry.

**Down** always runs SDK uninstall (wait, ignore-not-found) when the SDK path is selected — no **`.sh`** on down in either mode.

### Release install script environment

Release `.sh` scripts (probe or production) receive at least:

| Variable | Meaning |
|----------|---------|
| `KZERO_PHASE` | `up` when the script runs (install/upgrade) |
| `KZERO_RELEASE_NAME` | Release **name** from the step ref |
| `KZERO_RELEASE_NAMESPACE` | Namespace from the step ref |
| `KZERO_CLIENT_ID` | When `client.id` is set |
| `KZERO_OS_USER`, `KZERO_OS_UID` | Operator audit (see target block) |

Per-step `pre` / `post` on `release.*` steps use the same hook env table (including `KZERO_RELEASE_*`).

### Operator responsibilities

- Maintain one `.sh` per release name (install/upgrade logic, values files, `helm --wait`, registry auth).
- **Infra probe** charts follow the same layout (see [docs/examples/infra-probe.md](docs/examples/infra-probe.md)); kzero does not ship mandatory probe charts.
- **0.7.x** adds SDK-driven install without **`.sh`** when **`run.execution: native`**; configs that keep using **`.sh`** must continue to resolve `<helm.workspace>/<name>.sh` as today (**`run.execution: shell`**).

[↑ Back to top](#top)

## Preflight (live `down` / `up` / `reset`)

Before phase hooks in **live** mode, the engine calls the Kubernetes API (`Discovery().ServerVersion()`). Failure aborts with **`preflight: cannot reach Kubernetes API:`** (or kubeconfig load error). **Dry-run** prints a single plan line (`preflight: would verify Kubernetes API reachability`) and does not call the API. **`kzero analyze`** may emit a **warning** on stderr when preflight would fail in live mode (non-fatal).

**Engine log lines** (`[dry-run]`, `[retry]`, native dry-run messages) include a **`client_id=`** field when `client.id` is set (values with spaces are quoted). **`[live]`** lines omit **`client_id=`** (audit identity is printed once in the **`Kubernetes target:`** block, together with **`os_user`** and **`os_uid`**). In **`text`** mode every line is prefixed with **`YYYY/MM/DD HH:MM:SS: kzero - [LEVEL] -`** before the message body (**`LEVEL`**: **`DBG`**, **`INF`**, **`WRN`**, **`ERR`**). **`[live]`** and **`[dry-run]`** pipeline lines are **`INF`**; **`[retry]`** is **`WRN`**. **`[live]`** lines are emitted before scale, rollout wait, Helm uninstall, release scripts, and hook/custom script execution. Hook, custom, and release subprocesses receive **`KZERO_CLIENT_ID`** (when configured), **`KZERO_OS_USER`**, and **`KZERO_OS_UID`** in their environment.

### Per-step hooks in `dry-run`

Scripts are not executed. The engine prints planned invocations: a line for each non-empty `pre` / `post` hook (same labeling as other hooks) and the planned pipeline step line, in order.

### Ordering vs phase hooks

For a single phase (e.g. `down`), the order is:

`hooks.pre-down` → for each step: (**step `pre`** → **step body** → **step `post`**) → `hooks.post-down` (only if all steps and phase `post-*` prerequisites succeeded per fail-fast rules).

Unknown step option keys (outside the documented set) or unknown step shapes must fail validation.

[↑ Back to top](#top)

## 4. Command Behavior

[↑ Back to top](#top)

## `kzero analyze`
- Validates config and prints a **normalized execution plan** on **stdout**. Must not mutate cluster state.
- Exit code `0` on valid config; non-zero on invalid config.
- After a successful load, may print **non-fatal warnings** to **stderr** for deferred schema fields (when any remain unimplemented). Warnings do not change the exit code.

### Analyze stdout (v1)

In order (omit lines when the corresponding config value is empty):

1. **Header:** `Config`, `Schema`, `Run mode`; optional `Cluster`, `Client id`, `Run timeout`, `Helm workspace`.
2. **Phase hooks:** one line per set hook (`Hook pre-down:`, `Hook post-down:`, `Hook pre-up:`, `Hook post-up:`, `Hook on-error:`).
3. **Counts:** `Pipeline steps: down=N up=M`.
4. **`[down]`** section: for each step, `  <index>: <normalized step>` where the label uses the compact ref (e.g. `deployment.ns/app`, `pvc.ns/claim`, `exec.ns/pod`) or `custom: <path>`. Optional parenthetical metadata: `pre`, `post`, `replicas`, `wait_for_ready`, `timeout`, `container`, `command`; for `release.*` on **down**, `helm uninstall --wait --ignore-not-found` (shell) or SDK equivalent; on **up**, `script: …` (shell) or `helm upgrade --install (sdk, …)` (native/auto).
5. **`[up]`** section: same format as `[down]`.
6. **`Deferred`** block (only if any deferred field is set): heading `Deferred (accepted by schema; not implemented by v1 engine):` followed by bullet lines summarizing the same messages as stderr warnings.
7. **`Cluster validation`** (when the config lists at least one **`deployment`**, **`statefulset`**, **`pvc`**, **`exec`**, **`job`**, or **`cronjob`** step **and** a Kubernetes client can be built): heading `Cluster validation:` followed by one line per unique ref (`  OK  <ref>` or `  FAIL  <ref> (<reason>)`). Workloads: read-only **Get**; **`pvc`**: claim exists (or not-found OK); **`exec`**: pod exists and **`container`** is present; **`cronjob`**: object exists; **`job`**: exists or not-found OK (create on up). If any line is **FAIL**, `analyze` exits non-zero. If the client cannot be loaded, a **non-fatal** note is printed to **stderr** (`cluster validation skipped (...)`) and exit code stays **0** (plan-only mode).

`analyze` does **not** invoke the execution engine; it does **not** mutate cluster state. For planned hook/script invocations in `dry-run` mode, use `kzero down` / `kzero up` with `run.mode: dry-run`.

[↑ Back to top](#top)

## `kzero doctor`

Operator **preflight without mutations** (ROADMAP **#49**). Complements **`analyze`** (plan + optional cluster **Get** validation) by also checking host binaries and RBAC **can-i** hints before a live run.

### Checks (in order)

| Check | When | Pass / fail |
|-------|------|-------------|
| **config** | Always | Config file loads (`schema_version`, pipelines, `run`, …). Invalid YAML → command fails before other checks. |
| **binaries.kubectl** | **`run.execution`** is **`shell`** or **`auto`** (default when omitted is **`native`**) | **`command.kubectl`** or **`kubectl`** on **`PATH`**. Skipped with OK note when **`native`**. |
| **binaries.helm** | Same execution modes **and** at least one **`release.*`** step | **`command.helm`** or **`helm`** on **`PATH`**. Missing → **error** for **`shell`**, **warn** for **`auto`**. |
| **binaries.shell** | Config uses phase hooks, per-step **`pre`/`post`**, **`custom:`**, or shell-path **`release.*`** | **`command.shell`** or **`/bin/sh`**. Missing → **error**. |
| **kubernetes.api** | Always after config | Same handshake as live preflight (`Discovery().ServerVersion()` via **`run.kubeconfig`** / default rules). |
| **kubernetes.workloads** | Pipeline lists **`deployment`** / **`statefulset`** / **`pvc`** / **`exec`** / **`job`** / **`cronjob`** refs | Same read-only checks as **`analyze`** cluster validation (object exists; scalable workloads have **`spec.replicas`** set). |
| **kubernetes.rbac** | Those refs need scale/delete verbs | **`SelfSubjectAccessReview`** for **`get`/`update`/`patch`** on **`deployments`/`statefulsets`** (apps) and **`get`/`delete`** on **`persistentvolumeclaims`**, scoped to each step namespace. Denied → **error**. SAR API errors → **warn** (non-fatal). |

### Output and exit codes

- **`--output text`** (default): heading **`Doctor: OK`** or **`Doctor: FAIL`**, then one line per finding: **`[OK]`** / **`[WARN]`** / **`[ERROR]`** + check id + message.
- **`--output json`**: `{ "ok": bool, "findings": [{ "check", "severity", "message" }] }` (`severity` is **`ok`**, **`warn`**, or **`error`**).
- Exit **0** when no finding has severity **`error`** (warnings allowed).
- Exit **1** when config load fails; exit **2** when any finding is **`error`** (see [Process exit codes](#process-exit-codes-42)).

### Relation to other commands

| Command | Mutations | Plan | API | Binaries | RBAC SAR |
|---------|-----------|------|-----|----------|----------|
| **`analyze`** | No | Yes | Optional cluster **Get** | No | No |
| **`doctor`** | No | No | Required API ping + workloads + SAR | Yes (shell/auto) | Yes |
| **`down`/`up`/`reset` live** | Yes | Via engine logs | Preflight + steps | Used by shell executor | Enforced by API |

Does **not** run phase hooks, notify, verify, or **`infra_probe`**.

[↑ Back to top](#top)

## `kzero notify test`

- Loads **`notify.*`** from config and POSTs to **every enabled channel**. Does **not** contact the Kubernetes API or run pipeline steps.
- Default event: **`notify.test`**. Optional **`--event`**: `notify.test`, `pipeline.start`, `pipeline.success`, `pipeline.error`, `pipeline.stalled` (the last includes sample error text for formatting checks).
- Exit **0** when all channel POSTs succeed; **non-zero** when config is invalid, no channel is enabled, or any POST fails.
- Payload **`mode`** is **`test`** (independent of **`run.mode`** in YAML).

Operator cookbook (YAML per channel, env vars, troubleshooting): [docs/examples/notifications.md](docs/examples/notifications.md).

### Log levels (`--log-level`)

Four severity levels, aligned with [Microsoft.Extensions.Logging.LogLevel](https://learn.microsoft.com/dotnet/api/microsoft.extensions.logging.loglevel) (**Trace**, **Critical**, and **None** are out of scope for v1):

| CLI / JSON `level` | Tag (text) | Microsoft equivalent | Typical kzero output |
|--------------------|------------|----------------------|-------------------|
| **`debug`** | **`DBG`** | Debug | Analyze metadata (`Config:`, `Schema:`, `Run mode:`, retry/helm workspace lines) |
| **`info`** (default) | **`INF`** | Information | `Kubernetes target:`, pipeline plan (`[down]` / `[up]` steps), `[live]`, `[dry-run]`, cluster validation OK, subprocess stdout, command summary on success |
| **`warn`** | **`WRN`** | Warning | `[retry]`, deferred-feature warnings, cluster validation FAIL, verify FAIL |
| **`error`** | **`ERR`** | Error | Command summary when the command exits non-zero |

**Filtering:** `--log-level` sets the **minimum** severity (same as .NET). At **`info`**, **`DBG`** lines are hidden; at **`warn`**, only **`WRN`** and **`ERR`** appear.

Global flag on all commands: **`--log-level debug|info|warn|error`** (default **`info`**).

Global flag on all commands: **`--print-sample-config`** — writes sample YAML to **stdout** and exits (same content as **`configs/kzero.sample.yml`**). Use **`kzero --print-sample-config > kzero.yaml`** when the sample file is not on disk (e.g. Homebrew cask).

### Log format (`--log-format`)

Global flag on all commands (default **`text`**). Pipeline commands (`down`, `up`, `reset`, `notify test`) emit engine events through the structured logger:

| Mode | Engine stdout | Command summary (stderr) |
|------|---------------|---------------------------|
| **`text`** | Each line: `YYYY/MM/DD HH:MM:SS: kzero - [LEVEL] - …` where **`LEVEL`** is **`DBG`**, **`INF`**, **`WRN`**, or **`ERR`** (message body includes `[live]`, `[dry-run]`, analyze blocks, subprocess stdout). Filter with **`--log-level`** (default **`info`**). | Same prefix and level on `kzero <cmd> finished in …` (**`ERR`** when the command fails) |
| **`json`** | One JSON object per line (`time`, `app`, `level`, `kind`, `msg`, …) | JSON `command.summary` with `outcome` and `duration` |

Text lines follow operator maintenance conventions (timestamp, application name, severity, payload). Example:

```text
2026/06/11 16:13:08: kzero - [INF] - [live] scale deployment.app/frontend -> 0 replicas
2026/06/11 16:13:08: kzero - [DBG] - Config: kzero.yaml
2026/06/11 16:13:08: kzero - [WRN] - [retry] pipeline down step 3: transient error (attempt 2/3)
2026/06/11 16:13:08: kzero - [ERR] - kzero down failed after 2m11s
```

**`--log-level debug`** adds **`DBG`** analyze metadata; default **`info`** still prints the full pipeline plan and operational lines.

[↑ Back to top](#top)

## `kzero verify`

Read-only readiness checks after **`up`** (no mutations).

- **`verify.checks`** (default **`workloads_ready`**, **`nodes_ready`**):
  - **`workloads_ready`**: each unique **`deployment`** / **`statefulset`** in **`pipelines.up`** has **`ReadyReplicas`** (and **`UpdatedReplicas`**) ≥ desired count (`replicas` from YAML, default **1**).
  - **`nodes_ready`**: every node reports **`Ready=True`**.
  - **`pods_schedulable`** (**0.7.2 #27**): in each namespace referenced by **`pipelines.up`**, no pod stays **`Pending`** with **`PodScheduled=False`** / reason **`Unschedulable`** (affinity, taints, node selectors).
- Output: **`verify.format`** (`text` | `json`) or CLI **`--log-format json`**.
- Exit **0** when **`outcome: ready`**; **non-zero** when any check fails or the API is unreachable.
- **`run.verify: true`**: after a successful **`up`** or **`reset`** in **`live`** mode, run the same checks automatically (failure fails the command with **`post-up verify:`**).

JSON report shape: `{ "outcome", "cluster_name", "client_id", "checks": [{ "name", "ok", "items": [{ "ref", "ok", "detail" }] }] }`.

[↑ Back to top](#top)

## `kzero probe` / `infra_probe`

Optional **mini-pipeline** run before destructive **`down`** / **`reset`** to confirm **operator-maintained** platform paths still work: Helm/OCI chart pull, container registry (and **imagePullSecrets** if used), PVC **Bound** (StorageClass/CSI), and probe **`helm upgrade --install`** success. **kzero** does not ship a mandatory probe chart—you use the [anonymous Redis reference](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/infra-probe/kzero-probe-redis.sh) in **kzero-selfhosted** or any **`release.*`** / **`custom:`** steps you define.

```yaml
infra_probe:
  enabled: false
  before: ["reset"]       # default when enabled; include "down" to gate both
  fail_fast: true         # default true; false logs and continues main pipeline on probe failure
  cache_ttl: 0            # e.g. 30m — skip probe when last success is within TTL (live only)
  pipeline:
    up: [release.<ns>/probe-storage, ...]
    down: [release.<ns>/probe-storage, ...]
  checks:
    - pvc_bound: <namespace>/<claim>
    - release_ready: true
    - pods_schedulable: true   # optional; namespaces from infra_probe.pipeline.up
```

- **`kzero probe`**: runs **`pipeline.up`** → **`checks`** → **`pipeline.down`** (no main pipeline, no phase hooks).
- **Gate** (**`live`** only): when **`infra_probe.enabled`** and the command is listed in **`before`**, probe runs **before** the main pipeline (after **`Kubernetes target:`** and optional **`notify`** start).
- **Checks** (live): **`pvc_bound`** — PVC **`Status.Phase == Bound`**; **`release_ready`** — probe **`up`** completed without error; **`pods_schedulable`** — same scheduling sanity as **`kzero verify`** for probe pipeline namespaces.
- **Cache**: timestamp file under **`run.probe_cache_dir`** or OS user cache **`…/kzero/probe/probe-cache.json`**; invalidated when pipeline/check fingerprint changes.
- Probe steps use the same engine path as main pipelines (**`release.*`** via Helm SDK when **`run.execution`** is **`native`** / **`auto`**; shell scripts when **`shell`**).

Cookbook: [docs/examples/infra-probe.md](docs/examples/infra-probe.md). Reference assets: [kzero-selfhosted/run/examples/infra-probe/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/infra-probe).

[↑ Back to top](#top)

## `kzero down`
Execution order:
0. **`infra_probe`** gate when configured (**`live`** only; see above)
1. **Preflight** API reachability (**`live`**: fail-fast; **`dry-run`**: plan line)
2. `hooks.pre-down` (if set)
3. `pipelines.down` in strict list order; **each** step runs its optional `pre` script, then the step action, then its optional `post` script (see §3 Pipeline syntax).
4. `hooks.post-down` (only if step execution succeeded)

[↑ Back to top](#top)

## `kzero up`
Execution order:
1. **Preflight** (same as `down`)
2. `hooks.pre-up` (if set)
3. `pipelines.up` in strict list order; each step may run `pre` / `post` scripts as for `down`.
4. `hooks.post-up` (only if step execution succeeded)

[↑ Back to top](#top)

## `kzero reset`
Execution order:
0. **`infra_probe`** gate when configured (**`live`** only; before main `down`)
1. full `down` sequence
2. full `up` sequence

If `down` fails, `up` must not run.

[↑ Back to top](#top)

## 5. Failure Policy (Fail-Fast)

- Any failed hook or pipeline step aborts the current phase immediately.
- `hooks.on-error` runs once per failed command invocation.
- After `on-error`, command exits non-zero (see exit codes below).
- `post-*` hook for that phase must not run if phase execution failed.
- A step’s `post` hook must not run if that step’s `pre` hook failed or if the step’s main action failed.
- For `reset`, failure in `down` skips `up`.

### Process exit codes (#42)

Stable codes for wrappers and cron. Same pattern as [groot exitcode](https://github.com/hrodrig/groot/blob/main/internal/cmd/exitcode.go). Implementation: `internal/exitcode`. stderr messages stay the underlying error text (no `exit code N:` prefix).

| Code | Constant | When |
|------|----------|------|
| **0** | Success | Command completed without error |
| **1** | ConfigError | Config load/validate failure, invalid flags/args, or unclassified plain error |
| **2** | KubernetesError | Client/target resolve, cluster validation **FAIL**, verify/doctor check failures |
| **3** | ExecutorAborted | Pipeline step/hook failure, probe gate abort, watchdog stall / signal cancel (when not overridden by notify) |
| **4** | NotifyFailed | **`notify.require_delivery`** and error/stalled notify POST failed; or **`kzero notify test`** POST failure |

Scripts that only test **non-zero** remain compatible. Codes **2–4** are set only via explicit wraps where the failure class is unambiguous.

[↑ Back to top](#top)

## 6. Dry-Run Policy

- In `dry-run`, commands do not persist cluster mutations. Hook, `custom:`, and `release.*` steps are **plan-only** (logged, not executed).
- When `run.execution` is **`native`** or **`auto`** and a kubeconfig is available, `deployment` / `statefulset` steps use **server-side dry-run** (`Update` with `DryRun=All`): the API validates the scale without changing stored replica counts. Output includes `[dry-run] native scale … (server-side dry-run ok)` or a wrapped API error (e.g. `NotFound`, `Forbidden`). With **`shell`** execution, those steps remain plan-only (`[dry-run] pipeline …` lines).
- Hook and custom script execution policy for dry-run (v1):
  - default: do not execute scripts; print planned invocation.
  - Per-step `pre` / `post` scripts follow the same rule; the engine prints their planned paths in step order (see §3).
  - future option can allow dry-run script execution, but not required in v1.

[↑ Back to top](#top)

## 7. Logging and Output

- Every phase start/end is logged.
- Every step logs: phase, step index, step type, target, result.
- Failures include wrapped error context (`%w`) for root-cause tracing.

[↑ Back to top](#top)

## 8. TDD Plan and Acceptance Tests

[↑ Back to top](#top)

## A. Config parsing and validation
- `TestLoadConfig_MinimalValid`
  - Given minimal valid YAML, load succeeds.
- `TestLoadConfig_UnsupportedSchemaVersion`
  - Unsupported `schema_version` returns validation error.
- `TestLoadConfig_InvalidPipelineStep`
  - Unknown step format returns validation error.
- `TestLoadConfig_StatefulSetUpOptions`
  - `replicas`, `wait_for_ready`, `timeout` decode correctly.
- `TestLoadConfig_PipelineStepPrePost`
  - `pre` / `post` on a resource map step decode correctly.
- `TestLoadConfig_CustomStepWithPrePost`
  - `custom` step with sibling `pre` / `post` keys decodes correctly.
- `TestLoadConfig_CustomStepInvalidExtraKey`
  - Unknown keys on a `custom` map (e.g. `replicas`) fail validation.

[↑ Back to top](#top)

## B. Phase orchestration
- `TestDownOrder_PreStepsPost`
  - `pre-down` -> steps in order -> `post-down`.
- `TestDownOrder_PerStepPrePostAroundStep`
  - Global `pre-down` -> step `pre` -> step body -> step `post` -> `post-down` for a single scaled step with hooks.
- `TestUpOrder_PreStepsPost`
  - `pre-up` -> steps in order -> `post-up`.
- `TestResetOrder_DownThenUp`
  - Full down completes before up starts.

[↑ Back to top](#top)

## C. Failure handling
- `TestFailureInStep_TriggersOnErrorAndAbortsPhase`
  - Step failure triggers `on-error`, aborts remaining steps.
- `TestFailureInPreHook_TriggersOnErrorAndSkipsPhase`
  - Pre-hook failure triggers `on-error`; no pipeline steps run.
- `TestReset_FailureInDown_SkipsUp`
  - Down failure prevents any up hook/step execution.
- `TestPostHookNotRunAfterFailure`
  - `post-down`/`post-up` not executed after earlier phase failure.
- `TestFailureInPerStepPreHook_SkipsStepAndGlobalPost`
  - Failing step `pre` aborts before the main step; phase `post-down` does not run; `on-error` runs.
- `TestFailureInMainStep_SkipsPerStepPostHook`
  - Failing main step does not run that step’s `post` hook.

[↑ Back to top](#top)

## D. Dry-run behavior
- `TestDryRun_DoesNotExecuteMutatingOperations`
  - Workload/release actions are planned, not executed.
- `TestDryRun_CustomScriptIsPlannedOnly`
  - Custom scripts are not executed in default dry-run policy.

[↑ Back to top](#top)

## E. CLI surface
- `TestRootCommand_HasExpectedSubcommands`
  - analyze/doctor/down/up/reset are present.
- `TestAnalyze_InvalidConfigExitCode`
  - Analyze returns non-zero on invalid config.
- `TestDoctor_okJSON` / doctor package tests
  - Doctor report OK with fake client; binary and RBAC failures exit non-zero.

[↑ Back to top](#top)

## F. Live execution (per-step hooks)
- `TestLiveRunner_PerStepPreRunsBeforeKubectlWithStepEnv`
  - Step `pre` runs before `kubectl scale`; hook process receives `KZERO_PHASE`, `KZERO_PIPELINE_STEP_INDEX`, `KZERO_STEP_HOOK=pre`, and `KZERO_STEP_REF` as documented.

[↑ Back to top](#top)

## 9. Definition of Done (v1)

- All tests above implemented and passing via `make test`.
- `make lint` passes.
- `configs/kzero.sample.yml` matches the supported schema and examples.
- Command help text documents v1 scope and non-goals.

[↑ Back to top](#top)
