# Contributing

- Default branch for work: **`develop`**. **`main`** is production-ready; releases are tagged from `main` only (see [.cursor/rules/git-flow.mdc](.cursor/rules/git-flow.mdc) if present in your clone).
- Before opening a PR: **`make lint`**, **`make test`**, and **`make cover-check`** (total statement coverage must be **≥ 80%** unless you temporarily set `COVERAGE_MIN=` for a documented reason).
- Before a release tag: **`make release-check`** (lint, tests, govulncheck, Grype on the built image — requires Docker). Sync **`contrib/man/man1/kzero.1`**: update **`.TH`** date and `kzero vX.Y.Z` to match **`VERSION`**, and document new CLI flags/commands; `release-check` fails if `.TH` version drifts.

Use English for code, comments, commit messages, and docs.
