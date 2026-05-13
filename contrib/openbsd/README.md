# OpenBSD — kzero

**kzero** is a **CLI** only: there is no bundled **rc.d** script in this repository. Operators run **`kzero`** from cron, CI, or their own wrappers.

Official port skeleton: **`contrib/openbsd/port/`** (submit to **ports@openbsd.org**).

Release tarballs match **`DISTFILES`** in that port (binary **`kzero`** plus **`share/doc/kzero/LICENSE`** and **`share/examples/kzero/kzero.sample.yml`**). Build a matching tarball locally with **`make dist-openbsd`** from the repo root.
