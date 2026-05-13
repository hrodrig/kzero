# kzero roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, schema, and what the engine does **today**
- **[CHANGELOG.md](../CHANGELOG.md)** — what shipped in each release

When a roadmap item ships, update **CHANGELOG** and tick or remove the item here (or move it to a “Completed” subsection with the release tag).

**Last reviewed:** 2026-05-13

---

## v1.0.1 — operator honesty (short horizon)

| # | Item | Status |
|---|------|--------|
| 1 | **CLI warnings** at startup for config the v1 engine does not honor: `run.worker_concurrency > 1`, `retry.attempts > 1`, or `notify.{slack,discord}.enabled` (with tests). | Pending |
| 2 | **Explicit allow-list** for compact pipeline step kinds at parse time so `analyze` rejects what `live` cannot run. | **Done** (v0.2.1) |
| 3 | **Richer `analyze` output**: list normalized steps (reuse existing describe helpers) and flag schema fields that are declared but not implemented. | Pending |
| 4 | **DaemonSet**: remove from built-in scalable workloads; document workaround (`custom:` + `kubectl patch` nodeSelector, etc.). | **Done** (v0.2.1; see SPEC — [supported workload kinds](SPECIFICATIONS.md#supported-workload-kinds)) |

---

## v1.1 — execution engine

5. **Retry** with exponential backoff for transient failures (timeouts, connection errors, throttling), wired to `cfg.Retry`.
6. **Concurrency** via a bounded worker pool from `run.worker_concurrency`, preserving strict YAML order for `down` / `up` unless a future opt-in per-step parallelism is defined.
7. **Propagate `client.id`** into structured logs and hook environment (e.g. `KZERO_CLIENT_ID`).
8. **Typed / classified errors** from subprocesses (common exit codes and messages) to improve `on-error` behavior and UX.

---

## v1.2 — observability and notifications

9. **`log/slog`** with `--log-format json|text`.
10. **Secret redaction** in logs (e.g. `Bearer …`, `password=…`) and optional `--no-env-passthrough` for hooks.
11. **Implement `notify`** (Slack and Discord webhooks as promised by schema), minimal viable path without bespoke retry logic at first.
12. **`verify` mode** after `up`: confirm rollouts ready and emit a structured report (e.g. JSON).

---

## v1.3 — supply chain and more step types

13. **Cosign signing** and **SBOM** (e.g. Syft) in the GoReleaser pipeline.
14. **Raise coverage target** (e.g. 85%+) and fold **`make cover-check`** into **`make release-check`** when sustainable.
15. **Additional step types**: `job`, `cronjob` (suspend), generic `kubectl` patch/scale patterns for CRDs where safe.

---

## v2.0 — native Kubernetes client path

16. **`Executor` abstraction** (shell vs native client-go): start with native scale; keep shell for `kubectl wait` / `rollout status` until structured polling exists.
17. **Stronger dry-run**: server-side dry-run against the API where applicable.
18. **PVC strategy** documented as pipeline patterns (snapshot, wipe, init-job) — not necessarily core primitives on day one.

---

## Maintenance notes

- **GoReleaser**: address `nfpms` deprecation warnings (`maintainer`, `builds` → `ids`) on the next housekeeping release.
- **Integration tests**: optional `kind` / envtest-based paths for `LiveRunner` are not in scope for the table above until someone proposes a CI budget and flake policy (see SPEC testing baseline).
