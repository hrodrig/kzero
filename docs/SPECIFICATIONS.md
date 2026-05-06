# kzero v1 Specifications (TDD Baseline)

## 1. Purpose

`kzero` is a Go CLI that executes declarative, ordered Kubernetes workload pipelines.
Version 1 focuses on workload orchestration only (Deployments, StatefulSets, Helm release steps, and custom scripts).

This document is the source of truth for behavior and test expectations.

## 2. Scope (v1)

### In scope
- Declarative config from YAML (`kzero.yaml` or `--config` path).
- Ordered phase execution: `down`, `up`, and `reset` (`down` then `up`).
- Phase lifecycle hooks: `pre-down`, `post-down`, `pre-up`, `post-up`, `on-error`.
- Pipeline step types:
  - compact resource refs (`<kind>.<namespace>/<name>`)
  - Helm release refs (`release.<namespace>/<name>`)
  - custom script steps (`{ custom: ./path/script.sh }`)
- Run modes: `dry-run` and `live` (live can start as minimal implementation).
- Retry configuration for transient command failures.

### Out of scope
- Node lifecycle operations (`drain`, `cordon`, node deletion).
- Cloud/provider orchestration (`az`, `aws`, `gcloud`, Talos/k3s reset flows).
- Automatic nodepool scaling.

### Engine design principles

kzero stays **generic** and **configuration-driven**: the engine interprets validated YAML; it does not embed environment-specific playbooks.

**Implementation patterns to prefer:**
- Phased workflows with explicit ordering (`pre-*`, pipeline steps, `post-*`).
- Readiness waits and per-step / global timeouts.
- Bounded parallelism for independent steps (worker pool; cap via config).
- Safe notifications: optional channels, redact or mask secrets in logs, include run mode and correlation metadata (e.g. `client.id`, cluster name).
- kubectl and Helm execution with explicit timeouts, structured logs, and retry policy for transient failures.

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
- `notify` (optional; channel handling may be no-op in early v1)
- `retry.attempts`, `retry.delay`
- `run.kubeconfig`, `run.mode`, `run.timeout`, `run.worker_concurrency`, `run.operation_timeout`

## Pipeline syntax
- String step: `<kind>.<namespace>/<name>`
  - Examples:
    - `deployment.argocd/argocd-server`
    - `statefulset.database/postgresql`
    - `release.monitoring/kube-prometheus-stack`
- Map step with one key:
  - `custom: ./hooks/example-custom.sh`
  - Up step options:
    - `statefulset.database/postgresql: { replicas: 3, wait_for_ready: true, timeout: 10m }`

Any unknown step format must fail validation.

## 4. Command Behavior

## `kzero analyze`
- Validates config and prints normalized execution plan.
- Must not mutate cluster state.
- Exit code `0` on valid config; non-zero on invalid config.

## `kzero down`
Execution order:
1. `hooks.pre-down` (if set)
2. `pipelines.down` in strict list order
3. `hooks.post-down` (only if step execution succeeded)

## `kzero up`
Execution order:
1. `hooks.pre-up` (if set)
2. `pipelines.up` in strict list order
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
- For `reset`, failure in `down` skips `up`.

## 6. Dry-Run Policy

- In `dry-run`, commands are rendered/logged but not executed against cluster state.
- Hook and custom script execution policy for dry-run (v1):
  - default: do not execute scripts; print planned invocation.
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

## B. Phase orchestration
- `TestDownOrder_PreStepsPost`
  - `pre-down` -> steps in order -> `post-down`.
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

## 9. Definition of Done (v1)

- All tests above implemented and passing via `make test`.
- `make lint` passes.
- `configs/kzero.sample.yml` matches the supported schema and examples.
- Command help text documents v1 scope and non-goals.
