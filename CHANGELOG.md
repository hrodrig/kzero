# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-05-13

### Added

- **GNUmakefile** (pgwd-style): `help`, `build`, `build-all`, `install`, `clean`, `test`, `cover`, `lint` (includes **gocyclo** ≤14), `tools`, `security` (govulncheck), `docker-build`, `docker-scan`, **`release-check`**, **`release`**, **`snapshot`**.
- **GoReleaser** (`.goreleaser.yaml`), **Dockerfile** / **Dockerfile.release**, **GitHub Actions**: `ci.yml`, `security.yml`, `release.yml`.
- **`SECURITY.md`**, **`CONTRIBUTING.md`**, **`tools/scan.sh`**, **`codecov.yml`**, **`kzero version`** command.
- **Linux packages**: GoReleaser **nfpm** `.deb` / `.rpm` (plus existing **`archives`** `tar.gz` / `zip`), **`contrib/deb/`** maintainer scripts, **`contrib/README.md`**.

### Changed

- **Makefile** is a FreeBSD-friendly stub that forwards to **gmake** / **GNUmakefile** (same pattern as pgwd).

[Unreleased]: https://github.com/hrodrig/kzero/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/hrodrig/kzero/releases/tag/v0.2.0
