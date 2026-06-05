# kzero documentation

| Document | Purpose |
|----------|---------|
| [SPECIFICATIONS.md](SPECIFICATIONS.md) | Behavior contract, config shape, and TDD baseline |
| [ROADMAP.md](ROADMAP.md) | Prioritized planned work and known gaps (in-repo source of truth) |
| [diagrams.md](diagrams.md) | Mermaid diagrams: CLI vs engine, phase ordering, `reset`, failure path |
| [demo-kzero.yaml](demo-kzero.yaml) | Minimal YAML used by [demo.tape](demo.tape) (VHS) |
| [demo.tape](demo.tape) | VHS tape to record [demo.gif](demo.gif) |
| [examples/pipeline-order-and-integrity.md](examples/pipeline-order-and-integrity.md) | Ordered `down` steps, `wait_for_ready` limits, per-step `post` examples |
| [examples/automation-and-pipelines.md](examples/automation-and-pipelines.md) | CI/cron: live mode, env overrides, auto-confirm for YES-gated wrappers |
| [examples/waiting-between-pipeline-steps.md](examples/waiting-between-pipeline-steps.md) | Wait between steps: Helm `--wait`, `post` on `release.*`, `wait_for_ready` |
| [examples/hooks/wait-helm-release-ready.sh](examples/hooks/wait-helm-release-ready.sh) | Reference `post` hook after Helm install (release steps on `up`) |
| [examples/hooks/wait-master-ready.sh](examples/hooks/wait-master-ready.sh) | Reference `pre` hook: slave StatefulSet waits for master Deployment |
| [examples/hooks/wait-deployment-scale-down.sh](examples/hooks/wait-deployment-scale-down.sh) | Reference `post` hook after scale-to-zero on a Deployment |

Diagrams render on GitHub and in most Markdown viewers that support **Mermaid**.

## Terminal demo (VHS)

Short recording of `kzero --help`, `kzero version`, and `kzero analyze` using the minimal [demo-kzero.yaml](demo-kzero.yaml).

### Prerequisites

- [VHS](https://github.com/charmbracelet/vhs): e.g. `brew install vhs`
- `kzero` on `PATH`: from repo root run `make build` and `export PATH="$(pwd)/bin:$PATH"`, or `make install` (installs to `$GOBIN`)

### Render the demo

From the **repository root**, run **both** parts below. **Do not** run `vhs docs/demo.tape` alone from an interactive **zsh** session with Oh My Zsh: VHS inherits that shell and the recording can break (same guidance as [pgwd](https://github.com/hrodrig/pgwd) `docs/README.md`).

```bash
make build
export PATH="$(pwd)/bin:$PATH"
bash -c "vhs docs/demo.tape"
```

Or after `make install`:

```bash
bash -c "vhs docs/demo.tape"
```

Output: **`docs/demo.gif`** (path set by `Output` in `demo.tape`).

### When to re-record

Re-run the same commands after **`VERSION`** or visible CLI output changes, so the GIF stays accurate for README or release notes.
