# Long-running live pipelines and network loss

Operator patterns when a **bastion** runs **`kzero down`**, **`up`**, or **`reset`** against a remote API server (AKS, RKE, on-prem, etc.). Complements [notifications.md](notifications.md) and [SPECIFICATIONS.md](../SPECIFICATIONS.md) preflight/notify sections.

**Planned engine improvements:** [plan-0.8.x.md](../plan-0.8.x.md) (**API watchdog**, notify delivery visibility, reset phase-boundary preflight).

---

## Two-phase outage (typical)

```text
Phase 1 — API path fails (~15–30 min)
  Bastion may still reach Slack/PagerDuty.
  Risk: process blocked on helm --wait / rollout status; no alert sent.

Phase 2 — Total bastion network loss (~30 min+)
  No API, no notify, no SSH recovery.
  Evidence: local log file only.
```

Design maintenance assuming **both** phases can happen.

---

## What kzero does today (v0.7.3)

| Mechanism | Behavior |
|-----------|----------|
| **`pipeline.start` / `pipeline.error` / success** | Slack/webhook on live pipeline start, step failure, success |
| **Preflight** | **`ServerVersion`** once at start of each **`down`** / **`up`** phase — **not** continuous |
| **Timeouts** | **`run.timeout`** (whole pipeline), **`run.operation_timeout`** (per operation), Helm/step **`timeout`** |
| **Logs** | Timestamped **`[INF|WRN|ERR]`** on stdout; wrappers can tee to **`.logs/`** |
| **Gap** | Notify POST failures are not surfaced; no mid-pipeline API watchdog |

---

## Operator mitigations (until v0.8.0)

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
- Store under **`.logs/kzero-<cmd>-<timestamp>.log`** on the bastion disk
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
- Log mtime: no new lines in **`.logs/kzero-reset-*.log`** for **N** minutes
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

- [plan-0.8.x.md](../plan-0.8.x.md) — **0.8.x** API watchdog and notify delivery
- [waiting-between-pipeline-steps.md](waiting-between-pipeline-steps.md) — Helm/rollout waits
- [kzero-selfhosted automation](https://github.com/hrodrig/kzero-selfhosted/blob/develop/run/docs/automation-and-pipelines.md)
