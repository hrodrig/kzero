# Contributing

- **Branch policy:** `develop` is integration; `main` is production. Releases are tagged from `main` only.
- **Code:** From an up-to-date `develop`, create a **topic branch** (`feat/…`, `fix/…`, `chore/…`, `security/…`, `docs/…`), push it, and open a pull request **into `develop`**. Wait for green CI, then merge.
- **No direct push** to `develop` or `main` (GitHub rulesets `protect-develop` / `protect-main` — same pattern as [pgwd](https://github.com/hrodrig/pgwd) / [gghstats](https://github.com/hrodrig/gghstats)).
- **Release:** PR **`develop` → `main`**, then annotated tag `vX.Y.Z` on `main`. After every merge into `main`, sync **`main` → `develop`** so the next release PR is not **"out-of-date with the base branch"**.
- Before opening a PR: **`make lint`**, **`make test`**, and **`make cover-check`** (total statement coverage must be **≥ 80%** unless you temporarily set `COVERAGE_MIN=` for a documented reason).
- Before a release tag: **`make release-check`** (lint, tests, govulncheck, Grype on the built image — requires Docker). Sync **`contrib/man/man1/kzero.1`**: update **`.TH`** date and `kzero vX.Y.Z` to match **`VERSION`**, and document new CLI flags/commands; `release-check` fails if `.TH` version drifts.

Use English for code, comments, commit messages, and docs.
