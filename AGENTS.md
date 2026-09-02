# Agent Guidelines (kzero)

Context and instructions for AI coding agents working on **kzero** (declarative Kubernetes workload reset CLI). See [agents.md](https://agents.md/) for the format.

## Project overview

- **What it is:** Go CLI that runs **out-of-band** **`down` / `up` / `reset`** pipelines from YAML — ordered scale, Helm, PVC, exec, job/cronjob, hooks, dry-run/analyze, notify, and API watchdog. Not in-cluster GitOps; bastion-first (see [docs/deployment-models.md](docs/deployment-models.md)).
- **Entrypoint:** `cmd/kzero/main.go`. Core packages: `internal/config`, `internal/engine`, `internal/executor`, `internal/cli`, `internal/diff`, `internal/doctor`, `internal/notify`, `internal/probe`, `internal/verify`.
- **Config contract:** [SPECIFICATIONS.md](SPECIFICATIONS.md) (root; also linked as `docs/SPECIFICATIONS.md` in some clones). Example profile: [configs/kzero.sample.yml](configs/kzero.sample.yml). Feature cookbooks: `docs/examples/`.
- **External tools:** **`kubectl`** on `PATH` (required). **`helm`** when the profile uses `release.*` shell steps. Behavior is **configuration-first** — no product-specific playbooks hardcoded in Go (see `.cursor/rules/declarative-workflow-go.mdc` in a local clone).

## Scope (product vs operator)

- **This repo (`kzero`):** Go CLI, engine, validation, tests, CI, **`make release-check`**, binaries, `.deb`/`.rpm`, **`ghcr.io/hrodrig/kzero`**, Homebrew — same split as **pgwd** / **pgwd-selfhosted**.
- **Not here:** bastion cron/systemd, `docker run` operator notes, kind e2e lab assets, reference hook scripts, in-cluster manifests → **[hrodrig/kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)**. Do not add `run/` or operator deployment trees to this repository.
- **`.cursor/`** is **local-only** (not committed). Shared agent and release policy belong in tracked files such as this **AGENTS.md**, **README**, and **CONTRIBUTING.md**.

## Setup and build

- Install deps: `go mod download`.
- Build: **`make build`** (reads **`VERSION`**, ldflags for version/commit/date/branch). Root **`GNUmakefile`** is canonical; on **FreeBSD** use **`gmake`** via the BSD **`Makefile`** stub.
- Install to `$GOBIN`: `make install`. Optional kubectl plugin: `make install-kubectl-plugin`.
- Cross-compile: `make build-all` (output under `dist/`).

## Test and quality commands

- **Unit tests:** `make test` or `go test ./...`.
- **Coverage:** `make cover`; gate **`make cover-check`** (total statement coverage **≥ 80%**, override `COVERAGE_MIN=` only for a documented one-off).
- **Lint:** `make lint` (`gofmt -s`, `go vet ./...`, gocyclo ≤ 14). Fix format: `make lint-fix`.
- **Security:** `make security` (govulncheck; filtered advisories in `.govulncheck-ignore.yaml`).
- **Docker image scan:** `make docker-scan` (Grype on built image; requires Docker).
- **Integration (optional):** `make test-kind` — kind cluster smoke (see `testing/kind/`).
- **Before release:** **`make release-check`** — validates **`VERSION`** semver + man **`.TH`**, then lint, test, cover-check, security, docker-scan. **Requires Docker.**

Install tooling to `$GOBIN`: `make tools` (govulncheck, gocyclo).

## Git flow

- **Branches:** Day-to-day work on **`develop`**. Topic branch → **PR into `develop`** (green CI, merge, delete branch). Production: **PR `develop` → `main`**, then annotated tag **`v<semver>`** on **`main`** only.
- **Never** merge to **`main`**, create/push a release tag, or run **`make release`** / trigger GoReleaser **without explicit user approval** in the current conversation — even if `release-check` is green.
- **Commits:** Show the proposed commit message and wait for user approval before `git commit` (see `.cursor/rules/commit-message-review.mdc` locally).
- **Language:** English only for code, comments, commit messages, docs, and UI strings.

## Version bump (on `develop`, before merge/tag)

Do the **VERSION bump as a dedicated change on `develop`** after feature work lands, **before** proposing `make release-check` → merge to `main` → annotated tag. **Stop and ask** before merge/tag steps.

| # | Artifact | Action |
|---|----------|--------|
| 1 | **`VERSION`** | New semver without `v` (e.g. `1.1.1`) |
| 2 | **`README.md`** | Static **Version** badge `version-<semver>`; update “Shipped” / release tables if present |
| 3 | **`CHANGELOG.md`** | Move `[Unreleased]` into `## [X.Y.Z] - YYYY-MM-DD`; update compare links at bottom |
| 4 | **`contrib/man/man1/kzero.1`** | `.TH` line: month/year + `kzero v<VERSION>`; new CLI flags/commands |
| 5 | **`docs/demo.gif`** | Re-record when CLI output, version string, or demo scenario changed (`bash -c "vhs docs/demo.tape"` — see [docs/README.md](docs/README.md)) |
| 6 | **BSD ports** | `make port-freebsd-sync` and/or `make port-openbsd-sync` |
| 7 | **`docs/ROADMAP.md`** | **Last reviewed** date; shipped highlights; tick completed bands |
| 8 | **Gate** | `make release-check` — run only after user asks |
| 9 | **Ship** | Merge `develop` → `main`, annotated tag `v<semver>`, push tag — **only after user explicitly approves** |

**Follow-ups (other repos, after GitHub Release is green):** pin app version in **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)**; marketing/install sites on GitLab (`kzero-hermesrodriguez-com`, `get-kzero-hermesrodriguez-com`) if they hard-code a release.

## Man page sync

Keep **`contrib/man/man1/kzero.1`** aligned with the CLI. GoReleaser gzips it for packages; **`release-check`** fails if **`.TH`** version drifts from **`VERSION`**.

**Source of truth:** `internal/cli/`, `cmd/kzero/main.go`, [SPECIFICATIONS.md](SPECIFICATIONS.md).

## Docker

- Local image: `make docker-build` → `kzero:local`.
- Release image: **`ghcr.io/hrodrig/kzero`** (GoReleaser on tag push). Dockerfile uses pinned Go toolchain (keep in sync with `go.mod`).

## Repository structure

- `cmd/kzero/` — main package and black-box CLI tests.
- `internal/config/` — YAML load, validation, env overrides.
- `internal/engine/` — pipeline runner, dry-run, live execution, watchdog.
- `internal/executor/` — step types (workload, helm, pvc, exec, job, cronjob, shell, …).
- `internal/cli/` — Cobra commands (`analyze`, `diff`, `doctor`, `down`, `up`, `reset`, …).
- `docs/` — specifications index, examples, VHS demo (`docs/demo.tape` → `docs/demo.gif`), diagrams.
- `contrib/` — man page, FreeBSD/OpenBSD ports, packaging helpers.
- `testing/kind/` — optional kind e2e (not required for every PR; CI runs integration on `develop`/`main`).

## Skills

- **golang-pro** (`.agents/skills/golang-pro/SKILL.md` in a local clone): idiomatic Go, table-driven tests, concurrency, interfaces — use for non-trivial Go changes.

## Other instructions

- **README:** Keep badges and version badge in sync with **`VERSION`** (see `docs/readme-badges.md`).
- **CHANGELOG:** User-facing changes under `[Unreleased]`; finalize on release (`.cursor/rules/changelog.mdc` locally).
- **Supply chain:** Prefer resolving dependency work inside the clone (`go get`, `go mod tidy`, `go test ./...`). Do not disable checksum verification or use untrusted proxies unless the user explicitly accepts the risk.
- When adding dependencies: `go mod tidy` and ensure tests still pass.
