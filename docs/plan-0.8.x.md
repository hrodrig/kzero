# Plan 0.8.x — pipeline resilience, API watchdog, and notify delivery

**Status:** **Planned** — target band **`v0.8.0`** (first tag of the band).  
**Motivation:** production maintenance incident — control-plane path failed mid-run; ~15 minutes later the process detected API loss but **did not alert**; ~30 minutes later **total bastion network loss**; recovery ~4 hours later; **local logs** were the only evidence.

This document captures operator learnings as **product requirements** for **kzero**. It does not name legacy tooling or tenant stacks. For current shipped behavior see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md).

**Operator mitigations today (pre-0.8):** timestamped logs via wrappers ([kzero-selfhosted `run-kzero`](https://github.com/hrodrig/kzero-selfhosted/blob/develop/run/examples/full-reset-example/run-kzero)), short **`run.operation_timeout`**, **`on-error`** hooks, external watchdog — see [pipeline-network-loss.md](examples/pipeline-network-loss.md) (added with this plan).

---

## Incident timeline (requirements source)

| Phase | Observation | Requirement for kzero |
|-------|-------------|------------------------|
| **A — mid-run** | Long **`helm --wait`** / rollout wait while API path degrades | Fail fast; do not block silently past configured ceilings |
| **B — ~15 min** | Process knows API is unreachable; **no Slack/webhook** | **`pipeline.error`** (or dedicated event) must fire when API loss is detected, not only when a step returns |
| **B — same window** | Bastion may still reach Slack (API path ≠ notify path) | Notify attempts must **log `[ERR]` on failure** — never silent discard |
| **C — ~30 min** | Total network loss on bastion | Engine cannot fix; **local log file** must record last step, last API OK, timestamps |
| **D — +4 h** | Cluster reachable again | Operator reconstructs story from logs; optional **resume** is out of scope |

**Non-goals for 0.8.x:** automatic pipeline resume after partition; multi-bastion HA; built-in log shipping.

---

## Why 0.8.x (after 0.7.x)

**0.7.x** closed distroless primitives (Helm SDK, **`pvc`**, **`exec`**, probe native). **0.8.x** closes **operator safety during long live resets** on real bastions:

1. **API watchdog** during live pipelines (not only preflight at phase start)  
2. **Notify delivery visibility** (failed POST ≠ success)  
3. **Phase-boundary preflight** on **`reset`** (down → up)  
4. **Audit-friendly progress lines** on long waits  
5. **Documented** partial/total network loss patterns

**1.0.0** items (**#32–#34**, **#29**) stay in [ROADMAP.md](../ROADMAP.md).

---

## Roadmap items (0.8.x band)

| # | Item | Summary |
|---|------|---------|
| **35** | **Notify delivery visibility** | `notify.Dispatch` failures surface as **`[ERR]`** log lines (redacted URLs). Pipeline error still proceeds; operators see “alert failed to send”. Optional **`notify.require_delivery`** (fail pipeline if error notify POST fails) — default **false**. |
| **36** | **API watchdog during live pipelines** | Configurable periodic **`Discovery().ServerVersion()`** (or equivalent) while a live **`down`/`up`/`reset`** runs. On **N** consecutive failures or **M** elapsed unreachable → fail-fast **`PipelineError`**, **`pipeline.error`** notify, **`on-error`** hook. |
| **37** | **Preflight between `reset` phases** | After successful **`down`**, before **`up`**, run the same preflight as phase start. Surfaces API loss in the gap between destructive and restore work. |
| **38** | **Long-step progress logging** | During rollout wait / Helm wait / retry backoff, emit throttled **`[INF]`** lines: step ref, elapsed, last API check OK (watchdog). Aids post-mortem when notify is impossible. |
| **39** | **Config: `run.api_watchdog`** | YAML + **`KZERO_RUN_API_WATCHDOG_*`** env overrides: **`enabled`**, **`interval`**, **`fail_after`** (duration or consecutive failures). Defaults tuned for prod resets (e.g. interval **60s**, fail after **3** failures or **5m** unreachable). |
| **40** | **Operator docs + selfhosted cookbook** | [examples/pipeline-network-loss.md](examples/pipeline-network-loss.md); link from SPEC notify/preflight; selfhosted automation doc for wrappers and external watchdog. |

Optional stretch (same band, if small):

| # | Item | Summary |
|---|------|---------|
| **41** | **`pipeline.stalled` notify event** | Distinct webhook event when watchdog fires (vs step failure). Slack title e.g. **`kzero stalled`**; **`kzero notify test --event stalled`**. |

---

## Success criteria (0.8.0 tag)

| # | Criterion | Verify |
|---|-----------|--------|
| 1 | Simulated API unreachable mid-pipeline (fake client or envtest) triggers fail-fast within **`fail_after`** | Unit/integration test |
| 2 | **`pipeline.error`** dispatched on watchdog failure when notify enabled | Test + manual **`notify test --event error`** parity |
| 3 | Failed Slack POST logs **`[ERR]`** with redacted URL; exit code unchanged unless **`notify.require_delivery: true`** | Test |
| 4 | **`reset`**: preflight runs after **`down`**, before **`up`** | Engine test |
| 5 | Long wait emits at least one progress **`[INF]`** per **`interval`** | Test or documented manual |
| 6 | SPEC + README + [pipeline-network-loss.md](examples/pipeline-network-loss.md) describe two-phase outage pattern | Doc review |
| 7 | **`make release-check`** green; coverage ≥ 80% | CI |

---

## Implementation order (PR-sized slices)

Merge to **`develop`** in order:

| PR | Item | Roadmap | Notes |
|----|------|---------|--------|
| PR1 | Notify **`[ERR]`** on dispatch failure; remove silent `_ = Dispatch` in CLI/engine | **#35** | Small; high value |
| PR2 | **`run.api_watchdog`** config parse + env binding + analyze deferred summary | **#39** | Schema first |
| PR3 | Watchdog goroutine in engine live runs; cancel step context on trip | **#36** | Core behavior |
| PR4 | Reset phase-boundary preflight | **#37** | One function call in **`RunReset`** |
| PR5 | Throttled progress logs on native wait / helm wait | **#38** | Avoid log spam |
| PR6 | Optional **`pipeline.stalled`** event + **`notify test --event stalled`** | **#41** | Optional |
| PR7 | Docs, SPEC, selfhosted link, ROADMAP tick, **`VERSION` 0.8.0**, tag **`v0.8.0`** | **#40** | Band close |

Each PR: **`make lint`**, **`make test`**, **`make cover-check`**.

---

## Config sketch (SPEC draft — not implemented)

```yaml
run:
  mode: live
  timeout: 45m
  operation_timeout: 10m          # per-step ceiling; keep prod resets aggressive
  api_watchdog:
    enabled: true
    interval: 60s                   # check API between steps and during long waits
    fail_after: 5m                  # or fail_after_failures: 3 (exact shape TBD in SPEC PR)
notify:
  on_error: true
  require_delivery: false           # if true, pipeline exit non-zero when error-notify POST fails
```

Env (illustrative): **`KZERO_RUN_API_WATCHDOG_ENABLED`**, **`KZERO_RUN_API_WATCHDOG_INTERVAL`**, **`KZERO_RUN_API_WATCHDOG_FAIL_AFTER`**, **`KZERO_NOTIFY_REQUIRE_DELIVERY`**.

---

## Engine behavior (target)

```mermaid
sequenceDiagram
  participant Op as Operator / bastion
  participant K as kzero engine
  participant API as Kubernetes API
  participant N as notify channels

  Op->>K: kzero reset (live)
  K->>N: pipeline.start
  K->>API: preflight OK
  loop each step
    K->>API: step action / wait
    Note over K,API: watchdog tick every interval
    API--xK: unreachable (phase B)
    K->>N: pipeline.error (API watchdog)
    Note over N: ERR log if POST fails
    K->>Op: exit non-zero; log file has last step + timestamp
  end
```

**Today (0.7.3):** preflight only at phase start; notify errors discarded; no mid-pipeline API probe.

---

## Testing strategy

| Layer | Approach |
|-------|----------|
| **Unit** | Fake discovery client returns error after N calls; assert watchdog trips |
| **Unit** | Notify server returns 500; assert **`[ERR]`** line |
| **Engine** | **`RunReset`** calls preflight twice on success path (mock runner) |
| **Manual** | Lab cluster: **`kubectl`** proxy stop / firewall rule; confirm alert within **`fail_after`** |
| **Out of scope** | Chaos test for total bastion egress loss (operator doc only) |

---

## selfhosted alignment (not in kzero binary)

| Deliverable | Repo |
|-------------|------|
| [examples/pipeline-network-loss.md](examples/pipeline-network-loss.md) | **kzero** |
| Link from **`run/docs/automation-and-pipelines.md`** | **kzero-selfhosted** |
| **`run-kzero`** log header already includes target block | **kzero-selfhosted** (existing) |

---

## Relationship to 1.0.0

**0.8.x** does not block **1.0.0**, but operators running production **`reset`** should prefer **≥ 0.8.0** once tagged. **#32–#34** remain **1.0.0** scope.

---

## Cadence

Single band-close tag **`v0.8.0`** when criteria **1–7** pass (same pattern as **0.7.2**). Patch releases (**0.8.1+**) only if watchdog defaults need tuning after field feedback.

**Last reviewed:** 2026-06-12
