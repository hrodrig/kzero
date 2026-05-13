# FreeBSD port for kzero

Port files for building and installing **kzero** (CLI only; no rc.d or daemon).

## Install from port

When the port is in the official tree:

```bash
cd /usr/ports/sysutils/kzero
make install
```

Local port (copy `Makefile`, `pkg-plist`, `pkg-descr` from this directory):

```bash
cd ~/ports/sysutils/kzero
make install
```

After changing port files: `make deinstall && make clean && make install`.

## Test with a local distfile

1. From the **kzero** repo root, sync **PORTVERSION** with **`VERSION`**:

   ```bash
   make port-freebsd-sync
   ```

2. Build the tarball expected by **DISTFILES** (default arch **amd64**; override with **`FREEBSD_ARCH=arm64`**):

   ```bash
   make dist-freebsd
   ```

   Output: `dist/kzero_v<version>_freebsd_<arch>.tar.gz`.

3. Copy into **DISTDIR** or use **`MASTER_SITES=file:///.../`** as in the [FreeBSD Porter's Handbook](https://docs.freebsd.org/en/books/porters-handbook/).

The tarball contains: `kzero`, `share/doc/kzero/LICENSE`, `share/examples/kzero/kzero.sample.yml`.
