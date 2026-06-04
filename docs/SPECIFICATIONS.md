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
- **`retry` block** in YAML (`retry.attempts`, `retry.delay`): part of the configuration **contract**; the **current** engine does not perform automatic retries (see [Current engine: sequencing, retry, and worker concurrency](#current-engine-sequencing-retry-and-worker-concurrency)).

### Out of scope
- Node lifecycle operations (`drain`, `cordon`, node deletion).
- Cloud/provider orchestration (`az`, `aws`, `gcloud`, Talos/k3s reset flows).
- Automatic nodepool scaling.

### Engine design principles

kzero stays **generic** and **configuration-driven**: the engine interprets validated YAML; it does not embed environment-specific playbooks.

**Implementation patterns to prefer (target architecture):**
- Phased workflows with explicit ordering (`pre-*`, pipeline steps, `post-*`).
- Readiness waits and per-step / global timeouts.
- Bounded parallelism for independent steps (worker pool; cap via `run.worker_concurrency`) once implemented.
- Safe notifications: optional channels, redact or mask secrets in logs, include run mode and correlation metadata (e.g. `client.id`, cluster name).
- kubectl and Helm execution with explicit timeouts, structured logs, and a future retry policy for transient failures (today: fail-fast; see [Current engine: sequencing, retry, and worker concurrency](#current-engine-sequencing-retry-and-worker-concurrency)).

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
- `notify` (optional; channel handling may be no-op in early v1)
- `retry.attempts`, `retry.delay` (loaded; engine behavior: see subsection below)
- `run.kubeconfig`, `run.mode`, `run.timeout`, `run.worker_concurrency`, `run.operation_timeout`

<a id="current-engine-sequencing-retry-and-worker-concurrency"></a>
### Current engine: sequencing, retry, and worker concurrency

This subsection documents **observable behavior in the codebase today** (sequential `for` loops over pipeline steps; no reads of `cfg.Retry` or `cfg.Run.WorkerConcurrency` in the engine). It overrides informal “in scope” wording elsewhere when there is a conflict.

1. **Sequential pipeline steps:** For `kzero down` / `kzero up` / `kzero reset`, each entry in `pipelines.down` or `pipelines.up` runs **after** the previous step completes successfully. Steps do **not** run in parallel. Fail-fast: the first failing hook or step aborts the phase (see §5).
2. **`retry.attempts` and `retry.delay`:** Parsed and stored on the loaded `Config`, but the engine **does not** retry failed `kubectl`, `helm`, or hook subprocesses. A failure surfaces immediately; use `hooks.on-error`, external supervisors, or resilient wrapper scripts until retry logic ships.
3. **`run.worker_concurrency`:** Parsed and stored; the engine **does not** use it to schedule concurrent steps. Operators may keep the key for **forward compatibility** with a future worker pool.
4. **`notify.slack` / `notify.discord`:** Parsed and stored; the engine **does not** send webhooks or other notifications.
5. **CLI warnings:** After a successful config load, `kzero analyze`, `kzero down`, `kzero up`, and `kzero reset` print **non-fatal warnings** to **stderr** when `run.worker_concurrency > 1`, `retry.attempts > 1`, or `notify.slack.enabled` / `notify.discord.enabled` is true, because those settings are not honored by the v1 engine yet.
### Supported workload kinds

Compact step references (`<kind>.<namespace>/<name>`) in `pipelines.down` and `pipelines.up` are validated against an explicit allow-list at config load time. Unsupported kinds are rejected by `kzero analyze` before any live execution.

| Kind | Down action | Up action |
|------|-------------|-----------|
| `deployment` | `kubectl scale --replicas=0` | `kubectl scale --replicas=N` (N = step `replicas` or 1; optional `wait_for_ready` → `kubectl rollout status`) |
| `statefulset` | `kubectl scale --replicas=0` | `kubectl scale --replicas=N` (N = step `replicas` or 1; optional `wait_for_ready` → `kubectl rollout status`) |
| `release` | `<helm.workspace>/<name>.sh down` | `<helm.workspace>/<name>.sh up` |

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
- Map step with one key (resource or `custom`):
  - `custom: ./hooks/example-custom.sh`
  - `custom` mapping may include **only** these additional keys: `pre`, `post` (each a non-empty string path to a shell script).
  - Resource map value (object) may include:
    - Up / scale options: `replicas`, `wait_for_ready`, `timeout`
    - Per-step hooks: `pre`, `post` (non-empty string paths; see below)
  - Example (per-step hooks on a StatefulSet):
    - `statefulset.database/postgresql: { pre: ./hooks/before-pg.sh, post: ./hooks/after-pg.sh }`
  - Example (custom step with hooks):
    - `custom: ./hooks/main.sh` plus sibling YAML keys `pre:` and `post:` under the same list item.

### Per-step `pre` / `post` behavior (live mode)

For each pipeline step, when `run.mode` is `live`:

1. If `pre` is set, run `/bin/sh <pre>` before the step’s main action.
2. Run the main action (`kubectl scale` / `kubectl rollout status` / release script / custom script).
3. If `post` is set, run `/bin/sh <post>` **only if** the main action succeeded.

If `pre` fails, the main action and `post` for that step do not run; the phase fails and `hooks.on-error` applies per the global failure policy.

**Environment variables** passed to per-step hook scripts (in addition to the process environment and optional `KUBECONFIG` from `run.kubeconfig`):

| Variable | Meaning |
|----------|---------|
| `KZERO_PHASE` | `down` or `up` |
| `KZERO_PIPELINE_STEP_INDEX` | Zero-based index of this step in the phase’s pipeline list |
| `KZERO_STEP_HOOK` | `pre` or `post` |
| `KZERO_STEP_REF` | Set when the step has a compact ref (e.g. `deployment.ns/app`) |
| `KZERO_STEP_CUSTOM` | Set when the step is a `custom` script path |
| `KZERO_STEP_TYPE`, `KZERO_STEP_NAMESPACE`, `KZERO_STEP_NAME` | Set for `deployment`, `statefulset`, and `release` steps |

Release steps still receive release-specific variables on the **release** script invocation (`KZERO_RELEASE_NAME`, `KZERO_RELEASE_NAMESPACE`, etc., as implemented); per-step hooks use the table above for correlation.

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
- After a successful load, prints **non-fatal warnings** to **stderr** for deferred schema fields (same set as in [Current engine](#current-engine-sequencing-retry-and-worker-concurrency): `run.worker_concurrency > 1`, `retry.attempts > 1`, `notify.slack.enabled`, `notify.discord.enabled`). Warnings do not change the exit code.

### Analyze stdout (v1)

In order (omit lines when the corresponding config value is empty):

1. **Header:** `Config`, `Schema`, `Run mode`; optional `Cluster`, `Client id`, `Run timeout`, `Helm workspace`.
2. **Phase hooks:** one line per set hook (`Hook pre-down:`, `Hook post-down:`, `Hook pre-up:`, `Hook post-up:`, `Hook on-error:`).
3. **Counts:** `Pipeline steps: down=N up=M`.
4. **`[down]`** section: for each step, `  <index>: <normalized step>` where the label uses the compact ref (e.g. `deployment.ns/app`) or `custom: <path>`. Optional parenthetical metadata: `pre`, `post`, `replicas`, `wait_for_ready`, `timeout`; for `release.*` steps, `script: <helm.workspace>/<release>.sh`.
5. **`[up]`** section: same format as `[down]`.
6. **`Deferred`** block (only if any deferred field is set): heading `Deferred (accepted by schema; not implemented by v1 engine):` followed by bullet lines summarizing the same messages as stderr warnings.

`analyze` does **not** invoke the execution engine; it only lists the configured plan. For planned hook/script invocations in `dry-run` mode, use `kzero down` / `kzero up` with `run.mode: dry-run`.

## `kzero down`
Execution order:
1. `hooks.pre-down` (if set)
2. `pipelines.down` in strict list order; **each** step runs its optional `pre` script, then the step action, then its optional `post` script (see §3 Pipeline syntax).
3. `hooks.post-down` (only if step execution succeeded)

## `kzero up`
Execution order:
1. `hooks.pre-up` (if set)
2. `pipelines.up` in strict list order; each step may run `pre` / `post` scripts as for `down`.
3. `hooks.post-up` (only if step execution succeeded)

## `kzero reset`
Execution order:
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

- In `dry-run`, commands are rendered/logged but not executed against cluster state.
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
