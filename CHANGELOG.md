# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`docs/ROADMAP.md`**: in-repo prioritized roadmap (v1.0.1–v2.0) and status for items already shipped in v0.2.1; linked from **README** and **docs/README**.


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

[Unreleased]: https://github.com/hrodrig/kzero/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/hrodrig/kzero/releases/tag/v0.2.1
[0.2.0]: https://github.com/hrodrig/kzero/releases/tag/v0.2.0
