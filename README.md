# kzero

<a id="top"></a>

[![Version](https://img.shields.io/badge/version-1.0.1-blue.svg)](https://github.com/hrodrig/kzero/releases)
[![GitHub release](https://img.shields.io/github/v/release/hrodrig/kzero)](https://github.com/hrodrig/kzero/releases)
[![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8.svg)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/hrodrig/kzero)
[![CI](https://github.com/hrodrig/kzero/actions/workflows/ci.yml/badge.svg)](https://github.com/hrodrig/kzero/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/hrodrig/kzero/graph/badge.svg)](https://codecov.io/gh/hrodrig/kzero)
[![gghstats clones](https://gghstats.hermesrodriguez.com/api/v1/badge/hrodrig/kzero?metric=clones)](https://gghstats.hermesrodriguez.com/hrodrig/kzero)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/hrodrig/kzero)](https://pkg.go.dev/github.com/hrodrig/kzero)
[![deps.dev](https://img.shields.io/badge/deps.dev-go%20module-blue)](https://deps.dev/go/github.com%2Fhrodrig%2Fkzero)
[![Security](https://github.com/hrodrig/kzero/actions/workflows/security.yml/badge.svg)](https://github.com/hrodrig/kzero/actions/workflows/security.yml)
[![CodeQL](https://github.com/hrodrig/kzero/actions/workflows/codeql.yml/badge.svg)](https://github.com/hrodrig/kzero/actions/workflows/codeql.yml)


**Repo:** [github.com/hrodrig/kzero](https://github.com/hrodrig/kzero) · **Releases:** [Releases](https://github.com/hrodrig/kzero/releases) · **DeepWiki:** [hrodrig/kzero](https://deepwiki.com/hrodrig/kzero)

**The problem:** Platform maintenance often needs an **ordered** teardown and bring-up (scale down → wipe data → Helm uninstall → restore) — not continuous GitOps reconcile, not ad-hoc shell. When the API or data plane is unhealthy, recovery **inside** the same cluster shares the failure domain you are trying to fix: Jobs stall, logs vanish with the Pod, and alerts never leave the sick path.

**How kzero solves it:** A **bastion-first** CLI runs declarative YAML **`down` / `up` / `reset`** pipelines **out-of-band** — from a host you trust more than the cluster you are resetting. Config owns the playbook (order, hooks, dry-run / analyze); the engine provides fail-fast, optional API watchdog, and notify so maintenance stays repeatable without hardcoded product scripts in Go.

Where to run: [docs/deployment-models.md](docs/deployment-models.md). Scope vs alternatives: [docs/scope-and-alternatives.md](docs/scope-and-alternatives.md).

![kzero — out-of-band bastion-first declarative workload reset (down / up / reset from YAML)](docs/kzero-hero-oss.png)

<a id="terminal-demo"></a>
**Terminal demo** (recorded with [VHS](https://github.com/charmbracelet/vhs); source [`docs/demo.tape`](docs/demo.tape), config [`docs/demo-kzero.yaml`](docs/demo-kzero.yaml)):

![kzero CLI — analyze and dry-run down (terminal recording)](docs/demo.gif)

Regenerate from the repo root: **[docs/README.md — Terminal demo](docs/README.md#terminal-demo-vhs)**.

Declarative **Kubernetes workload** orchestration: ordered **down** / **up** (and **reset**) pipelines from YAML, with phase hooks, optional **per-step** `pre` / `post` scripts, workload scale steps, **Helm releases** (shell scripts or **Helm SDK** chart manifests under **`helm.workspace`**), **`pvc`** / **`exec`** API steps, **`custom:`** scripts, **API watchdog** with throttled progress logs and **`pipeline.stalled`** alerts.

**Operator deployment (bastion-first, out-of-band):** see **[docs/deployment-models.md](docs/deployment-models.md)**. **Scope vs alternatives:** [docs/scope-and-alternatives.md](docs/scope-and-alternatives.md). Playbooks and annotated profiles: **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)** — this repo ships the CLI binary, packages, container image, and Homebrew cask only (same split as [pgwd](https://github.com/hrodrig/pgwd) / [pgwd-selfhosted](https://github.com/hrodrig/pgwd-selfhosted)).

**Related tools (same maintainer):**
- **[pgwd](https://github.com/hrodrig/pgwd)** — PostgreSQL connection watchdog ([live traffic](https://gghstats.hermesrodriguez.com/hrodrig/pgwd); deploy: [pgwd-selfhosted](https://github.com/hrodrig/pgwd-selfhosted))
- **[gghstats](https://github.com/hrodrig/gghstats)** — GitHub repo traffic beyond 14 days ([live demo](https://gghstats.hermesrodriguez.com); deploy: [gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted))
- **[kzero](https://github.com/hrodrig/kzero)** — bastion-first declarative workload reset ([live traffic](https://gghstats.hermesrodriguez.com/hrodrig/kzero); deploy: [kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted))
- **[groot](https://github.com/hrodrig/groot)** — Kubernetes diagnostics archive ([live traffic](https://gghstats.hermesrodriguez.com/hrodrig/groot); deploy: [groot-selfhosted](https://github.com/hrodrig/groot-selfhosted))

**Releases** ([GitHub Releases](https://github.com/hrodrig/kzero/releases)) ship **binaries**, **`.deb`** / **`.rpm`**, **`ghcr.io/hrodrig/kzero`**, and **Homebrew**. **Supply chain (v0.7.0+):** SPDX / CycloneDX SBOMs + Cosign on **`checksums.txt`** and GHCR — see [verify Cosign](#verify-cosign-v070). No Helm charts as release artifacts.

Behavior and acceptance: **[SPECIFICATIONS.md](SPECIFICATIONS.md)**. **Shipped:** **v1.0.0** — stable contract: default **`run.execution: native`** (**#32**), process exit codes **0–4** (**#42**), product kind CI (**#34**), PVC/StatefulSet cookbook (**#33**). Prior: **v0.9.2** (doctor, completion, **`kubectl-kzero`**, retry jitter, JSON Schema); **v0.9.0** (graceful shutdown, `require_delivery`, E2E smoke) — [CHANGELOG.md](CHANGELOG.md). Mitigations: [pipeline-network-loss.md](docs/examples/pipeline-network-loss.md). Diagrams: **[docs/diagrams.md](docs/diagrams.md)**.

## Table of contents

- [README badges](docs/readme-badges.md)
- [Terminal demo](#terminal-demo)
- [Features](#features)
- [Requirements](#requirements)
- [Install or update](#install-or-update)
- [Operator deployment](#operator-deployment)
- [Quick start](#quick-start)
- [First run](#first-run)
- [Usage examples](#usage-examples)
- [Configuration reference](#configuration-reference)
- [Environment and precedence](#environment-and-precedence)
- [Troubleshooting](#troubleshooting)
- [Security note](#security-note)
- [Releases and CI](#releases-and-ci)
- [Verify Cosign (v0.7.0+)](#verify-cosign-v070)
- [Development](#development)
- [Per-step `pre` / `post` (example)](#per-step-pre-post-example)
- [Pipeline order and integrity on `down`](#pipeline-order-and-integrity-on-down)
- [Get involved](#get-involved)
- [License](#license)

---

## Features

- **Configuration-first** (`schema_version: "1.0"`): pipelines and hooks live in config, not hardcoded playbooks.
- **Commands**: `analyze`, `doctor`, `target`, `notify test`, `verify`, `probe`, `down`, `up`, `reset`, `completion`, `version` — global **`--log-format text|json`** and **`--log-level`** on pipeline output (see SPEC). (`reset` = full `down` then `up`; if `down` fails, `up` does not run). Every pipeline command prints a **`Kubernetes target:`** block (`started_at`, optional `client_id`, context, cluster, API server, kubeconfig path) before work starts. **`kzero notify test`** verifies outbound **`notify.*`** channels without running a pipeline. **`analyze`** optionally checks the API that each `deployment` / `statefulset` / **`pvc`** / **`exec`** in the plan exists when kubeconfig loads.
- **Exit codes (stable for wrappers):** **0** success; **1** config; **2** Kubernetes; **3** executor / pipeline abort; **4** notify delivery (`require_delivery` or **`notify test`** POST fail) — [SPEC §5](SPECIFICATIONS.md#process-exit-codes-42).
- **Phase hooks**: `pre-down`, `post-down`, `pre-up`, `post-up`, `on-error` (shell script paths).
- **Per-step hooks** (`pre` / `post`): optional shell scripts for a **single** pipeline step—run immediately before and after that step’s main action; **`post` runs only if the main action succeeds**.
- **Step types**: compact refs (`deployment.ns/name`, `statefulset.ns/name`, `pvc.ns/claim`, `exec.ns/pod`), `release.ns/name` (**Helm SDK** chart manifest **`<release>.yaml`** or legacy **`<release>.sh`** under `helm.workspace`), and `custom: ./script.sh` (with optional sibling `pre` / `post` keys on the same YAML mapping). DaemonSet is **not** supported as a built-in kind because `kubectl scale` rejects it (no `/scale` subresource); use a `custom:` step with `kubectl patch` to set a `nodeSelector` that drains the pods.
- **`run.execution`**: **`native`** (default when omitted — client-go scale + **Helm SDK** + API **`pvc`** / **`exec`**), **`shell`** (opt-in — **`kubectl`** / **`helm`** subprocesses), or **`auto`** (native with shell fallback). On a **bastion**, **`native`** avoids host **`kubectl`** / **`helm`** while staying out-of-band; in-cluster Jobs may use the distroless image — see [deployment-models.md](docs/deployment-models.md) and [SPEC — `run.execution`](SPECIFICATIONS.md#workload-execution-backend-runexecution).
- **Run modes**: `dry-run` (plan only, no cluster mutations) and `live`.

**Libraries** (see [`go.mod`](go.mod)): [Cobra](https://github.com/spf13/cobra) **v1.10.2**, [Viper](https://github.com/spf13/viper) **v1.21.0**.

[↑ Back to top](#top)

## Requirements

Host tooling depends on **`run.execution`** and your pipeline step types (see [SPECIFICATIONS.md](SPECIFICATIONS.md)):

| Path | Host tools |
|------|------------|
| **`run.execution: native`** (default) / **`auto`** | Valid **kubeconfig** (or in-cluster SA); **no host `kubectl`** for scale/wait/**`pvc`**/**`exec`**/**Helm SDK** **`release.*`** |
| **`run.execution: shell`** (opt-in) | **`kubectl`** on `PATH` (or **`command.kubectl`**); **`helm`** when using **`release.*`** shell scripts |
| Phase hooks, **`custom:`**, per-step **`pre`/`post`** | Always **`/bin/sh <script>`** (shebang ignored). Scripts must be **POSIX**/`/bin/sh`-safe — on Ubuntu **`/bin/sh`** is often **dash** (`pipefail` / `[[` fail). See [SPEC — Hook and script interpreter](SPECIFICATIONS.md#hook-and-script-interpreter-binsh). |

- **RBAC** sufficient for the operations in your pipelines (for example **`get`/`patch`/`scale`**, PVC delete, Helm releases, pod exec)
- **Go 1.26.5+** if you [build from source](#quick-start) (`make build`) or use [`go install`](#install-with-go)

[↑ Back to top](#top)

## Install or update

Pre-built **`.deb`**, **`.rpm`**, **`.tar.gz`** (and **`.zip`** on Windows), plus **multi-arch** container images on **`ghcr.io/hrodrig/kzero`**, are on **[GitHub Releases](https://github.com/hrodrig/kzero/releases)** and **[latest release](https://github.com/hrodrig/kzero/releases/latest)**. The **release** badge at the top of this README shows the current tag at a glance.

**Why not a single `latest` URL for every file?** GitHub’s `…/releases/latest/download/<file>` only works if the **asset filename is identical** on every release. GoReleaser here uses the **git tag (with `v`)** in Linux package and archive basenames (for example **`kzero_v1.0.0_linux_amd64.deb`**), while the download path is still `…/download/v1.0.0/…`. **Pick names from the release page**, use the **snippet below**, or use the **badge**.

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

| Format | Example (tag **`v1.0.0`** in the URL path; artifact basename includes the same **`v1.0.0`**) |
|--------|------------------------------------------------------------------|
| **`.deb`** | `curl -fsSL -o /tmp/kzero_v1.0.0_linux_amd64.deb https://github.com/hrodrig/kzero/releases/download/v1.0.0/kzero_v1.0.0_linux_amd64.deb` then `sudo apt install /tmp/kzero_v1.0.0_linux_amd64.deb` |
| **`.rpm`** | `curl -fsSLO https://github.com/hrodrig/kzero/releases/download/v1.0.0/kzero_v1.0.0_linux_amd64.rpm` then `sudo rpm -Uvh kzero_v1.0.0_linux_amd64.rpm` or `sudo dnf install ./kzero_v1.0.0_linux_amd64.rpm` |
| **`.tar.gz` (Linux)** | `curl -fsSLO https://github.com/hrodrig/kzero/releases/download/v1.0.0/kzero_v1.0.0_linux_amd64.tar.gz` then `tar xzf kzero_v1.0.0_linux_amd64.tar.gz` and run **`./kzero`** from the extracted tree (see **`share/examples/kzero/kzero.sample.yml`**) |
| **`.tar.gz` (macOS)** | `curl -fsSLO https://github.com/hrodrig/kzero/releases/download/v1.0.0/kzero_v1.0.0_darwin_amd64.tar.gz` (or **`…_darwin_arm64.tar.gz`** on Apple silicon) |

**Update:** download a newer release and run the same install command again (`rpm -Uvh`, `apt install` over the `.deb`, or replace the tarball tree).

**Windows:** use the **`.zip`** asset for your arch (for example **`kzero_v1.0.0_windows_amd64.zip`**), unpack, run **`kzero.exe`** where **`kubectl`** is available.

**Docker:** `docker pull ghcr.io/hrodrig/kzero:v1.0.0` (match the image tag to the **[release](https://github.com/hrodrig/kzero/releases)** you want). Published images use **`gcr.io/distroless/static-debian13:nonroot`** (static **`kzero`** binary only: no shell, no BusyBox/Alpine runtime). **`Dockerfile`** in this repo uses the same final stage. Package: [ghcr.io/hrodrig/kzero](https://github.com/hrodrig/kzero/pkgs/container/kzero).

**Homebrew** and **BSD packaging** helpers: see **[Install or update](#install-or-update)** and **`contrib/README.md`**.

### Homebrew (macOS / Linux)

```bash
brew install hrodrig/kzero/kzero
```

Tap: **[hrodrig/homebrew-kzero](https://github.com/hrodrig/homebrew-kzero)** (cask updated automatically on each release).

Then run **`kzero --config /etc/kzero/kzero.yaml analyze`** (or follow **[Quick start](#quick-start)** to build from a clone).

### Shell completion

```bash
# Bash / Zsh (current session)
source <(kzero completion bash)
source <(kzero completion zsh)

# Fish (persist)
kzero completion fish > ~/.config/fish/completions/kzero.fish
```

```powershell
kzero completion powershell | Out-String | Invoke-Expression
```

### Install as a kubectl plugin (#52)

Release archives ship **two** binaries from the same `cmd/kzero`: **`kzero`** and **`kubectl-kzero`**. With **`kubectl-kzero`** on `$PATH`, kubectl plugin discovery enables **`kubectl kzero …`**.

```bash
# Homebrew / release tarball already installs both
kubectl plugin list | grep kzero

# Or from a local build
PREFIX=/usr/local make install-kubectl-plugin
kubectl kzero analyze --config ./kzero.yaml
```

`kzero version` / `kubectl kzero version` print the binary label (`kzero` vs `kubectl-kzero`) so logs show which entry point fired.

### Doctor (preflight without mutations)

```bash
kzero doctor --config ./kzero.yaml
kzero doctor --config ./kzero.yaml --output json
```

Checks config load, **`kubectl`/`helm`** on **`PATH`** (when **`run.execution`** is **`shell`** or **`auto`**; skipped under default **`native`**), API reachability, pipeline workload refs, and SelfSubjectAccessReview RBAC hints. Config fail → exit **1**; check errors → exit **2**; warnings allowed with exit **0**.

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

From any machine with Go **1.26.5+** (installs to `$(go env GOPATH)/bin`; ensure that directory is on your `PATH`):

```bash
go install github.com/hrodrig/kzero/cmd/kzero@latest
```

Use a **release tag** instead of `@latest` if you want a pinned version (for example `@v1.0.0`). Module reference: [pkg.go.dev/github.com/hrodrig/kzero](https://pkg.go.dev/github.com/hrodrig/kzero).

**End-to-end operator profile** (maintenance reset: truncate, Helm infra, PVC wipe, **`infra_probe`**, notify): **[kzero-selfhosted — full-reset-example](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/full-reset-example)** with [validation runbook](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/docs/full-reset-validation.md).

[↑ Back to top](#top)

## First run

kzero reads **one YAML file** per invocation (default **`./kzero.yaml`**; after a `.deb`/`.rpm` install use **`kzero --config /etc/kzero/kzero.yaml`**).

1. Start from **[`configs/kzero.sample.yml`](configs/kzero.sample.yml)** (clone, release tarball, **`/etc/kzero/kzero.yaml`**, or **`kzero --print-sample-config > kzero.yaml`**).
2. Keep **`run.mode: dry-run`** until **`kzero analyze`** matches expectations; see [SPECIFICATIONS.md](SPECIFICATIONS.md).

```bash
# From clone or tarball:
cp configs/kzero.sample.yml kzero.yaml
# Homebrew / binary-only install (no configs/ in PATH):
kzero --print-sample-config > kzero.yaml
kzero analyze
kzero down    # dry-run when run.mode: dry-run
```

For **bastion**, **cron**, **CI**, and **live** patterns: **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)** → [run/README.md](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/README.md). For a **full platform reset** playbook (hooks, Helm SDK manifests, transcripts): [full-reset-example](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/full-reset-example).

[↑ Back to top](#top)

## Operator deployment

**Where to run kzero:** **[docs/deployment-models.md](docs/deployment-models.md)** — **bastion / management host (recommended)** for production **`reset`**; in-cluster Job is optional, not the default recovery model.

| Goal | Start here |
|------|------------|
| **Full platform reset** (truncate, PVC, Helm SDK, probe) | [kzero-selfhosted/run/examples/full-reset-example/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/full-reset-example) · [validation runbook](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/docs/full-reset-validation.md) |
| **PVC / StatefulSet data patterns** | [pvc-statefulset-data-strategy.md](docs/examples/pvc-statefulset-data-strategy.md) — scale→wait→`pvc.*`, wipe, snapshot/`custom:`, init |
| **Bastion / cron / systemd** | [kzero-selfhosted/run/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run) — [standalone](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/standalone/README.md), [automation & CI](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/docs/automation-and-pipelines.md) |
| **Network loss during live reset** | [pipeline-network-loss.md](docs/examples/pipeline-network-loss.md) — **`run.api_watchdog`**, notify **`[ERR]`** / **`require_delivery`**, phase-boundary preflight; supplemental mitigations |
| **`docker run`** (analyze / version; live limits) | [run/docker/README.md](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/docker/README.md) |
| **kind e2e** (product CI) | [testing/kind/](testing/kind/) — GHA **`integration-kind`**; full lab: [kzero-selfhosted/testing/kind](https://github.com/hrodrig/kzero-selfhosted/tree/main/testing/kind) |
| **Reference hooks & probe assets** | [run/examples/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples) |

Install the **CLI** here ([Install or update](#install-or-update)); run it from a host with **kubeconfig** and the tools your YAML requires (**`kubectl`** / **`helm`** on the shell path; **`native`** reduces host dependencies — see [Requirements](#requirements)).

[↑ Back to top](#top)

## Usage examples

Paths use **`./bin/kzero`** after `make build`. If you installed from **[Releases](#install-or-update)** or **`go install`**, use **`kzero`** on your `PATH` the same way.

### Plan only (`analyze`, `down` / `up` in `dry-run`)

```bash
kzero analyze --config ./kzero.yaml
# With run.mode: dry-run in YAML:
kzero down --config ./kzero.yaml
```

`**analyze`** validates the profile and prints the normalized plan on stdout: run metadata, phase hooks, indexed **`[down]`** / **`[up]`** steps (including release script paths and per-step options), and a **Deferred** block when unimplemented schema fields are set. See [SPECIFICATIONS.md](SPECIFICATIONS.md) → **`kzero analyze`**.

### Test notifications (no pipeline)

```bash
kzero notify test --config ./kzero.yaml
kzero notify test -c ./kzero.yaml --event pipeline.error
```

Verifies **`notify.*`** channels without contacting the API or running **`down`** / **`up`**. Full cookbook: [docs/examples/notifications.md](docs/examples/notifications.md).

### Readiness verify (post-up)

```bash
kzero verify --config ./kzero.yaml
kzero verify -c ./kzero.yaml --log-format json
```

Set **`run.verify: true`** to run verify automatically after a successful **`up`** or **`reset`** in **`live`** mode.

### Infra probe (pre-destructive gate)

```bash
kzero probe --config ./kzero.yaml
```

Optional **`infra_probe`** runs a throwaway mini-pipeline before **`down`** / **`reset`** in **`live`** mode. Cookbook: [docs/examples/infra-probe.md](docs/examples/infra-probe.md). Reference Redis assets: [kzero-selfhosted/run/examples/infra-probe/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/infra-probe).

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

### Automation and CI/CD

Cron, GitHub Actions, and YES-gated wrappers: **[kzero-selfhosted/run/docs/automation-and-pipelines.md](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/docs/automation-and-pipelines.md)**.

### Version metadata

```bash
kzero version
```

[↑ Back to top](#top)

## Configuration reference

Full schema, validation, and acceptance criteria: **[SPECIFICATIONS.md](SPECIFICATIONS.md)**. Annotated sample: **[configs/kzero.sample.yml](configs/kzero.sample.yml)**. **Editor autocomplete:** [configs/kzero.schema.json](configs/kzero.schema.json) (YAML Language Server `$schema` — see sample header). Runtime validation remains the Go loader.

| Block / key | Purpose |
|-------------|---------|
| **`schema_version`** | Must be **`1.0`** today. |
| **`cluster`** | Metadata (`name`, `environment`, …) for labels and notifications. |
| **`helm`** | **`workspace`**: directory of **`<release>.yaml`** Helm SDK chart manifests (recommended with **`run.execution: native`**) and/or legacy **`<release>.sh`** scripts; optional **`registries`** for OCI login. |
| **`command`** | Optional paths for **`kubectl`** and **`helm`**. |
| **`hooks`** | Optional global scripts: **`pre-down`**, **`post-down`**, **`pre-up`**, **`post-up`**, **`on-error`**. |
| **`notify`** | Optional outbound alerts: **`slack`**, **`discord`**, **`teams`**, **`pagerduty`**, **`webhook`**; **`live`** events **`pipeline.start`**, **`success`**, **`error`**, **`stalled`**. Optional **`require_delivery`** (fail run with exit **4** if error/stalled POST fails). Test with **`kzero notify test`** — [notifications.md](docs/examples/notifications.md). |
| **`pipelines`** | See [`pipelines`](#pipelines) below. |
| **`retry`** | See [`retry`](#retry) below. |
| **`run`** | See [`run`](#run) below. |

### `run`

| Key | Purpose |
|-----|---------|
| **`mode`** | **`dry-run`** (log plan only) or **`live`** (execute steps). Required. |
| **`execution`** | **`native`** (default when omitted), **`shell`** (opt-in), or **`auto`** — see [Features](#features) and SPEC. |
| **`timeout`** | Wall-clock budget for a full **`down`**, **`up`**, or **`reset`** (Go duration, e.g. **`25m`**). |
| **`kubeconfig`** | Path passed to **`kubectl`** / **`helm`**; empty uses the process environment / default kubeconfig search. |
| **`operation_timeout`** | Per-operation ceiling (e.g. **`45s`**) for individual kubectl/helm calls inside a step. |

Pipeline steps always run **sequentially** in YAML order (no parallel execution). On **`down`**, workload steps scale to **0** without waiting for pods to exit unless you add per-step **`post`** (or **`pre`** on the next step). See [Pipeline order and integrity on `down`](#pipeline-order-and-integrity-on-down) and [SPEC — Current engine](SPECIFICATIONS.md#current-engine-sequencing-retry-and-concurrency).

### `retry`

| Key | Purpose |
|-----|---------|
| **`attempts`** | Total tries per pipeline step in **`live`** mode (**integer**, minimum effective **1**). |
| **`delay`** | Base wait before the first retry (**Go duration**, e.g. **`8s`**); wait is **full jitter** in **`[0, delay × 2^(n−1)]`** (capped at **2m**; default base **5s** if `delay` is zero). |

Retries rerun the whole step (per-step **pre**, main, **post**). Only **transient** failures retry (API timeouts, conflicts, 429/503, connection errors). **`NotFound`** / **`Forbidden`** fail immediately. **`dry-run`** never retries. See [SPEC — Current engine](SPECIFICATIONS.md#current-engine-sequencing-retry-and-concurrency).

### `pipelines`

- **`down`** and **`up`** are ordered lists. Each item is either:
  - a **string** step reference: **`deployment.ns/name`**, **`statefulset.ns/name`**, **`pvc.ns/claim`**, **`exec.ns/pod`**, **`release.ns/name`** (DaemonSet is not a built-in kind — see [Features](#features)); or
  - a **single-key map** whose key is one of the above **or** **`custom`**, with optional fields beside that key (see below).
- **`release.*`** steps require **`helm.workspace`**. With **`run.execution: native`** / **`auto`**, use **`<release>.yaml`** chart manifests (Helm SDK); with **`shell`**, use **`<release>.sh`** scripts (see SPEC and [full-reset-example](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/full-reset-example/helm)).

Optional fields on a **map** step (same YAML mapping as the step ref, alongside **`pre`** / **`post`**):

| Field | Applies to | Purpose |
|-------|------------|---------|
| **`pre`** / **`post`** | workload, `release`, **`custom`** | Shell scripts run immediately before / after the main action; **`post`** only if the main action succeeds. |
| **`replicas`** | mainly **`up`** / scale targets | Target replica count (integer). |
| **`wait_for_ready`** | workloads on **`up`** | If **`true`**, wait for rollout / ready after scale-up. **Not** used to wait for pod termination on **`down`**. |
| **`timeout`** | step-level override | Go duration string (e.g. **`10m`**) for that step’s bounded wait / operations. |

[↑ Back to top](#top)

## Environment and precedence

- **`KZERO_`** environment variables override values from the loaded YAML (**Viper** `AutomaticEnv`). Nested keys use underscores (for example **`run.mode`** → **`KZERO_RUN_MODE`**, **`client.id`** → **`KZERO_CLIENT_ID`**).
- **`--config /path/to/kzero.yaml`** selects that file explicitly. When omitted, the default file is **`./kzero.yaml`** (must exist; there is no built-in fallback path).

[↑ Back to top](#top)

## Troubleshooting

### `read config: …` or “cannot find the file”

kzero always loads a real file path. With no **`--config`**, that path is **`./kzero.yaml`** relative to the **current working directory**. **Fix:** `cd` to the directory that contains **`kzero.yaml`**, pass **`--config /absolute/path/kzero.yaml`**, or copy **`configs/kzero.sample.yml`** / **`/etc/kzero/kzero.yaml`** to **`./kzero.yaml`**.

### `run.mode must be one of: dry-run, live`

Only those two literals are accepted (see validation in [`internal/config/load.go`](https://github.com/hrodrig/kzero/blob/main/internal/config/load.go)). Check for typos, quotes, or trailing spaces in YAML.

### `helm.workspace is required when pipelines include release steps`

Any **`release.ns/name`** entry requires **`helm.workspace`** in the root config. Set it to the directory that contains **`<release>.yaml`** (Helm SDK) and/or **`<release>.sh`** (shell path), or remove **`release.*`** steps if you are not using Helm-driven releases.

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

1. Work on **`develop`**; merge to **`main`** when ready.
2. Before tagging: **`make release-check`** (Docker): lint, test, cover ≥80%, govulncheck, Grype.
3. On **`main`**: annotated tag + push → **Release** workflow (GoReleaser, GHCR, Cosign, SBOMs, [homebrew-kzero](https://github.com/hrodrig/homebrew-kzero)).
4. Local: **`make release`** (**main** only). Snapshot: **`make snapshot`** → **`dist/`**.

### Verify Cosign (v0.7.0+)

```bash
# checksums.txt (download assets + .sig / .pem from the GitHub Release)
cosign verify-blob --certificate-identity-regexp 'https://github.com/hrodrig/kzero/.+' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate checksums.txt.pem --signature checksums.txt.sig checksums.txt

# GHCR image (replace TAG)
cosign verify --certificate-identity-regexp 'https://github.com/hrodrig/kzero/.+' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ghcr.io/hrodrig/kzero:TAG
```

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
    - statefulset.data/job-queue:
        pre: ./hooks/pre-scale-purge.sh
```

Here `pre-scale-purge.sh` runs **before** the scale action for `job-queue` (**`native`** API or **`shell`** `kubectl scale` — so the pod can still accept exec). Optional `post` runs **after** a successful scale (or other main action for that step).

## <a id="pipeline-order-and-integrity-on-down"></a>Pipeline order and integrity on `down`

List order defines **when** each step runs, not automatic “wait until pods are gone.” For **`down`**, use a **`post`** hook after the upstream workload if the next step must not start until termination finishes:

```yaml
pipelines:
  down:
    - deployment.app/consumer:
        post: ./hooks/wait-deployment-scale-down.sh
    - deployment.app/producer
```

Reference hook script: [kzero-selfhosted/run/examples/hooks/wait-deployment-scale-down.sh](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/hooks/wait-deployment-scale-down.sh). Full walkthrough: [`docs/examples/pipeline-order-and-integrity.md`](docs/examples/pipeline-order-and-integrity.md).

**Waiting between steps on `up`** (Helm `--wait` in scripts, `post` on `release.*`, `wait_for_ready` on workloads): [`docs/examples/waiting-between-pipeline-steps.md`](docs/examples/waiting-between-pipeline-steps.md).

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
