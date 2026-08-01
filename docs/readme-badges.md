# README badges

Meanings of the status badges at the top of the [repository README](../README.md).

| Badge | Meaning |
|-------|---------|
| **Version** | Static badge aligned with the repo **`VERSION`** file (next release target). |
| **GitHub release** | Latest published **tag** on GitHub; can lag **`VERSION`** until a release is cut. |
| **Go** | Matches **`go.mod`**. |
| **License** | Points at this repository’s license file. |
| **Ask DeepWiki** | Links to [DeepWiki](https://deepwiki.com/) AI-generated docs for this repository (see also [badge maker](https://deepwiki.com/badge-maker)). |
| **CI**, **Security**, **CodeQL** | [GitHub Actions](https://github.com/hrodrig/kzero/actions) workflows. |
| **codecov** | Coverage uploaded from CI. |
| **pkg.go.dev**, **deps.dev** | Go module and dependency summaries. |
| **gghstats clones** | Git clone traffic for this repo (see [gghstats](https://github.com/hrodrig/gghstats)). |

When bumping **`VERSION`**, keep the static **Version** badge in sync — see [`.cursor/rules/readme-badges-version.mdc`](../.cursor/rules/readme-badges-version.mdc) (local) and release checklist in [AGENTS.md](../AGENTS.md).
