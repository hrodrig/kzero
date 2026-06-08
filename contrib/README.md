# contrib — packaging helpers for kzero

**kzero is a CLI**, not a daemon or a shipped systemd service. Packages only install the binary and an example config path.

This tree holds files referenced by **GoReleaser** (`.goreleaser.yaml`): maintainer scripts for Linux **`.deb`** / **`.rpm`**, plus **FreeBSD** and **OpenBSD** port skeletons under **`contrib/freebsd/`** and **`contrib/openbsd/`**.

## Linux packages (.deb / .rpm)

GoReleaser **nfpm** integration produces, per Linux architecture:

- **`*.deb`** and **`*.rpm`** — install **`/usr/bin/kzero`** and **`/etc/kzero/kzero.yaml`** (from `configs/kzero.sample.yml`; `config|noreplace` on upgrades)
- **`contrib/deb/prerm.sh`** — no-op (CLI-only package)
- **`contrib/deb/postrm.sh`** — on **`apt purge`**, removes **`/etc/kzero`** so no stale config remains

**Tarballs / zip:** the **`archives`** section in `.goreleaser.yaml` emits **`*.tar.gz`** (and **`*.zip`** on Windows), including **FreeBSD** and **OpenBSD** builds. Each archive includes **`share/man/man1/kzero.1`**, **`share/doc/kzero/LICENSE`**, and **`share/examples/kzero/kzero.sample.yml`** (same layout as the BSD port distfiles).

**Linux packages:** **`.deb`** and **`.rpm`** install **`/usr/share/man/man1/kzero.1.gz`** in addition to **`/usr/bin/kzero`** and **`/etc/kzero/kzero.yaml`**.

**Homebrew:** GoReleaser publishes **`Casks/kzero.rb`** to **[hrodrig/homebrew-kzero](https://github.com/hrodrig/homebrew-kzero)** on each tagged release (`brew install hrodrig/kzero/kzero`). Requires **`HOMEBREW_TAP_TOKEN`** in the kzero repo (see README **Releases and CI**).

Local install from a clone: **`make install-man`** ( **`MANDIR=/usr/share/man`** for system-wide).

## FreeBSD and OpenBSD ports

Maintainer-facing skeletons live under **`contrib/freebsd/`** and **`contrib/openbsd/port/`** (submit to the official trees when ready).

- **`make port-freebsd-sync`** / **`make port-openbsd-sync`** — refresh **`PORTVERSION`** / **`DISTNAME`** / **`PKGNAME`** / **`MASTER_SITES`** / **`DISTFILES`** from the repo **`VERSION`** file.
- **`make dist-freebsd`** — cross-build **`dist/kzero_v<semver>_freebsd_<arch>.tar.gz`** (default **`FREEBSD_ARCH=amd64`**).
- **`make dist-openbsd`** — cross-build **`dist/kzero_v<semver>_openbsd_<arch>.tar.gz`** (default **`OPENBSD_ARCH=amd64`**).

kzero is a **CLI** only: these ports do not install **rc.d** or systemd units. See **`contrib/freebsd/README.md`**, **`contrib/openbsd/README.md`**, and **`contrib/openbsd/port/README.md`**.

Generate everything locally (no git tag required):

```bash
make snapshot
```

Artifacts appear under **`dist/`**.

## Scheduling (optional, operator-owned)

If you want periodic or boot-time runs, invoke **`/usr/bin/kzero`** yourself from **cron**, **CI**, or a **unit file you write** (not provided by this repository). The packages do not enable or ship any long-running service.
