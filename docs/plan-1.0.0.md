# Plan 1.0.0 — stable contract

**Status:** **Gates complete on `develop`** (2026-07-15). After **0.9.x** (shipped through **v0.9.2**). **#33**, **#42**, **#34**, **#32** done — validate locally, then tag path (**no push until OK**).

**Motivation:** Promote kzero from “mature 0.x operator CLI” to a **1.0** promise: YAML **`schema_version`**, executor defaults, and CLI exit behavior stable enough for long-lived wrappers and bastion automation. Bastion-first posture from **0.9.x** stays the default story ([deployment-models.md](deployment-models.md)).

For shipped behavior see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md). Prior band: [plan-0.9.x.md](plan-0.9.x.md) (**Done**).

**Out of band for 1.0.0:** **#29**, hook interpreter, resume-from-step — see [plan-1.1.0.md](plan-1.1.0.md). **#55** log upload — optional stretch, not a **1.0.0** tag gate.

---

## Priority tiers

| Tier | Items | Notes |
|------|-------|--------|
| **Required for v1.0.0** | **#32**, **#33**, **#34**, **#42** | All four must land before the major tag. |
| **Optional stretch** | **#55** | Only if it does not delay the major. |
| **Explicitly deferred** | **#29**, #56–#57 | [plan-1.1.0.md](plan-1.1.0.md) |

---

## Roadmap items

| # | Item | Summary |
|---|------|---------|
| **32** | **Default native execution** | **Done** — omitted → **`native`**; **`shell`** opt-in. Migration in CHANGELOG; sample/schema/SPEC/README. |
| **33** | **PVC / StatefulSet data strategy** | **Done** — [pvc-statefulset-data-strategy.md](examples/pvc-statefulset-data-strategy.md); SPEC/README links. No new step kind. |
| **34** | **kind / envtest in CI** | **Done** — `testing/kind/` + job **`integration-kind`** (budget 20m, flake policy in README). envtest skipped (fake-client + smoke already cover API unit path). |
| **42** | **Exit code taxonomy** | **Done** — `internal/exitcode` codes **0–4**; wraps at unambiguous CLI/engine sites; SPEC §5 + man. Pattern: groot `internal/cmd/exitcode.go`. |
| **55** | *(Optional)* **Post-pipeline log upload** | After a run, push log file to S3/GCS/SFTP; hooks/selfhosted remain default. |

---

## Suggested PR order

| PR | Item | Why this order |
|----|------|----------------|
| PR1 | **#33** PVC/data patterns docs | **Done** (cookbook + SPEC/README). |
| PR2 | **#42** Exit codes | **Done** (`internal/exitcode` + wraps). |
| PR3 | **#34** kind/envtest CI | **Done** (`testing/kind/` + **`integration-kind`**). |
| PR4 | **#32** Default **`native`** | **Done** (breaking default + migration + sample/SPEC). |
| PR5 | Tag **`v1.0.0`** | Full [release checklist](../.cursor/rules/release-tests.mdc): VERSION, CHANGELOG, man `.TH`, BSD ports, demo.gif, **`make release-check`**. **No push until user validates.** |

Alternate: PR4 before PR3 if pilot demand is “native-first now” and CI kind can follow in **1.0.1**.

Each PR: **`make lint`**, **`make test`**, **`make cover-check`** (and **`go test -race`** awareness for CLI tests that touch `newRootCmd`).

---

## Success criteria (v1.0.0 tag)

| # | Criterion | Verify |
|---|-----------|--------|
| 1 | Omitted **`run.execution`** → **`native`**; **`shell`** documented as opt-in | **Met** (#32) |
| 2 | PVC/StatefulSet data patterns published and linked from SPEC/README | **Met** (#33) |
| 3 | CI runs kind or envtest job with flake policy + budget | **Met** (#34; confirm **`integration-kind`** green after push) |
| 4 | Documented exit codes for unambiguous failure classes | **Met** (#42) |
| 5 | Migration notes for **0.9.x → 1.0.0** (especially #32) | CHANGELOG |
| 6 | **`make release-check`** green; coverage ≥ 80% | CI |
| 7 | No silent change to in-cluster auth / empty kubeconfig behavior | SPEC note + tests |

---

## Non-goals

- Promoting in-cluster Job as primary **`reset`** path
- Prometheus / OTel / chaos-mesh in product repo
- Built-in **`job`** / **`cronjob`** steps (**#29**)
- Breaking **`schema_version: "1.0"`** key set without a schema bump plan

---

## Relationship to operators

- **kzero** (this repo): CLI, engine, CI, packaging.
- **kzero-selfhosted**: bastion/cron/kind e2e examples — update when #32/#33/#34 change recommended defaults or CI fixtures.

**Last reviewed:** 2026-07-15 (all required gates **#32–#34**, **#42** on develop; no push until user validates)
