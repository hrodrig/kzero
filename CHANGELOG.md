# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-06-04

### Added

- **`run.execution`**: `shell` (default), `native` (client-go), or `auto` (native with shell fallback) for `deployment` / `statefulset` steps in live mode.
- **`internal/executor`**: `Shell` and `Native` workloads; typed API error sentinels; fake-clientset tests.

### Changed

- **Docs:** [docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md) documents workload execution backend; [docs/ROADMAP.md](docs/ROADMAP.md) marks 0.4.x items 5–9 shipped in **0.4.0**.
- **Dependencies:** `k8s.io/client-go` and `k8s.io/api` for native execution; `golang.org/x/net` v0.55.0 (govulncheck clean with client-go).

## [0.2.3] - 2026-06-04

### Added

- **CLI warnings** (stderr) after successful config load when **`run.worker_concurrency` > 1**, **`retry.attempts` > 1**, or **`notify.slack.enabled` / `notify.discord.enabled`** is true — those fields are accepted by the schema but not yet implemented by the v1 engine (`analyze`, `down`, `up`, `reset`).
- **`kzero analyze`**: prints normalized **`[down]`** / **`[up]`** step lists (with optional script paths, hooks, and scale options), phase hooks, run metadata, and a **Deferred** summary on stdout for schema fields not yet honored by the engine.

### Changed

- **Docs:** [docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md) and [README.md](README.md) describe the **`kzero analyze`** stdout/stderr contract.
- **Roadmap:** strategic priority for **client-go** / native executor (**0.4.x**); **0.3.x** operator-honesty band complete.
- **Go toolchain:** `go.mod` and build images require **Go 1.26.4+** (stdlib).

### Security

- Bump minimum Go version to **1.26.4** to address [GO-2026-5037](https://pkg.go.dev/vuln/GO-2026-5037) (`crypto/x509` hostname parsing) reported by `govulncheck` on Go 1.26.3.

## [0.2.2] - 2026-05-13

### Added

- **`docs/ROADMAP.md`**: in-repo prioritized roadmap (semver bands **0.3.x**–**1.0.0 (future)**) plus **Shipped** for **0.2.0** / **0.2.1**; linked from **README** and **docs/README**.

### Changed

- **`docs/ROADMAP.md`**: rename milestone headings from fictional `v1.0.x` / `v2.0` style labels to **0.3.x**–**1.0.0 (future)** bands aligned with public releases starting at **0.2.0**.

### Fixed

- **`CHANGELOG.md`**: restore the **`[0.2.1]`** section (Removed / Changed) that had been folded under **`[Unreleased]`** by mistake, so published **0.2.1** release notes match the tagged release again.

## [0.2.1] - 2026-05-13

### Removed

- `daemonset.<namespace>/<name>` step kind in `pipelines.down` / `pipelines.up`. The v1 engine ran `kubectl scale daemonset/...`, which the Kubernetes API server rejects because DaemonSet has no `/scale` subresource (`Error from server (NotFound)`). Configs that reference `daemonset.*` now fail at parse time. **Migration:** replace those steps with a `custom: ./hooks/<name>.sh` step that runs `kubectl patch daemonset ... --type=strategic -p '{"spec":{"template":{"spec":{"nodeSelector":{"kzero.io/disabled":"true"}}}}}'` to drain the pods (and a matching script on `up` that removes the nodeSelector key). See `docs/SPECIFICATIONS.md` → *Supported workload kinds*.

### Changed

- `pipelines.{down,up}` reject unsupported step kinds at config load time via an explicit allow-list (`deployment`, `statefulset`, `release`). Previously, refs such as `cronjob.<ns>/<name>`, `job.<ns>/<name>`, or `service.<ns>/<name>` passed validation and failed only later in live mode with `unsupported pipeline resource type`. `kzero analyze` now surfaces the problem before any cluster mutation.

## [0.2.0] - 2026-05-13

### Added

- **GNUmakefile** (pgwd-style): `help`, `build`, `build-all`, `install`, `clean`, `test`, `cover`, `lint` (includes **gocyclo** ≤14), `tools`, `security` (govulncheck), `docker-build`, `docker-scan`, **`release-check`**, **`release`**, **`snapshot`**.
- **GoReleaser** (`.goreleaser.yaml`), **Dockerfile** / **Dockerfile.release**, **GitHub Actions**: `ci.yml`, `security.yml`, `release.yml`.
- **`SECURITY.md`**, **`CONTRIBUTING.md`**, **`tools/scan.sh`**, **`codecov.yml`**, **`kzero version`** command.
- **Linux packages**: GoReleaser **nfpm** `.deb` / `.rpm` (plus existing **`archives`** `tar.gz` / `zip`), **`contrib/deb/`** maintainer scripts, **`contrib/README.md`**.

### Changed

- **Makefile** is a FreeBSD-friendly stub that forwards to **gmake** / **GNUmakefile** (same pattern as pgwd).

[Unreleased]: https://github.com/hrodrig/kzero/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/hrodrig/kzero/compare/v0.2.3...v0.4.0
[0.2.3]: https://github.com/hrodrig/kzero/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/hrodrig/kzero/releases/tag/v0.2.2
[0.2.1]: https://github.com/hrodrig/kzero/releases/tag/v0.2.1
[0.2.0]: https://github.com/hrodrig/kzero/releases/tag/v0.2.0
