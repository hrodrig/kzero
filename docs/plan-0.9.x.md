# Plan 0.9.x — bastion-first hardening

**Status:** **Planned** — band not yet tagged.

**Motivation:** External audits (2026-06) and operator learnings from **0.8.x** agree: kzero should **orchestrate** clusters from **out-of-band** hosts (bastion, management VM, cron). In-cluster Job is **supported** but must not be marketed as the primary **`reset`** path when API or network reliability is uncertain. See [deployment-models.md](deployment-models.md).

This document captures **0.9.x** requirements. For shipped behavior see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md).

**Relationship to 1.0.0:** **0.9.x** closes operational gaps and documentation; **1.0.0** (**#32–#34**, **#42**) remains the stable-contract semver band.

---

## Priority tiers (0.9.0 vs 0.9.1)

| Tier | Items | Target tag |
|------|-------|------------|
| **Required for 0.9.0** | **#43**, **#44**, **#48** (docs remainder) | **v0.9.0** |
| **Strongly recommended** | **#45** (E2E smoke), **#46** (watchdog tests), **#47** (SPEC contract index) | **v0.9.0** if schedule allows; else early **0.9.1** |
| **Stretch / patch** | **#49** (`doctor` / `validate --strict`), **#50** (retry jitter), **#51** (JSON Schema) | **0.9.1+** |

**#49** (`kzero doctor` or **`validate --strict`**) is high operator value (kubeconfig reachability, RBAC hints for declared steps, binary presence for **`shell`**) — prioritize in **0.9.1** if it does not fit **0.9.0**.

---

## Roadmap items (0.9.x band)

| # | Item | Summary |
|---|------|---------|
| **43** | **`notify.require_delivery`** | When **`true`**, pipeline exits non-zero if **`pipeline.error`** / watchdog notify POST fails (**#35** deferred part). |
| **44** | **Graceful shutdown** | SIGTERM/SIGINT → cancel context, flush log line with phase/step; important for bastion **cron** / **systemd**. |
| **45** | **E2E smoke in CI** | Minimal pipeline via **kind** or **kzero-selfhosted** fixture; validates binary + config on PR — not “kzero saves cluster from inside”. |
| **46** | **Watchdog test coverage** | API unreachable during long wait (extends **0.8.0** criterion #1). |
| **47** | **SPEC contract index** | Table: implemented vs deferred vs experimental; aligns with **`analyze`** Deferred warnings. |
| **48** | **Docs polish** | [deployment-models.md](deployment-models.md) (matrix, warnings, docker bastion — band kickoff); What's new **0.8.x**; **`cosign verify`** in README; README length trim. |
| **49** | *(Stretch)* **`validate --strict` / `doctor`** | Config + API ping + binary presence + RBAC hints for pipeline steps. |
| **50** | *(Stretch)* **Retry jitter** | Randomize backoff delay (existing **0.5.2** retry). |
| **51** | *(Stretch)* **JSON Schema** | Editor autocomplete for **`kzero.yaml`**. |

---

## Success criteria (0.9.0 tag)

| # | Criterion | Verify |
|---|-----------|--------|
| 1 | [deployment-models.md](deployment-models.md) linked from README, SPEC, ROADMAP | Doc review |
| 2 | **`require_delivery: true`** fails pipeline when error notify POST fails | Unit/integration test |
| 3 | SIGTERM during live step cancels within bounded time; last step logged | Test |
| 4 | CI runs at least one E2E or kind smoke job | Workflow green |
| 5 | **`make release-check`** green; coverage ≥ 80% | CI |
| 6 | **No regressions** for existing **in-cluster** configs (`InClusterConfig`, empty **`run.kubeconfig`**, native Job paths) — behavior unchanged unless explicitly documented | Test + SPEC note |

---

## Implementation order (PR-sized slices)

Merge to **`develop`** in order:

| PR | Item | Roadmap | Priority |
|----|------|---------|----------|
| PR1 | [deployment-models.md](deployment-models.md) + ROADMAP/SPEC/README cross-links | **#48** (partial) | **Done** (matrix, warnings, docker bastion, decision flow) |
| PR2 | Graceful shutdown (signal → context cancel) | **#44** | **Required** — implement early; affects all live runs |
| PR3 | **`notify.require_delivery`** engine wiring + tests | **#43** | **Required** |
| PR4 | E2E smoke job in CI | **#45** | Strongly recommended |
| PR5 | Watchdog mid-wait tests | **#46** | Strongly recommended |
| PR6 | SPEC contract vs deferred index | **#47** | Strongly recommended |
| PR7 | Docs polish + **`VERSION` 0.9.0** tag | **#48** | Required |

Stretch after **v0.9.0**: **#49–#51** in **0.9.1** patches.

Each PR: **`make lint`**, **`make test`**, **`make cover-check`**.

---

## Non-goals

- In-cluster Job as recommended **`reset`** orchestrator
- Prometheus metrics, OpenTelemetry, chaos-mesh in product repo
- Replacing **1.0.0** default-native or exit-code taxonomy work
- Breaking in-cluster auth or native Job execution without a documented migration

**Last reviewed:** 2026-06-29
