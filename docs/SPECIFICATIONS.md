# kzero v1 Specifications (TDD Baseline)

## 1. Purpose

`kzero` is a Go CLI that executes declarative, ordered Kubernetes workload pipelines.
Version 1 focuses on workload orchestration only (Deployments, StatefulSets, Helm release steps, and custom scripts).

This document is the source of truth for behavior and test expectations.

Visual overviews (Mermaid): **[diagrams.md](diagrams.md)**.

## 2. Scope (v1)

### In scope
- Declarative config from YAML (`kzero.yaml` or `--config` path).
- Ordered phase execution: `down`, `up`, and `reset` (`down` then `up`).
- Phase lifecycle hooks: `pre-down`, `post-down`, `pre-up`, `post-up`, `on-error`.
- Optional **per-step** hooks: `pre` and `post` (shell scripts) scoped to a single pipeline step, executed immediately before and after that step’s main action.
- Pipeline step types:
  - compact resource refs (`<kind>.<namespace>/<name>`)
  - Helm release refs (`release.<namespace>/<name>`)
  - custom script steps (`{ custom: ./path/script.sh }`, optionally with `pre` / `post` in the same mapping)
- Run modes: `dry-run` and `live` (live can start as minimal implementation).
- **`retry` block** in YAML (`retry.attempts`, `retry.delay`): per-pipeline-step retries in **`live`** mode with exponential backoff (see [Current engine: sequencing, retry, and worker concurrency](#current-engine-sequencing-retry-and-worker-concurrency)).

### Out of scope
- Node lifecycle operations (`drain`, `cordon`, node deletion).
- Cloud/provider orchestration (`az`, `aws`, `gcloud`, Talos/k3s reset flows).
- Automatic nodepool scaling.

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

## 3. Configuration Contract

## Required top-level keys
- `schema_version` (string; expected `"1.0"` in v1)
- `pipelines`
- `run`

## Supported sections
- `cluster` (metadata only)
- `helm.workspace`
- `client.id`
- `command.helm`, `command.kubectl`
- `hooks.pre-down`, `hooks.post-down`, `hooks.pre-up`, `hooks.post-up`, `hooks.on-error`
- `pipelines.down` / `pipelines.up` list items; map-valued steps may include `pre` / `post` (per-step hook script paths), `replicas`, `wait_for_ready`, `timeout` where documented in §3
- `notify` (optional; outbound HTTP in **live** mode — see § notify)
- `verify` (optional; post-up readiness — see § **`kzero verify`**)
- `infra_probe` (optional; pre-destructive probe — see § **`kzero probe`**)
- `retry.attempts`, `retry.delay` (loaded; engine behavior: see subsection below)
- `run.kubeconfig`, `run.mode`, `run.execution`, `run.timeout`, `run.operation_timeout`, `run.no_env_passthrough`

### `run.kubeconfig` and in-cluster auth

When **`run.kubeconfig`** is empty or omitted, the engine loads API credentials in order: default kubeconfig discovery (`KUBECONFIG`, `~/.kube/config`, in-cluster mount at `/var/run/secrets/kubernetes.io`), then **`rest.InClusterConfig()`** (Pod service account token). This supports **Job/CronJob** runs without mounting a kubeconfig Secret.

- Leave **`run.kubeconfig`** empty in in-cluster manifests; mount pipeline YAML via ConfigMap or Secret.
- The Job **ServiceAccount** needs RBAC in **each namespace referenced by pipeline steps** (`deployment.ns/name`, etc.), not only the Pod namespace.
- **`Kubernetes target:`** prints an **`in-cluster`** block (API server, service-account namespace) for audit; step namespaces come from compact refs.
- **`release.*`**, phase hooks, and **`custom:`** still invoke **`/bin/sh`** on the shell path — with **`run.execution: native`** or **`auto`**, **`release.*`** uses the **Helm SDK** (chart manifest under **`helm.workspace`**) instead of host **`helm`** / **`.sh`**. Operator Job examples: [kzero-selfhosted `run/in-cluster/`](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/in-cluster).

**Removed from contract (schema 1.0):** `run.worker_concurrency` is **not** supported. Legacy configs that still set it are ignored (unknown key under `run`). Pipeline parallelism is intentionally out of scope; express ordering and optional batching in YAML step order and `custom:` scripts.

<a id="current-engine-sequencing-retry-and-concurrency"></a>
### Current engine: sequencing, retry, and concurrency

This subsection documents **observable behavior in the codebase today** (strictly sequential pipeline steps; **`cfg.Retry`** honored in **live** mode). It overrides informal “in scope” wording elsewhere when there is a conflict.

1. **Sequential pipeline steps:** For `kzero down` / `kzero up` / `kzero reset`, each entry in `pipelines.down` or `pipelines.up` runs **after** the previous step completes successfully. Steps do **not** run in parallel. Fail-fast: the first failing hook or step aborts the phase (see §5). On **down**, `deployment` / `statefulset` steps set replicas without waiting for pods to terminate unless a step defines its own wait semantics via hooks or future fields.
2. **`retry.attempts` and `retry.delay`:** In **`run.mode: live`**, each **pipeline step** (pre-hook, main step, post-hook as one unit) may be retried up to **`retry.attempts`** times. After failure *n*, the engine waits **`retry.delay × 2^(n−1)`** (capped at **2m**) before the next try. Retries apply only to **transient** errors (API timeout/conflict/429/503, `context.DeadlineExceeded`, common connection/timeout strings). **`ErrNotFound`**, **`ErrForbidden`**, and **`context.Canceled`** are not retried. **`dry-run`** does not retry. A line `[retry] pipeline …` is written to the command output stream when a retry occurs.
3. **`notify`:** When **`run.mode: live`** and at least one channel is **`enabled`**, the CLI sends **`pipeline.start`** (after the **`Kubernetes target:`** block), **`pipeline.success`** on completion, and **`pipeline.error`** on fail-fast **before** the **`on-error`** hook. Channels: **`slack`**, **`discord`**, **`teams`**, **`pagerduty`** (Events API v2), and **`webhook`** (generic JSON payload). **`notify.on_error`** defaults to **true** when any channel is enabled. **`dry-run`** does not send pipeline notifications. Use **`kzero notify test`** to POST a test event without running a pipeline (see § **`kzero notify test`**). Webhook URLs, bearer tokens, and common `*_TOKEN` / `*_KEY` / `*_SECRET` patterns are redacted in notify payloads and engine logs.
4. **Secret redaction and hook environment:** Engine logs (text and JSON), notify **`pipeline.error`** payloads, and subprocess stdout/stderr written to the output stream scrub common secret patterns. Set **`run.no_env_passthrough: true`** or pass **`--no-env-passthrough`** on **`down`** / **`up`** / **`reset`** to omit the host **`os.Environ()`** from hook, **`custom:`**, **`release`**, and **`kubectl`** subprocesses — only **`KZERO_*`**, optional **`KUBECONFIG`**, and correlation fields remain. Hooks that depend on host **`PATH`**, cloud SDK env vars, or other inherited credentials must not use this flag.
5. **CLI warnings:** Deferred-feature warnings are reserved for schema fields not yet implemented; **`notify.*`** channels no longer emit deferred warnings once enabled.

### Workload execution backend (`run.execution`)

When `run.mode` is `live`, `deployment` and `statefulset` steps use a **Workload** executor selected by `run.execution` (default **`shell`** if omitted):

| Value | Behavior |
|-------|----------|
| `shell` | `kubectl scale` and `kubectl rollout status` (subprocess; honors `command.kubectl` and `KUBECONFIG` from `run.kubeconfig`). |
| `native` | `k8s.io/client-go`: update workload replica count and poll readiness (no `kubectl` for scale/wait). Requires a valid kubeconfig / in-cluster config. **`release.*`** steps use **Helm SDK** (`helm.sh/helm/v3`) instead of shell **`helm`** / **`.sh`** scripts. **`pvc.*`** steps delete claims via the API (always native; ignores `run.execution`). **`exec.*`** steps run commands in a pod/container via **remotecommand** (always native). |
| `auto` | Try **native** (workloads + Helm SDK for releases); on client init failure, fall back to **shell** and print a one-line notice on the run output stream. |

Hooks, `custom:` steps, and per-step `pre`/`post` always use `/bin/sh` regardless of `run.execution`.

API errors on the native path are wrapped with stable sentinels (`ErrNotFound`, `ErrForbidden`, `ErrConflict` in `internal/executor`) for `errors.Is` in tests or `on-error` hooks.

Subprocess failures on the **shell** path (`kubectl`, `helm`, hook scripts) are classified with **`WrapSubprocess`**: exit codes and common stderr/stdout patterns map to the same sentinels plus **`ErrTransient`** for likely temporary network/API errors. **`retry`** honors **`ErrTransient`** the same way as native transient API errors.

### Supported workload kinds

Compact step references (`<kind>.<namespace>/<name>`) in `pipelines.down` and `pipelines.up` are validated against an explicit allow-list at config load time. Unsupported kinds are rejected by `kzero analyze` before any live execution.

| Kind | Down action | Up action |
|------|-------------|-----------|
| `deployment` | scale to 0 (`shell`: `kubectl scale`; `native`: API update) | scale to N (default 1; optional `wait_for_ready` → rollout wait) |
| `statefulset` | scale to 0 | scale to N (default 1; optional `wait_for_ready` → rollout wait) |
| `release` | `helm uninstall <name> -n <namespace> --wait --ignore-not-found` (live); dry-run logs the same | `<helm.workspace>/<name>.sh` (install/upgrade script) |
| `pvc` | delete claim via API (`DeletePropagationBackground`, ignore-not-found) | same (delete only; typically used after scale-down on **`down`**) |
| `exec` | run `command` in pod container via API exec subresource | same |

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

**Waiting between steps on `up`** (Helm `--wait` in release scripts, `post` on `release.*`, `wait_for_ready` on workloads): [examples/waiting-between-pipeline-steps.md](examples/waiting-between-pipeline-steps.md).

More examples (StatefulSet `pre` before scale, assert scripts): [examples/pipeline-order-and-integrity.md](examples/pipeline-order-and-integrity.md).

### Per-step `pre` / `post` behavior (live mode)

For each pipeline step, when `run.mode` is `live`:

1. If `pre` is set, run `/bin/sh <pre>` before the step’s main action.
2. Run the main action (workload scale / rollout wait per `run.execution`, or release script / custom script).
3. If `post` is set, run `/bin/sh <post>` **only if** the main action succeeded.

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
| `KZERO_STEP_TYPE`, `KZERO_STEP_NAMESPACE`, `KZERO_STEP_NAME` | Set for `deployment`, `statefulset`, `release`, `pvc`, and `exec` steps |
| `KZERO_RELEASE_NAME`, `KZERO_RELEASE_NAMESPACE` | Set for **`release`** steps (per-step `pre`/`post` and release `.sh` scripts) |

Release `.sh` scripts also receive `KZERO_PHASE` on install (see engine implementation).

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
- **Infra probe** charts follow the same layout (see [examples/infra-probe.md](examples/infra-probe.md)); kzero does not ship mandatory probe charts.
- **0.7.x** adds SDK-driven install without **`.sh`** when **`run.execution: native`**; configs that keep using **`.sh`** must continue to resolve `<helm.workspace>/<name>.sh` as today (**`run.execution: shell`**).

## Preflight (live `down` / `up` / `reset`)

Before phase hooks in **live** mode, the engine calls the Kubernetes API (`Discovery().ServerVersion()`). Failure aborts with **`preflight: cannot reach Kubernetes API:`** (or kubeconfig load error). **Dry-run** prints a single plan line (`preflight: would verify Kubernetes API reachability`) and does not call the API. **`kzero analyze`** may emit a **warning** on stderr when preflight would fail in live mode (non-fatal).

**Engine log lines** (`[dry-run]`, `[retry]`, native dry-run messages) include a **`client_id=`** field when `client.id` is set (values with spaces are quoted). **`[live]`** lines omit **`client_id=`** (audit identity is printed once in the **`Kubernetes target:`** block, together with **`os_user`** and **`os_uid`**). In **`text`** mode every line is prefixed with **`YYYY/MM/DD HH:MM:SS: kzero - [LEVEL] -`** before the message body (**`LEVEL`**: **`DBG`**, **`INF`**, **`WRN`**, **`ERR`**). **`[live]`** and **`[dry-run]`** pipeline lines are **`INF`**; **`[retry]`** is **`WRN`**. **`[live]`** lines are emitted before scale, rollout wait, Helm uninstall, release scripts, and hook/custom script execution. Hook, custom, and release subprocesses receive **`KZERO_CLIENT_ID`** (when configured), **`KZERO_OS_USER`**, and **`KZERO_OS_UID`** in their environment.

### Per-step hooks in `dry-run`

Scripts are not executed. The engine prints planned invocations: a line for each non-empty `pre` / `post` hook (same labeling as other hooks) and the planned pipeline step line, in order.

### Ordering vs phase hooks

For a single phase (e.g. `down`), the order is:

`hooks.pre-down` → for each step: (**step `pre`** → **step body** → **step `post`**) → `hooks.post-down` (only if all steps and phase `post-*` prerequisites succeeded per fail-fast rules).

Unknown step option keys (outside the documented set) or unknown step shapes must fail validation.

## 4. Command Behavior

## `kzero analyze`
- Validates config and prints a **normalized execution plan** on **stdout**. Must not mutate cluster state.
- Exit code `0` on valid config; non-zero on invalid config.
- After a successful load, may print **non-fatal warnings** to **stderr** for deferred schema fields (when any remain unimplemented). Warnings do not change the exit code.

### Analyze stdout (v1)

In order (omit lines when the corresponding config value is empty):

1. **Header:** `Config`, `Schema`, `Run mode`; optional `Cluster`, `Client id`, `Run timeout`, `Helm workspace`.
2. **Phase hooks:** one line per set hook (`Hook pre-down:`, `Hook post-down:`, `Hook pre-up:`, `Hook post-up:`, `Hook on-error:`).
3. **Counts:** `Pipeline steps: down=N up=M`.
4. **`[down]`** section: for each step, `  <index>: <normalized step>` where the label uses the compact ref (e.g. `deployment.ns/app`) or `custom: <path>`. Optional parenthetical metadata: `pre`, `post`, `replicas`, `wait_for_ready`, `timeout`; for `release.*` steps on **down**, `helm uninstall --wait --ignore-not-found`; on **up**, `script: <helm.workspace>/<release>.sh`.
5. **`[up]`** section: same format as `[down]`.
6. **`Deferred`** block (only if any deferred field is set): heading `Deferred (accepted by schema; not implemented by v1 engine):` followed by bullet lines summarizing the same messages as stderr warnings.
7. **`Cluster validation`** (only when the config lists at least one `deployment` or `statefulset` step **and** a Kubernetes client can be built from `run.kubeconfig` / default loading rules): heading `Cluster validation:` followed by one line per unique workload ref (`  OK  <ref>` or `  FAIL  <ref> (<reason>)`). Checks use a read-only **Get** (existence and `spec.replicas` set). If any line is **FAIL**, `analyze` exits non-zero. If the client cannot be loaded, a **non-fatal** note is printed to **stderr** (`cluster validation skipped (...)`) and exit code stays **0** (plan-only mode).

`analyze` does **not** invoke the execution engine; it does **not** mutate cluster state. For planned hook/script invocations in `dry-run` mode, use `kzero down` / `kzero up` with `run.mode: dry-run`.

## `kzero notify test`

- Loads **`notify.*`** from config and POSTs to **every enabled channel**. Does **not** contact the Kubernetes API or run pipeline steps.
- Default event: **`notify.test`**. Optional **`--event`**: `notify.test`, `pipeline.start`, `pipeline.success`, `pipeline.error` (the last includes sample `failed_step` / `error` fields for formatting checks).
- Exit **0** when all channel POSTs succeed; **non-zero** when config is invalid, no channel is enabled, or any POST fails.
- Payload **`mode`** is **`test`** (independent of **`run.mode`** in YAML).

Operator cookbook (YAML per channel, env vars, troubleshooting): [examples/notifications.md](examples/notifications.md).

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

### Log format (`--log-format`)

Global flag on all commands (default **`text`**). Pipeline commands (`down`, `up`, `reset`, `notify test`) emit engine events through the structured logger:

| Mode | Engine stdout | Command summary (stderr) |
|------|---------------|---------------------------|
| **`text`** | Each line: `YYYY/MM/DD HH:MM:SS: kzero - [LEVEL] - …` where **`LEVEL`** is **`DBG`**, **`INF`**, **`WRN`**, or **`ERR`** (message body includes `[live]`, `[dry-run]`, analyze blocks, subprocess stdout). Filter with **`--log-level`** (default **`info`**). | Same prefix and level on `kzero <cmd> finished in …` (**`ERR`** when the command fails) |
| **`json`** | One JSON object per line (`time`, `app`, `level`, `kind`, `msg`, …) | JSON `command.summary` with `outcome` and `duration` |

Text lines follow operator maintenance conventions (timestamp, application name, severity, payload). Example:

```text
2026/06/11 16:13:08: kzero - [INF] - [live] scale deployment.cloudbridge/webui -> 0 replicas
2026/06/11 16:13:08: kzero - [DBG] - Config: kzero.yaml
2026/06/11 16:13:08: kzero - [WRN] - warning: notify.teams is accepted by schema but not implemented
2026/06/11 16:13:08: kzero - [ERR] - kzero down failed after 2m11s
```

**`--log-level debug`** adds **`DBG`** analyze metadata; default **`info`** still prints the full pipeline plan and operational lines.

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

Cookbook: [examples/infra-probe.md](examples/infra-probe.md). Reference assets: [kzero-selfhosted/run/examples/infra-probe/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/infra-probe).

## `kzero down`
Execution order:
0. **`infra_probe`** gate when configured (**`live`** only; see above)
1. **Preflight** API reachability (**`live`**: fail-fast; **`dry-run`**: plan line)
2. `hooks.pre-down` (if set)
3. `pipelines.down` in strict list order; **each** step runs its optional `pre` script, then the step action, then its optional `post` script (see §3 Pipeline syntax).
4. `hooks.post-down` (only if step execution succeeded)

## `kzero up`
Execution order:
1. **Preflight** (same as `down`)
2. `hooks.pre-up` (if set)
3. `pipelines.up` in strict list order; each step may run `pre` / `post` scripts as for `down`.
4. `hooks.post-up` (only if step execution succeeded)

## `kzero reset`
Execution order:
0. **`infra_probe`** gate when configured (**`live`** only; before main `down`)
1. full `down` sequence
2. full `up` sequence

If `down` fails, `up` must not run.

## 5. Failure Policy (Fail-Fast)

- Any failed hook or pipeline step aborts the current phase immediately.
- `hooks.on-error` runs once per failed command invocation.
- After `on-error`, command exits non-zero.
- `post-*` hook for that phase must not run if phase execution failed.
- A step’s `post` hook must not run if that step’s `pre` hook failed or if the step’s main action failed.
- For `reset`, failure in `down` skips `up`.

## 6. Dry-Run Policy

- In `dry-run`, commands do not persist cluster mutations. Hook, `custom:`, and `release.*` steps are **plan-only** (logged, not executed).
- When `run.execution` is **`native`** or **`auto`** and a kubeconfig is available, `deployment` / `statefulset` steps use **server-side dry-run** (`Update` with `DryRun=All`): the API validates the scale without changing stored replica counts. Output includes `[dry-run] native scale … (server-side dry-run ok)` or a wrapped API error (e.g. `NotFound`, `Forbidden`). With **`shell`** execution, those steps remain plan-only (`[dry-run] pipeline …` lines).
- Hook and custom script execution policy for dry-run (v1):
  - default: do not execute scripts; print planned invocation.
  - Per-step `pre` / `post` scripts follow the same rule; the engine prints their planned paths in step order (see §3).
  - future option can allow dry-run script execution, but not required in v1.

## 7. Logging and Output

- Every phase start/end is logged.
- Every step logs: phase, step index, step type, target, result.
- Failures include wrapped error context (`%w`) for root-cause tracing.

## 8. TDD Plan and Acceptance Tests

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

## B. Phase orchestration
- `TestDownOrder_PreStepsPost`
  - `pre-down` -> steps in order -> `post-down`.
- `TestDownOrder_PerStepPrePostAroundStep`
  - Global `pre-down` -> step `pre` -> step body -> step `post` -> `post-down` for a single scaled step with hooks.
- `TestUpOrder_PreStepsPost`
  - `pre-up` -> steps in order -> `post-up`.
- `TestResetOrder_DownThenUp`
  - Full down completes before up starts.

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

## D. Dry-run behavior
- `TestDryRun_DoesNotExecuteMutatingOperations`
  - Workload/release actions are planned, not executed.
- `TestDryRun_CustomScriptIsPlannedOnly`
  - Custom scripts are not executed in default dry-run policy.

## E. CLI surface
- `TestRootCommand_HasExpectedSubcommands`
  - analyze/down/up/reset are present.
- `TestAnalyze_InvalidConfigExitCode`
  - Analyze returns non-zero on invalid config.

## F. Live execution (per-step hooks)
- `TestLiveRunner_PerStepPreRunsBeforeKubectlWithStepEnv`
  - Step `pre` runs before `kubectl scale`; hook process receives `KZERO_PHASE`, `KZERO_PIPELINE_STEP_INDEX`, `KZERO_STEP_HOOK=pre`, and `KZERO_STEP_REF` as documented.

## 9. Definition of Done (v1)

- All tests above implemented and passing via `make test`.
- `make lint` passes.
- `configs/kzero.sample.yml` matches the supported schema and examples.
- Command help text documents v1 scope and non-goals.
