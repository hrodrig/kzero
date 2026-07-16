# Plan 1.1.0 — post-1.0 operator ergonomics (bounded)

**Status:** **Draft** — planning only (2026-07-15). Starts **after** [plan-1.0.0.md](plan-1.0.0.md) ships (**#32–#34**, **#42**).

**Motivation:** Close the highest-value gaps that **do not** turn kzero into a daemon, multi-cluster control plane, or secret broker. Stay **bastion-first**, **config-first**, sequential by default.

For shipped behavior see [CHANGELOG.md](../CHANGELOG.md), [ROADMAP.md](../ROADMAP.md), and [SPECIFICATIONS.md](../SPECIFICATIONS.md).

---

## In scope (tag gates)

| # / ID | Item | Summary |
|--------|------|---------|
| **56** | **Configurable hook interpreter** | Opt-in path for hook / **`custom:`** / shell **`release.*`** scripts (e.g. **`command.shell`** or **`run.hook_interpreter`**) instead of always **`/bin/sh`**. Default remains **`/bin/sh`** (POSIX). Fixes Ubuntu **dash** vs bashisms (`pipefail`, `[[`) without magic shebang. SPEC today: [Hook and script interpreter](../SPECIFICATIONS.md#hook-and-script-interpreter-binsh). |
| **29** | **`job` / `cronjob` + safe CRD patch** | Built-in steps: Job lifecycle, CronJob suspend/resume (or equivalent), and a **narrow** patch/scale pattern for CRDs — prefer **native**; shell fallback where needed. Until then: **`custom:`**. |
| **57** | **Resume / restart from step** | **Phase A (preferred first):** document and/or CLI aid to re-run a pipeline **from step index N** (YAML slice / flag) so operators avoid full replay after mid-reset failure. **Phase B (optional):** on-disk run state + resume — only if Phase A proves insufficient; requires clear idempotency rules. |

## Stretch (not required for v1.1.0)

| # / ID | Item | Notes |
|--------|------|--------|
| **59** | **Helm SDK → v4.2.3+** | **Viable / recommended.** Migrate `helm.sh/helm/v3` → **`helm.sh/helm/v4`** (pin **v4.2.3** or newer). Drops **GO-2026-5932** (`x/crypto/openpgp` → ProtonMail fork). Ahead of Helm **v3** security EOL (**2026-11-11**). See [Helm v4 spike](#helm-v4-spike-2026-07-16). Prefer early optional PR if capacity; not a hard tag gate. |
| **58** | **`kzero diff`** | Live cluster vs plan (replicas, Helm, PVC deletes). Complements **`analyze`** / **`doctor`**. Ship only if cheap after gates above. |
| **55** | **Post-pipeline log upload** | Parked unless operators insist; wrappers / selfhosted remain default. |

---

## Suggested PR order

| PR | Item | Why |
|----|------|-----|
| PR0 | *(Optional)* **#59** Helm SDK v4 | Deps/security before feature work; clears govulncheck ignore; bumps `k8s.io/*` with Helm. |
| PR1 | **#56** hook interpreter | Small surface; cures documented Ubuntu pain; unblocks bash ops without breaking POSIX default. |
| PR2 | **#29** job / cronjob / CRD patch | Largest remaining step-type gap. |
| PR3 | **#57** Phase A (restart from step) | Operational resume without full state machine. |
| PR4 | *(Optional)* **#57** Phase B or **#58** diff | Only with design note + tests. |
| PR5 | Tag **`v1.1.0`** | Release checklist ([release-tests](../.cursor/rules/release-tests.mdc)). |

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
| **Verdict** | **Viable.** Track as stretch **#59**; ship early in 1.1 if capacity (PR0). Do **not** block **#56/#29/#57** on it. |

**Out of spike scope:** rewriting release scripts, requiring host Helm v4 CLI, Helm plugins / Wasm.

---

## Success criteria (v1.1.0 tag)

| # | Criterion | Verify |
|---|-----------|--------|
| 1 | Default interpreter still **`/bin/sh`**; opt-in bash (or path) documented + tested | Unit + SPEC |
| 2 | At least one of **`job`** / **`cronjob`** (or documented patch step) usable in live/dry-run | Tests + sample + SPEC |
| 3 | Operator can restart a failed run from a later step without rewriting the whole YAML by hand | Docs and/or flag + test |
| 4 | **`make release-check`** green | CI |

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

**Last reviewed:** 2026-07-16 (added Helm SDK v4 spike / **#59**)
