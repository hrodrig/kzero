# kzero documentation

| Document | Purpose |
|----------|---------|
| [deployment-models.md](deployment-models.md) | **Where to run kzero** — bastion-first (out-of-band); in-cluster optional |
| [scope-and-alternatives.md](scope-and-alternatives.md) | **When to use kzero** vs Helm, GitOps, scripts, provisioning, DR (by scope) |
| [../SPECIFICATIONS.md](../SPECIFICATIONS.md) | Behavior contract, config shape, and TDD baseline |
| [../ROADMAP.md](../ROADMAP.md) | Prioritized planned work and known gaps (in-repo source of truth) |
| [plan-1.0.0.md](plan-1.0.0.md) | **Draft** — stable contract (**#32–#34**, **#42**); after **v0.9.2** |
| [plan-1.1.0.md](plan-1.1.0.md) | **Draft** — post-1.0 bounded: hook interpreter, **#29**, resume-from-step |
| [plan-0.9.x.md](plan-0.9.x.md) | **Done** — **v0.9.0**–**v0.9.2** bastion-first + stretch (**#43–#53**) |
| [plan-0.6.0.md](plan-0.6.0.md) | Implementation plan for **v0.6.0** (notify, preflight, verify, infra probe, slog) |
| [plan-0.8.x.md](plan-0.8.x.md) | **Done** — **0.8.x** band shipped in **v0.8.0** (API watchdog, notify delivery visibility, reset phase-boundary preflight) |
| [diagrams.md](diagrams.md) | Mermaid diagrams: CLI vs engine, phase ordering, `reset`, failure path |
| [demo-kzero.yaml](demo-kzero.yaml) | Minimal YAML used by [demo.tape](demo.tape) (VHS) |
| [demo.tape](demo.tape) | VHS tape to record [demo.gif](demo.gif) |
| [examples/pipeline-order-and-integrity.md](examples/pipeline-order-and-integrity.md) | Ordered `down` steps, `wait_for_ready` limits, per-step `post` examples |
| [examples/notifications.md](examples/notifications.md) | **notify** channels, `kzero notify test`, env overrides, live vs dry-run |
| [examples/pipeline-network-loss.md](examples/pipeline-network-loss.md) | Long live **`reset`** on bastions: two-phase outage pattern, **v0.8.0** engine features + supplemental mitigations |
| [examples/infra-probe.md](examples/infra-probe.md) | **`infra_probe`** schema, checks, gate (`kzero probe`) |
| [examples/waiting-between-pipeline-steps.md](examples/waiting-between-pipeline-steps.md) | Wait between steps: Helm `--wait`, `post` on `release.*`, `wait_for_ready` |
| [examples/automation-and-pipelines.md](examples/automation-and-pipelines.md) | Stub → operator CI/cron docs in **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)** |
| [examples/hooks/README.md](examples/hooks/README.md) | Stub → reference hook scripts in **kzero-selfhosted** |

**Operator deployment** (bastion, cron, docker run, kind e2e, hook scripts): **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)** → [run/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run).

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
