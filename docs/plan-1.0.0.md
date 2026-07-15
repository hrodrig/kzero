# Plan 1.0.0 — stable contract

**Status:** **Draft** — planning only (2026-07-12). Implement on **`develop`** after **0.9.x** (shipped through **v0.9.2**).

**Motivation:** Promote kzero from “mature 0.x operator CLI” to a **1.0** promise: YAML **`schema_version`**, executor defaults, and CLI exit behavior stable enough for long-lived wrappers and bastion automation. Bastion-first posture from **0.9.x** stays the default story ([deployment-models.md](deployment-models.md)).

For shipped behavior see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md). Prior band: [plan-0.9.x.md](plan-0.9.x.md) (**Done**).

**Out of band for 1.0.0:** **#29** (`job` / `cronjob` / CRD patch) — **post-1.x** (`custom:` until then). **#55** post-pipeline log upload — optional stretch, not a tag gate.

---

## Priority tiers

| Tier | Items | Notes |
|------|-------|--------|
| **Required for v1.0.0** | **#32**, **#33**, **#34**, **#42** | All four must land before the major tag. |
| **Optional stretch** | **#55** | Only if it does not delay the major. |
| **Explicitly deferred** | **#29** | post-1.x |

---

## Roadmap items

| # | Item | Summary |
|---|------|---------|
| **32** | **Default native execution** | When **`run.execution`** is omitted, use **`native`** (not **`shell`**). Operators opt into **`shell`** explicitly. **Breaking default** — migration note in CHANGELOG/SPEC/README; sample configs and analyze Deferred/warnings as needed. |
| **33** | **PVC / StatefulSet data strategy** | Document pipeline **patterns** (snapshot, wipe, init-job) beyond core **`pvc`** delete — cookbooks under **`docs/examples/`**, links from SPEC; no new mandatory step kind unless already justified. |
| **34** | **kind / envtest in CI** | Product-repo integration tests with **documented flake policy** and runtime budget. Builds on smoke (**#45**); may reuse **kzero-selfhosted** kind fixtures where practical. |
| **42** | **Exit code taxonomy** | Map subsystem failures to stable non-zero codes (config, Kubernetes client/API, executor abort, notify delivery, …). Pattern: [groot exitcode](https://github.com/hrodrig/groot/blob/main/pkg/cmd/exitcode.go). Existing wrappers that only check “non-zero” stay compatible; document new codes. |
| **55** | *(Optional)* **Post-pipeline log upload** | After a run, push log file to S3/GCS/SFTP; hooks/selfhosted remain default. |

---

## Suggested PR order

| PR | Item | Why this order |
|----|------|----------------|
| PR1 | **#33** PVC/data patterns docs | Low risk; clarifies operator contract before default/native change. |
| PR2 | **#42** Exit codes | Wrapper-facing contract; independent of executor default. |
| PR3 | **#34** kind/envtest CI | Confidence gate before flipping defaults. |
| PR4 | **#32** Default **`native`** | Last — breaking default + migration + sample/SPEC sync. |
| PR5 | Tag **`v1.0.0`** | Full [release checklist](../.cursor/rules/release-tests.mdc): VERSION, CHANGELOG, man `.TH`, BSD ports, demo.gif, **`make release-check`**. |

Alternate: PR4 before PR3 if pilot demand is “native-first now” and CI kind can follow in **1.0.1**.

Each PR: **`make lint`**, **`make test`**, **`make cover-check`** (and **`go test -race`** awareness for CLI tests that touch `newRootCmd`).

---

## Success criteria (v1.0.0 tag)

| # | Criterion | Verify |
|---|-----------|--------|
| 1 | Omitted **`run.execution`** → **`native`**; **`shell`** documented as opt-in | Unit/config tests + SPEC |
| 2 | PVC/StatefulSet data patterns published and linked from SPEC/README | Doc review |
| 3 | CI runs kind or envtest job with flake policy + budget | Workflow green |
| 4 | Documented exit codes for unambiguous failure classes | Tests + SPEC/man |
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

**Last reviewed:** 2026-07-12 (draft after **v0.9.2**)
