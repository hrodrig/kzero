# Plan 1.1.0 — post-1.0 operator ergonomics (bounded)

**Status:** **Active** — priority 2026-08-01: **#59** / **#29** / **#58** done on develop; **#29b** CRD patch follow-up; **#55** parked; **#57** deferred (complexity). **#56** shipped in **v1.0.2**.

**Motivation:** Close the highest-value gaps that **do not** turn kzero into a daemon, multi-cluster control plane, or secret broker. Stay **bastion-first**, **config-first**, sequential by default.

For shipped behavior see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md).

---

## In scope (tag gates)

| # / ID | Item | Summary |
|--------|------|---------|
| **56** | **Configurable hook interpreter** | **Shipped in v1.0.2.** Opt-in **`command.shell`** for hook / **`custom:`** / shell **`release.*`** scripts (default **`/bin/sh`**). Fixes Ubuntu **dash** vs bashisms without magic shebang. SPEC: [Hook and script interpreter](../SPECIFICATIONS.md#hook-and-script-interpreter-commandshell). |
| **59** | **Helm SDK → v4.2.3+** | **Done on develop.** Migrated to **`helm.sh/helm/v4` v4.2.3**; dropped **GO-2026-5932** ignores; **`k8s.io/*` v0.36.x**. See [Helm v4 spike](#helm-v4-spike-2026-07-16). |
| **29** | **`job` / `cronjob` + safe CRD patch** | **MVP done on develop:** Job lifecycle + CronJob suspend/resume (native). **#29b:** narrow patch/scale for CRDs (dynamic client) — until then **`custom:`**. |

## Deferred (not in immediate queue)

| # / ID | Item | Notes |
|--------|------|--------|
| **57** | **Resume / restart from step** | Deferred (complexity). Phase A / Phase B remain documented for a later band if needed. |

## Stretch (not required for v1.1.0)

| # / ID | Item | Notes |
|--------|------|--------|
| **58** | **`kzero diff`** | **Done (MVP)** on develop — `--phase up|down`; replicas / suspend / presence. Cookbook [examples/diff.md](examples/diff.md). |
| **55** | **Post-pipeline log upload** | **Parked** unless operators insist; wrappers / selfhosted remain default. |

---

## Suggested PR order

| PR | Item | Why |
|----|------|-----|
| PR0 | **#56** hook interpreter | **Done (v1.0.2)** — `command.shell`. |
| PR1 | **#59** Helm SDK v4 | **Done on develop** — clears govulncheck ignore; bumps `k8s.io/*`. |
| PR2 | **#29** job / cronjob MVP | **Done** on develop (CRD patch → **#29b**). |
| PR3 | **#58** `kzero diff` | **Done** on develop. |
| PR4 | Tag **`v1.1.0`** | Release checklist ([release-tests](../.cursor/rules/release-tests.mdc)). |

**Deferred:** **#57** (resume-from-step). **Parked:** **#55** (log upload).

---

## Helm v4 spike (2026-07-16)

**Question:** Can kzero move from **`helm.sh/helm/v3 v3.21.0`** to **`helm.sh/helm/v4 v4.2.3`** in the **1.1.0** band?

| Check | Result |
|-------|--------|
| **GO-2026-5932** | **Fixed in Helm v4** via [PR #31320](https://github.com/helm/helm/pull/31320) (`ProtonMail/go-crypto`). **Not** fixed on Helm **v3.21.x** (still imports `golang.org/x/crypto/openpgp`). |
| **Helm v3 support** | Support mode: bugfixes ended **2026-07-08**; **security fixes until 2026-11-11**. Staying on v3 past that is risk. |
| **kzero surface** | Contained: `internal/executor/helm_sdk.go`, `registry_auth.go`, helm tests / shell path unchanged. Uses `action` Upgrade/Install/Uninstall/History, `chart/loader`, `cli`, `registry`, `values` — maps to v4 SDK packages. |
| **Import churn** | Module **`helm.sh/helm/v4`**. Chart types move toward **`pkg/chart/v2`** (path break; APIs largely familiar). Logging hooks may expect **`slog.Handler`** instead of `log.Printf`-style funcs — verify at compile. |
| **k8s clients** | Helm **v4.2.3** pulls **`k8s.io/client-go v0.36.1`** (kzero today **v0.35.1**) — expect coordinated bump of **`k8s.io/api` / `apimachinery` / `client-go`**. |
| **Operator charts** | Chart format remains compatible per Helm docs; no YAML schema change for kzero pipelines. Host **`helm` CLI v3** still fine for **`run.execution: shell`**. |
| **Effort** | Medium: one focused PR (imports + compile fix + unit tests + `make release-check` / Grype). Remove **GO-2026-5932** from `.govulncheck-ignore.yaml` when graph is clean. |
| **Verdict** | **Viable.** Ship as **PR1** in 1.1 queue (**#59** before **#29**). |

**Out of spike scope:** rewriting release scripts, requiring host Helm v4 CLI, Helm plugins / Wasm.

---

## Success criteria (v1.1.0 tag)

| # | Criterion | Verify |
|---|-----------|--------|
| 1 | Default interpreter still **`/bin/sh`**; opt-in bash (or path) documented + tested | Unit + SPEC (**#56** done) |
| 2 | Helm SDK on **`helm.sh/helm/v4`**; **GO-2026-5932** cleared from ignore lists when graph is clean | `make security` / Grype (**#59**) |
| 3 | At least one of **`job`** / **`cronjob`** (or documented patch step) usable in live/dry-run | **Met** — tests + SPEC (**#29** MVP) |
| 4 | **`make release-check`** green | CI |

(**#57** resume-from-step deferred — not a 1.1 tag gate.)

---

## Explicitly parked (not 1.1)

Do **not** pull into this band without a new plan:

- Parallel step waves (revisit of removed **`worker_concurrency`**)
- Built-in webhook triggers, **`kzero schedule`**, SSH bastion tunnel
- Multi-cluster single YAML, Slack approval gates
- Vault / cloud secret-manager plugins
- Prometheus / OTel (beyond optional tiny metrics later)

Hooks as systemd/cron remains **kzero-selfhosted**.

---

## Relationship to 1.0.0

**1.0.0** locks the stable contract (defaults, exit codes, kind CI, PVC patterns). **1.1.0** adds ergonomics and step types **without** breaking that promise. Prefer additive schema keys; any breaking change needs a migration note.

**Last reviewed:** 2026-08-01 (queue **#59** done → **#29** MVP done → **#58**; **#29b** CRD follow-up; **#57** deferred; **#55** parked)
