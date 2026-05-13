# kzero

<a id="top"></a>

[![Version](https://img.shields.io/badge/version-0.2.0-blue.svg)](https://github.com/hrodrig/kzero/releases)
[![GitHub release](https://img.shields.io/github/v/release/hrodrig/kzero)](https://github.com/hrodrig/kzero/releases)
[![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8.svg)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/hrodrig/kzero)
[![CI](https://github.com/hrodrig/kzero/actions/workflows/ci.yml/badge.svg)](https://github.com/hrodrig/kzero/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/hrodrig/kzero/graph/badge.svg)](https://codecov.io/gh/hrodrig/kzero)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/hrodrig/kzero)](https://pkg.go.dev/github.com/hrodrig/kzero)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrodrig/kzero)](https://goreportcard.com/report/github.com/hrodrig/kzero)
[![deps.dev](https://img.shields.io/badge/deps.dev-go%20module-blue)](https://deps.dev/go/github.com/hrodrig/kzero)
[![Security](https://github.com/hrodrig/kzero/actions/workflows/security.yml/badge.svg)](https://github.com/hrodrig/kzero/actions/workflows/security.yml)
[![CodeQL](https://github.com/hrodrig/kzero/actions/workflows/codeql.yml/badge.svg)](https://github.com/hrodrig/kzero/actions/workflows/codeql.yml)

**Repo:** [github.com/hrodrig/kzero](https://github.com/hrodrig/kzero) · **Releases:** [Releases](https://github.com/hrodrig/kzero/releases) · **DeepWiki:** [hrodrig/kzero](https://deepwiki.com/hrodrig/kzero)

*Badges:* **Version** is a static badge aligned with the repo **`VERSION`** file (next release target). **GitHub release** shows the latest published **tag** on GitHub; it can lag the **`VERSION`** file until a release is cut. **Go** matches **`go.mod`**. **License** points at this repository’s license file. **Ask DeepWiki** links to [DeepWiki](https://deepwiki.com/) AI-generated docs for this repository (see also [badge maker](https://deepwiki.com/badge-maker)). **CI**, **Security**, and **CodeQL** reflect [GitHub Actions](https://github.com/hrodrig/kzero/actions) workflows. **codecov** tracks coverage uploaded from CI. **pkg.go.dev**, **Go Report Card**, and **deps.dev** summarize the Go module and dependencies.

![kzero overview — declarative Kubernetes workload orchestration (pipelines, hooks, workload step types)](docs/kzero-hero-oss.png)

<a id="terminal-demo"></a>
**Terminal demo** (recorded with [VHS](https://github.com/charmbracelet/vhs); source [`docs/demo.tape`](docs/demo.tape), config [`docs/demo-kzero.yaml`](docs/demo-kzero.yaml)):

![kzero CLI — analyze and dry-run down (terminal recording)](docs/demo.gif)

Regenerate from the repo root: **[docs/README.md — Terminal demo](docs/README.md#terminal-demo-vhs)**.

Declarative **Kubernetes workload** orchestration: ordered **down** / **up** (and **reset**) pipelines from YAML, with phase hooks, optional **per-step** `pre` / `post` scripts, `kubectl` scale targets, Helm release helper scripts, and custom steps.

**Releases** ([GitHub Releases](https://github.com/hrodrig/kzero/releases)) ship standalone **binaries** and archives (**`.tar.gz`** / **`.zip`**), Linux **`.deb`** / **`.rpm`**, **Docker** images on **`ghcr.io/hrodrig/kzero`** (distroless runtime — see **[Install or update](#install-or-update)**), and **Homebrew** when published there. This repository does **not** ship Helm charts as a release artifact. Optional operator-focused extras live in **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)**.

Behavior, schema, and acceptance criteria are defined in **[docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md)**. **Diagrams** (Mermaid): **[docs/diagrams.md](docs/diagrams.md)**.

## Table of contents

- [Terminal demo](#terminal-demo)
- [Features](#features)
- [Requirements](#requirements)
- [Install or update](#install-or-update)
- [Quick start](#quick-start)
- [First run](#first-run)
- [Usage examples](#usage-examples)
- [Configuration reference](#configuration-reference)
- [Environment and precedence](#environment-and-precedence)
- [Troubleshooting](#troubleshooting)
- [Security note](#security-note)
- [Releases and CI](#releases-and-ci)
- [Development](#development)
- [Per-step `pre` / `post` (example)](#per-step-pre-post-example)
- [Get involved](#get-involved)
- [License](#license)

---

## Features

- **Configuration-first** (`schema_version: "1.0"`): pipelines and hooks live in config, not hardcoded playbooks.
- **Commands**: `analyze`, `down`, `up`, `reset`, `version` (`reset` = full `down` then `up`; if `down` fails, `up` does not run).
- **Phase hooks**: `pre-down`, `post-down`, `pre-up`, `post-up`, `on-error` (shell script paths).
- **Per-step hooks** (`pre` / `post`): optional shell scripts for a **single** pipeline step—run immediately before and after that step’s main action; **`post` runs only if the main action succeeds**.
- **Step types**: compact refs (`deployment.ns/name`, `statefulset.ns/name`, `daemonset.ns/name`), `release.ns/name` (scripts under `helm.workspace`), and `custom: ./script.sh` (with optional sibling `pre` / `post` keys on the same YAML mapping).
- **Run modes**: `dry-run` (plan only, no cluster mutations) and `live`.

**Libraries** (see [`go.mod`](go.mod)): [Cobra](https://github.com/spf13/cobra) **v1.10.2**, [Viper](https://github.com/spf13/viper) **v1.21.0**.

[↑ Back to top](#top)

## Requirements

- **`kubectl`** on `PATH` (or set **`command.kubectl`** in YAML) with a kubeconfig that can reach the target cluster
- **RBAC** sufficient for the operations in your pipelines (for example **`get`/`patch`/`scale`** on workloads, Helm if you use **`release.*`** steps)
- **`helm`** on `PATH` when you use **`release.namespace/name`** steps (or set **`command.helm`**); not required for workload-only configs
- **Go 1.26.3+** if you [build from source](#quick-start) (`make build`) or use [`go install`](#install-with-go)

[↑ Back to top](#top)

## Install or update

Pre-built **`.deb`**, **`.rpm`**, **`.tar.gz`** (and **`.zip`** on Windows), plus **multi-arch** container images on **`ghcr.io/hrodrig/kzero`**, are on **[GitHub Releases](https://github.com/hrodrig/kzero/releases)** and **[latest release](https://github.com/hrodrig/kzero/releases/latest)**. The **release** badge at the top of this README shows the current tag at a glance.

**Why not a single `latest` URL for every file?** GitHub’s `…/releases/latest/download/<file>` only works if the **asset filename is identical** on every release. GoReleaser here uses the **git tag (with `v`)** in Linux package and archive basenames (for example **`kzero_v0.2.0_linux_amd64.deb`**), while the download path is still `…/download/v0.2.0/…`. **Pick names from the release page**, use the **snippet below**, or use the **badge**.

### Install latest `.deb` (Debian / Ubuntu, `amd64`)

```bash
# Latest published release tag (python3 or jq). Linux .deb basename includes the tag WITH "v".
TAG="$(curl -fsSL https://api.github.com/repos/hrodrig/kzero/releases/latest | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])')"
# Alternative: TAG="$(curl -fsSL https://api.github.com/repos/hrodrig/kzero/releases/latest | jq -r .tag_name)"

[ -n "$TAG" ] || { echo "Could not resolve tag (empty). Install python3 or jq, or set TAG manually from the Releases page." >&2; exit 1; }

DEB="kzero_${TAG}_linux_amd64.deb"
URL="https://github.com/hrodrig/kzero/releases/download/${TAG}/${DEB}"
TMP="/tmp/${DEB}"

# Download to /tmp so user _apt can read the file (apt often cannot read ~/.deb when $HOME is mode 700).
if ! curl -fsSL "$URL" -o "$TMP"; then
  echo "Download failed (curl exit $?). Check URL: $URL" >&2
  exit 1
fi
if [ ! -f "$TMP" ]; then
  echo "Expected $TMP after download — not found." >&2
  exit 1
fi
sudo apt install "$TMP"
```

Paste the block **as a whole**, or chain with `&&`, so **`apt` does not run** after a failed **`curl`**. **`curl -f`** (via `-fsSL`) exits non‑zero on HTTP errors (404, etc.).

**`apt` + `_apt` / “Permission denied” under `$HOME`:** if you `curl` the `.deb` into **`~`** and run `sudo apt install ./kzero_….deb`, Debian/Ubuntu may warn that **`_apt` cannot read the file** (home directory not world-executable). Use **`/tmp`** as above, or `sudo cp "$DEB" /tmp/` then `sudo apt install "/tmp/$DEB"`.

**Empty `TAG`:** if `jq`/`python3` failed, you get `.../download//kzero__linux_amd64.deb` and a broken filename.

**`.deb` / `.rpm`** install **`kzero`** to **`/usr/bin`** and place **`/etc/kzero/kzero.yaml`** (from the sample; **`config|noreplace`** on upgrades). The CLI defaults to **`./kzero.yaml`** when **`--config`** is omitted—use **`kzero --config /etc/kzero/kzero.yaml …`** after install, or copy that file to **`./kzero.yaml`**. Use **`linux_arm64`** in the download filename on ARM64 Linux.

### Fixed-tag examples (copy from the release page if you prefer)

| Format | Example (tag **`v0.2.0`** in the URL path; artifact basename includes the same **`v0.2.0`**) |
|--------|------------------------------------------------------------------|
| **`.deb`** | `curl -fsSL -o /tmp/kzero_v0.2.0_linux_amd64.deb https://github.com/hrodrig/kzero/releases/download/v0.2.0/kzero_v0.2.0_linux_amd64.deb` then `sudo apt install /tmp/kzero_v0.2.0_linux_amd64.deb` |
| **`.rpm`** | `curl -fsSLO https://github.com/hrodrig/kzero/releases/download/v0.2.0/kzero_v0.2.0_linux_amd64.rpm` then `sudo rpm -Uvh kzero_v0.2.0_linux_amd64.rpm` or `sudo dnf install ./kzero_v0.2.0_linux_amd64.rpm` |
| **`.tar.gz` (Linux)** | `curl -fsSLO https://github.com/hrodrig/kzero/releases/download/v0.2.0/kzero_v0.2.0_linux_amd64.tar.gz` then `tar xzf kzero_v0.2.0_linux_amd64.tar.gz` and run **`./kzero`** from the extracted tree (see **`share/examples/kzero/kzero.sample.yml`**) |
| **`.tar.gz` (macOS)** | `curl -fsSLO https://github.com/hrodrig/kzero/releases/download/v0.2.0/kzero_v0.2.0_darwin_amd64.tar.gz` (or **`…_darwin_arm64.tar.gz`** on Apple silicon) |

**Update:** download a newer release and run the same install command again (`rpm -Uvh`, `apt install` over the `.deb`, or replace the tarball tree).

**Windows:** use the **`.zip`** asset for your arch (for example **`kzero_v0.2.0_windows_amd64.zip`**), unpack, run **`kzero.exe`** where **`kubectl`** is available.

**Docker:** `docker pull ghcr.io/hrodrig/kzero:v0.2.0` (match the image tag to the **[release](https://github.com/hrodrig/kzero/releases)** you want). Published images use **`gcr.io/distroless/static-debian12:nonroot`** (static **`kzero`** binary only: no shell, no BusyBox/Alpine runtime). **`Dockerfile`** in this repo uses the same final stage. Package: [ghcr.io/hrodrig/kzero](https://github.com/hrodrig/kzero/pkgs/container/kzero).

**Homebrew** and **BSD packaging** helpers: when published, see **Releases** and **`contrib/README.md`**.

Then run **`kzero --config /etc/kzero/kzero.yaml analyze`** (or follow **[Quick start](#quick-start)** to build from a clone).

[↑ Back to top](#top)

## Quick start

```bash
make build
./bin/kzero --help
./bin/kzero analyze --config configs/kzero.sample.yml
```

On FreeBSD, use **`gmake`** (or plain `make`, which forwards to GNU Make via the stub **Makefile**).

Copy and edit `configs/kzero.sample.yml` (or use `--config` / `./kzero.yaml`). Override values via environment variables with the `KZERO_` prefix (see Viper / sample file comments).

If you installed from **[Releases](#install-or-update)** or **`go install`** (below), use **`kzero`** on your `PATH` instead of **`./bin/kzero`**.

### Install with Go

From any machine with Go **1.26.3+** (installs to `$(go env GOPATH)/bin`; ensure that directory is on your `PATH`):

```bash
go install github.com/hrodrig/kzero/cmd/kzero@latest
```

Use a **release tag** instead of `@latest` if you want a pinned version (for example `@v0.2.0`). Module reference: [pkg.go.dev/github.com/hrodrig/kzero](https://pkg.go.dev/github.com/hrodrig/kzero).

[↑ Back to top](#top)

## First run

kzero reads **one YAML file** per invocation. If you omit **`--config`**, it loads **`./kzero.yaml`** from the **current working directory** (there is no automatic search under **`/etc`**—after a `.deb`/`.rpm` install use **`kzero --config /etc/kzero/kzero.yaml …`** or copy that file to **`./kzero.yaml`**).

1. Start from **[`configs/kzero.sample.yml`](configs/kzero.sample.yml)** in a clone, from **`share/examples/kzero/kzero.sample.yml`** inside a **release tarball**, or from **`/etc/kzero/kzero.yaml`** after a Linux package install.
2. Keep **`run.mode: dry-run`** until **`kzero analyze`** and pipeline plans match what you expect; switch to **`live`** only when ready ([docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md)).
3. Set **`command.kubectl`** / **`command.helm`** to absolute paths for cron or minimal environments.

```bash
cp configs/kzero.sample.yml kzero.yaml
# Edit kzero.yaml: cluster, pipelines, helm.workspace, hook paths, notify, run.*
kzero analyze
kzero down
```

[↑ Back to top](#top)

## Usage examples

Paths use **`./bin/kzero`** after `make build`. If you installed from **[Releases](#install-or-update)** or **`go install`**, use **`kzero`** on your `PATH` the same way.

### Plan only (`analyze`, `down` / `up` in `dry-run`)

```bash
kzero analyze --config ./kzero.yaml
# With run.mode: dry-run in YAML:
kzero down --config ./kzero.yaml
```

### Explicit config path

```bash
kzero analyze --config /path/to/prod/kzero.yaml
kzero up --config /path/to/prod/kzero.yaml
```

### `reset` (`down` then `up`)

```bash
kzero reset --config ./kzero.yaml
```

If **`down`** fails, **`up`** is **not** executed (see [`RunReset`](https://github.com/hrodrig/kzero/blob/main/internal/engine/engine.go#L44)).

### Version metadata

```bash
kzero version
```

[↑ Back to top](#top)

## Configuration reference

Full schema, validation, and acceptance criteria: **[docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md)**. Annotated sample: **[configs/kzero.sample.yml](configs/kzero.sample.yml)**.

| Block / key | Purpose |
|-------------|---------|
| **`schema_version`** | Must be **`1.0`** today. |
| **`cluster`** | Metadata (`name`, `environment`, …) for labels and notifications. |
| **`helm`** | **`workspace`**: directory with `<release>.sh` scripts and values for **`release.ns/name`** steps. |
| **`command`** | Optional paths for **`kubectl`** and **`helm`**. |
| **`hooks`** | Optional global scripts: **`pre-down`**, **`post-down`**, **`pre-up`**, **`post-up`**, **`on-error`**. |
| **`notify`** | Optional **`slack`** / **`discord`** blocks (see SPEC). |
| **`pipelines`** | See [`pipelines`](#pipelines) below. |
| **`retry`** | See [`retry`](#retry) below. |
| **`run`** | See [`run`](#run) below. |

### `run`

| Key | Purpose |
|-----|---------|
| **`mode`** | **`dry-run`** (log plan only) or **`live`** (execute `kubectl` / `helm`). Required. |
| **`timeout`** | Wall-clock budget for a full **`down`**, **`up`**, or **`reset`** (Go duration, e.g. **`25m`**). |
| **`kubeconfig`** | Path passed to **`kubectl`** / **`helm`**; empty uses the process environment / default kubeconfig search. |
| **`worker_concurrency`** | Integer in the schema; the **v1** engine runs pipeline steps **sequentially** in YAML order. Parsed and stored for forward compatibility—see [SPEC — Current engine](docs/SPECIFICATIONS.md#current-engine-sequencing-retry-and-worker-concurrency). |
| **`operation_timeout`** | Per-operation ceiling (e.g. **`45s`**) for individual kubectl/helm calls inside a step. |

### `retry`

| Key | Purpose |
|-----|---------|
| **`attempts`** | Declared retry count (**integer**). |
| **`delay`** | Backoff between attempts (**Go duration**, e.g. **`8s`**). |

The engine **does not** retry failed steps yet; keys are part of the contract. See [SPEC — Current engine](docs/SPECIFICATIONS.md#current-engine-sequencing-retry-and-worker-concurrency).

### `pipelines`

- **`down`** and **`up`** are ordered lists. Each item is either:
  - a **string** step reference: **`deployment.ns/name`**, **`statefulset.ns/name`**, **`daemonset.ns/name`**, **`release.ns/name`**; or
  - a **single-key map** whose key is one of the above **or** **`custom`**, with optional fields beside that key (see below).
- **`release.*`** steps require **`helm.workspace`** to point at the directory of **`<release>.sh`** scripts (see SPEC and sample).

Optional fields on a **map** step (same YAML mapping as the step ref, alongside **`pre`** / **`post`**):

| Field | Applies to | Purpose |
|-------|------------|---------|
| **`pre`** / **`post`** | workload, `release`, **`custom`** | Shell scripts run immediately before / after the main action; **`post`** only if the main action succeeds. |
| **`replicas`** | mainly **`up`** / scale targets | Target replica count (integer). |
| **`wait_for_ready`** | workloads with readiness | If **`true`**, wait for rollout / ready condition (see SPEC). |
| **`timeout`** | step-level override | Go duration string (e.g. **`10m`**) for that step’s bounded wait / operations. |

[↑ Back to top](#top)

## Environment and precedence

- **`KZERO_`** environment variables override values from the loaded YAML (**Viper** `AutomaticEnv`). Nested keys use underscores (for example **`run.mode`** → **`KZERO_RUN_MODE`**).
- **`--config /path/to/kzero.yaml`** selects that file explicitly. When omitted, the default file is **`./kzero.yaml`** (must exist; there is no built-in fallback path).

[↑ Back to top](#top)

## Troubleshooting

### `read config: …` or “cannot find the file”

kzero always loads a real file path. With no **`--config`**, that path is **`./kzero.yaml`** relative to the **current working directory**. **Fix:** `cd` to the directory that contains **`kzero.yaml`**, pass **`--config /absolute/path/kzero.yaml`**, or copy **`configs/kzero.sample.yml`** / **`/etc/kzero/kzero.yaml`** to **`./kzero.yaml`**.

### `run.mode must be one of: dry-run, live`

Only those two literals are accepted (see validation in [`internal/config/load.go`](https://github.com/hrodrig/kzero/blob/main/internal/config/load.go)). Check for typos, quotes, or trailing spaces in YAML.

### `helm.workspace is required when pipelines include release steps`

Any **`release.ns/name`** entry requires **`helm.workspace`** in the root config. Set it to the directory that contains your **`<release>.sh`** scripts, or remove **`release.*`** steps if you are not using Helm-driven releases.

### `reset` never ran `up` / pipeline stopped halfway

**`reset`** runs **`down`** then **`up`** under one **`run.timeout`**. If **`down`** returns an error, **`up` is skipped** ([`RunReset`](https://github.com/hrodrig/kzero/blob/main/internal/engine/engine.go#L44)). Inspect the **`down`** failure (hooks, RBAC, or a failing step) before re-running.

[↑ Back to top](#top)

## Security note

- **`run.mode: live`** performs real cluster mutations (`kubectl` / `helm` as configured). Stay on **`dry-run`** until reviews pass.
- **Hooks** (`hooks.*`, per-step **`pre`/`post`**, **`custom:`**) run as the **same OS user** that invokes **`kzero`**. Only reference scripts you trust; treat changes like production code.
- Use **least-privilege RBAC** for the kube identity in use; pipeline steps may scale, delete, or upgrade workloads depending on your YAML.

See **`SECURITY.md`** for reporting vulnerabilities.

[↑ Back to top](#top)

## Releases and CI

Aligned with **[pgwd](https://github.com/hrodrig/pgwd)**:

1. Work on **`develop`**; merge to **`main`** when ready.
2. Before tagging: run **`make release-check`** (requires **Docker**): semver **`VERSION`**, **`make lint`** (gofmt, go vet, **gocyclo** ≤14), **`make test`**, **`make security`** (govulncheck), **`make docker-scan`** (Grype on the image; use **`GRYPE_FAIL_ON`** to tune the gate, default **high**).
3. On **`main`**: create an annotated tag (e.g. `git tag -a v0.2.0 -m "Release 0.2.0"`) and **`git push origin v0.2.0`**. The **Release** workflow runs **`make release-check`** then **GoReleaser** (binaries + **`ghcr.io/hrodrig/kzero`**).
4. Local release after checks: **`make release`** (same as CI tail; **main** branch only).

Snapshot builds (no git tag): **`make snapshot`** → artifacts under **`dist/`** (archives **`.tar.gz`** / **`.zip`**, Linux **`.deb`** / **`.rpm`**, checksums). See **`contrib/README.md`** for packaging notes.

[↑ Back to top](#top)

## Development

```bash
make test
make lint
make build
```

See **`CONTRIBUTING.md`** and **`CHANGELOG.md`**. Security reporting: **`SECURITY.md`**.

[↑ Back to top](#top)

## <a id="per-step-pre-post-example"></a>Per-step `pre` / `post` (example)

Use this when something must happen **between** ordered steps—for example, run a script while a StatefulSet is still running, **then** scale that StatefulSet to zero in the same step:

```yaml
pipelines:
  down:
    - deployment.app/worker
    - statefulset.app/etl-state-store:
        pre: ./hooks/truncate-etljobs.sh
```

Here `truncate-etljobs.sh` runs **before** `kubectl scale` for `etl-state-store` (so the pod can still accept `kubectl exec` or similar). Optional `post` runs **after** a successful scale (or other main action for that step).

**Custom** step with hooks (same list item, multiple keys):

```yaml
    - custom: ./hooks/maintenance.sh
      pre: ./hooks/before-maintenance.sh
      post: ./hooks/after-maintenance.sh
```

[↑ Back to top](#top)

## Get involved

Found kzero useful? We'd love your help to make it better. You can:

- **Report bugs** or **suggest features** — [open an issue](https://github.com/hrodrig/kzero/issues)
- **Contribute code** — see [CONTRIBUTING.md](CONTRIBUTING.md) for how to submit a pull request
- **Star the repo** — it helps others discover kzero

Thanks for using kzero.

[↑ Back to top](#top)

## License

See [LICENSE](LICENSE).

[↑ Back to top](#top)
