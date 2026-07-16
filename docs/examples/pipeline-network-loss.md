# Long-running live pipelines and network loss

Operator patterns when a **bastion** runs **`kzero down`**, **`up`**, or **`reset`** against a remote API server (AKS, RKE, on-prem, etc.). Complements [notifications.md](notifications.md) and [SPECIFICATIONS.md](../../SPECIFICATIONS.md) preflight/notify sections.

**Engine features (shipped v0.8.0):** [plan-0.8.x.md](../plan-0.8.x.md) — **API watchdog**, notify delivery **`[ERR]`** logs, reset phase-boundary preflight, throttled progress logs, **`pipeline.stalled`** event.

---

## Two-phase outage (typical)

```text
Phase 1 — API path fails (~15–30 min)
  Bastion may still reach Slack/PagerDuty.
  Risk: process blocked on helm --wait / rollout status; watchdog should trip
        and dispatch pipeline.stalled / pipeline.error when configured.

Phase 2 — Total bastion network loss (~30 min+)
  No API, no notify, no SSH recovery.
  Evidence: local log file only.
```

Design maintenance assuming **both** phases can happen.

---

## What kzero does today (v0.8.0)

| Mechanism | Behavior |
|-----------|----------|
| **`pipeline.start` / `pipeline.error` / success** | Slack/webhook on live pipeline start, step failure, success |
| **`pipeline.stalled`** | Distinct notify event when **`run.api_watchdog`** trips mid-pipeline (**v0.8.0**) |
| **Preflight** | **`ServerVersion`** at start of each **`down`** / **`up`** phase; **re-run after `down` before `up`** on **`reset`** (**#37**) |
| **`run.api_watchdog`** | Periodic API reachability during live runs; cancels stuck step and dispatches **`pipeline.stalled`** when **`fail_after`** exceeded (**#36**) |
| **Notify dispatch failures** | Failed POSTs log **`[ERR]`** with redacted URLs (**#35**); pipeline exits non-zero when **`notify.require_delivery: true`** and **`pipeline.error`** / **`pipeline.stalled`** POST fails (**#43**, **v0.9.x**) |
| **Long waits** | Throttled **`[INF]`** progress lines every 30s during rollout/Helm waits (**#38**) |
| **Timeouts** | **`run.timeout`** (whole pipeline), **`run.operation_timeout`** (per operation), Helm/step **`timeout`** |
| **Per-step retry** | Live mode retries transient API/network errors (incl. **`connection lost`** / **`http2: client connection lost`** since **v1.0.1**) |
| **Logs** | Timestamped **`[INF|WRN|ERR]`** on stdout; wrappers can tee to **`.logs/`** |
| **Remaining gap** | Total bastion network loss still blocks all notify paths; no automatic pipeline resume; klog mid-stream noise may appear without failing the step |

### Example: enable API watchdog

```yaml
run:
  mode: live
  timeout: 45m
  operation_timeout: 8m
  api_watchdog:
    enabled: true
    interval: 60s
    fail_after: 5m
```

Tune **`interval`** / **`fail_after`** per cluster; see [SPECIFICATIONS.md](../../SPECIFICATIONS.md) → **`run.api_watchdog`**.

---

## Supplemental operator mitigations

### 1. Aggressive timeouts in production YAML

```yaml
run:
  mode: live
  timeout: 45m
  operation_timeout: 8m    # fail a stuck scale/helm call before "15 min silence"
```

Tune per cluster; long **`helm --wait`** steps may need explicit step **`timeout`**.

### 2. Timestamped log file (mandatory for prod resets)

Use a wrapper that records **`kzero target`** and full stdout/stderr:

- [kzero-selfhosted `run-kzero`](https://github.com/hrodrig/kzero-selfhosted/blob/develop/run/examples/full-reset-example/run-kzero) — pattern reference
- Store under **`.logs/kzero-<cmd>-<cluster-slug>-<timestamp>.log`** on the bastion disk (`cluster-slug` from `kzero target --output slug`, e.g. `develop-cluster`)
- Treat logs as **primary evidence** when notify and API are both gone

### 3. `on-error` hook with redundant alert

```yaml
hooks:
  on-error: ./hooks/alert-on-failure.sh
```

Script runs on the bastion (same host as kzero). Example actions: `curl` secondary webhook, `mail`, write flag file for external monitor. Must not depend on Kubernetes API.

### 4. External watchdog (recommended)

Separate machine or SaaS monitor:

- Process check: **`kzero`** PID still running after **N** minutes
- Log mtime: no new lines in **`.logs/kzero-reset-*-<timestamp>.log`** for **N** minutes (glob by cluster slug when multiple targets share a bastion)
- Alert even if kzero never reaches **`pipeline.error`**

### 5. Notify test before live reset

```bash
kzero notify test --config prod.yaml --event error
```

Confirms Slack/webhook path **before** destructive work — does not guarantee delivery after network partition.

### 6. Separate network paths

Where possible: API via private link/VPN; notify via public HTTPS. Phase 1 alerts may work when Phase 2 API is dead.

---

## After an outage

1. Collect **`.logs/`** from bastion (even if process died).
2. Find last **`[INF]`** step line and **`Kubernetes target:`** block.
3. Compare with **`kzero analyze`** once API returns — drift shows partial reset.
4. Do **not** blindly re-run **`reset` live`** — assess cluster state first.

---

## See also

- [plan-0.8.x.md](../plan-0.8.x.md) — **0.8.x** incident learnings and success criteria (shipped **v0.8.0**)
- [waiting-between-pipeline-steps.md](waiting-between-pipeline-steps.md) — Helm/rollout waits
- [kzero-selfhosted automation](https://github.com/hrodrig/kzero-selfhosted/blob/develop/run/docs/automation-and-pipelines.md)
