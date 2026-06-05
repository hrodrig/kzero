# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Preflight** (**0.6.x #19**): live **`down`** / **`up`** / **`reset`** verify Kubernetes API reachability before phase hooks; dry-run plan line; **`analyze`** stderr warning when preflight would fail.
- **Helm workspace contract** (**0.6.x #22bis**): SPEC documents flat **`<helm.workspace>/<release>.sh`**, env vars, and operator responsibilities before **0.7.x** Helm SDK.
- **`kzero probe`** / **`infra_probe`** (**0.6.x #22**): declarative mini-pipeline (up → checks → down); optional gate before **`down`** / **`reset`** in live mode; checks **`pvc_bound`**, **`release_ready`**; result cache via **`cache_ttl`** and **`run.probe_cache_dir`**.
- **Docs:** [docs/examples/infra-probe.md](docs/examples/infra-probe.md) — probe goals (registry, creds, volumes), gate, cache; anonymous Redis reference under [docs/examples/infra-probe/](docs/examples/infra-probe/) (operator may use it or supply any chart).
- **`kzero verify`** (**0.6.x #21**): post-up readiness checks (`workloads_ready`, `nodes_ready`); text/JSON report; **`run.verify`** auto-runs after successful **`up`** / **`reset`** in live mode.
- **`--log-format text|json`** (**0.6.x #16**): structured JSON lines for engine events (`live`, `dry-run`, `retry`) and command summary; text mode preserves legacy `[live]` / `[dry-run]` lines.
- **Operator audit** (**0.6.x #20**): **`Kubernetes target:`** prints **`os_user`** / **`os_uid`**; hooks and subprocesses receive **`KZERO_OS_USER`** / **`KZERO_OS_UID`**.
- **Multi-channel `notify`** (**0.6.x #18**): **`slack`**, **`discord`**, **`teams`**, **`pagerduty`** (Events API v2), and generic **`webhook`**; events **`pipeline.start`**, **`pipeline.success`**, **`pipeline.error`** in **`live`** mode.
- **`kzero notify test`**: POST to enabled channels without running a pipeline; optional **`--event`** for payload previews.
- **Docs:** [docs/examples/notifications.md](docs/examples/notifications.md) — channel setup, env overrides, test-then-live workflow.

### Changed

- **`notify.*`**: no longer emits deferred-feature warnings when enabled.
- **0.6 plan:** internal merge order notify → slog → verify → infra probe; Helm workspace SPEC (**#22bis**) before **0.7.x** Helm SDK.
- **`make clean`**: removes **`coverage.out`** (artifact stays gitignored).

## [0.5.7] - 2026-06-05

### Added

- **Subprocess error taxonomy** (**0.5.x #15**, band closed): `WrapSubprocess` classifies kubectl/helm/hook failures by exit code and stderr patterns; stable sentinels **`ErrNotFound`**, **`ErrForbidden`**, **`ErrTransient`** for `errors.Is` and per-step retry.

### Changed

- **Shell path** (scale, rollout wait, helm uninstall, release scripts, hooks) returns classified errors instead of raw `exec` messages.

## [0.5.6] - 2026-06-05

### Changed

- **CLI:** `down`, `up`, and `reset` share `buildPipelineCmd` (less duplication).
- **Engine:** document `LiveRunner` workload cache thread-safety (one runner per invocation, sequential steps).

### Added

- **Tests:** raise `internal/cli`, `internal/cluster`, and `internal/engine` coverage above 80%; total statement coverage **84%**.
- **Ports:** sync FreeBSD and OpenBSD port Makefiles to **0.5.6**.

## [0.5.5] - 2026-06-05

### Added

- **`make release-check`** now runs **`cover-check`** (minimum statement coverage gate, default 80%).
- **CLI integration test** `TestClientID_e2eAnalyzeAndDownDryRun`: `client.id` on `analyze` stdout and `client_id=` in `down` dry-run logs.
- **`[live]`** structured action lines (scale, rollout wait, Helm uninstall, release scripts, hooks) for operator visibility.
- **`client.id` audit:** **`client_id:`** in the **`Kubernetes target:`** block (once per command); **`[dry-run]`** / **`[retry]`** lines include **`client_id=`**; **`KZERO_CLIENT_ID`** in hook/script environments.
- **`Kubernetes target:`** includes **`started_at:`** (RFC3339) on pipeline commands (`target`, `down`, `up`, `reset`).
- **Docs:** [docs/examples/automation-and-pipelines.md](docs/examples/automation-and-pipelines.md) — CI/cron, live mode, auto-confirm for YES-gated wrappers.
- **Docs:** [docs/examples/waiting-between-pipeline-steps.md](docs/examples/waiting-between-pipeline-steps.md) — Helm `--wait`, `post` on `release.*`, `wait_for_ready`, master/slave `pre` hook.
- **Reference hooks:** [wait-helm-release-ready.sh](docs/examples/hooks/wait-helm-release-ready.sh), [wait-master-ready.sh](docs/examples/hooks/wait-master-ready.sh).

### Fixed

- **`KZERO_RELEASE_NAME`** / **`KZERO_RELEASE_NAMESPACE`** on per-step `pre`/`post` hooks for `release.*` steps.
- **[wait-helm-release-ready.sh](docs/examples/hooks/wait-helm-release-ready.sh):** use `kubectl rollout status` / `kubectl wait` (`helm status` has no `--wait`).
- **Docs:** CHANGELOG compare links for 0.5.2–0.5.4; SPEC `release` down documents `helm uninstall` (0.5.4 behavior).

## [0.5.4] - 2026-06-04

### Added

- **`client.id` propagation:** when set, engine log lines (`[dry-run]`, `[retry]`, native dry-run) include a **`client_id=`** field; hook, custom, and release scripts receive **`KZERO_CLIENT_ID`** in their environment. Override via **`KZERO_CLIENT_ID`** on config load. Roadmap **#14** done.
- **Docs:** [docs/examples/pipeline-order-and-integrity.md](docs/examples/pipeline-order-and-integrity.md) and reference hook [docs/examples/hooks/wait-deployment-scale-down.sh](docs/examples/hooks/wait-deployment-scale-down.sh); README and SPEC clarify sequential `down` order vs pod termination; sample profile includes `deployment.app/consumer` → `producer` example.

### Changed

- **`release.*` on `down`:** live mode runs **`helm uninstall <release> -n <namespace> --wait --ignore-not-found`** instead of executing `<helm.workspace>/<release>.sh`. **`up`** still runs the install script. `kzero analyze` shows `helm uninstall` in the down plan.

## [0.5.3] - 2026-06-04

### Removed

- **`run.worker_concurrency`** removed from the configuration contract. The engine always runs pipeline steps **sequentially** in YAML order. Legacy YAML keys are ignored; CLI warnings for `worker_concurrency` are dropped. Roadmap item **0.5.x #13** (pipeline parallelism) is **closed** as out of scope.

## [0.5.2] - 2026-06-04

### Added

- **Per-step retry in live mode**: `retry.attempts` and `retry.delay` are honored for each pipeline step (pre + main + post). Transient failures (API timeout/conflict/429/503, rollout deadlines, common connection errors) retry with backoff **`delay × 2^(n−1)`** (max **2m**). Non-retriable: `NotFound`, `Forbidden`, `context.Canceled`. **`dry-run`** unchanged (no retries). Logs `[retry] pipeline …` before each wait.

## [0.5.1] - 2026-06-04

Pilot polish after **0.5.0**: stronger dry-run on the native path and configurable timing-line colors.

### Added

- **Server-side dry-run for native scale**: with `run.mode: dry-run` and `run.execution: native` or `auto`, `deployment` / `statefulset` steps call the API with `DryRun=All` (validates RBAC and object state without persisting replica changes). Shell execution and `release` / `custom` steps remain plan-only.
- **Colored elapsed time** on command summary lines: green when the command succeeds, yellow when it fails. Controlled by **`run.color`** in YAML (`auto`, `always`, `never`; default `auto`), overridable with **`KZERO_RUN_COLOR`** / legacy **`KZERO_COLOR`** when `auto`, plus **`NO_COLOR`** and **`FORCE_COLOR`**. Summary is written to **stderr** so `kzero down | tee log` still shows colors on the terminal when `run.color: always`.

## [0.5.0] - 2026-06-04

First **pilot-ready** operator release: safe cluster identification, env overrides, and run timing for live Helm/script pipelines.

### Added

- **`Kubernetes target:`** block on **`analyze`**, **`down`**, **`up`**, and **`reset`**: prints resolved **context**, **cluster**, **namespace**, **api_server**, and **kubeconfig** path before any pipeline work (avoids hitting the wrong environment).
- **`kzero target`**: print only the Kubernetes target block for a config file.
- **Elapsed time** on every command: final line `kzero <command> finished in …` or `failed after …`.
- **`KZERO_*` env overrides** for `run.mode`, `run.kubeconfig`, and related keys now apply on config load (`BindEnv`).

## [0.4.1] - 2026-06-04

### Added

- **`kzero analyze` cluster validation**: when kubeconfig loads, **Get** checks for each unique `deployment` / `statefulset` ref; **FAIL** lines and non-zero exit if missing or not scalable; **stderr** skip note when the API client cannot be built.

## [0.4.0] - 2026-06-04

### Added

- **`run.execution`**: `shell` (default), `native` (client-go), or `auto` (native with shell fallback) for `deployment` / `statefulset` steps in live mode.
- **`internal/executor`**: `Shell` and `Native` workloads; typed API error sentinels; fake-clientset tests.

### Changed

- **Docs:** [docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md) documents workload execution backend; [docs/ROADMAP.md](docs/ROADMAP.md) marks 0.4.x items 5–9 shipped in **0.4.0**.
- **Dependencies:** `k8s.io/client-go` and `k8s.io/api` for native execution; `golang.org/x/net` v0.55.0 (govulncheck clean with client-go).

## [0.2.3] - 2026-06-04

### Added

- **CLI warnings** (stderr) after successful config load when **`run.worker_concurrency` > 1**, **`retry.attempts` > 1**, or **`notify.slack.enabled` / `notify.discord.enabled`** is true — those fields are accepted by the schema but not yet implemented by the v1 engine (`analyze`, `down`, `up`, `reset`).
- **`kzero analyze`**: prints normalized **`[down]`** / **`[up]`** step lists (with optional script paths, hooks, and scale options), phase hooks, run metadata, and a **Deferred** summary on stdout for schema fields not yet honored by the engine.

### Changed

- **Docs:** [docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md) and [README.md](README.md) describe the **`kzero analyze`** stdout/stderr contract.
- **Roadmap:** strategic priority for **client-go** / native executor (**0.4.x**); **0.3.x** operator-honesty band complete.
- **Go toolchain:** `go.mod` and build images require **Go 1.26.4+** (stdlib).

### Security

- Bump minimum Go version to **1.26.4** to address [GO-2026-5037](https://pkg.go.dev/vuln/GO-2026-5037) (`crypto/x509` hostname parsing) reported by `govulncheck` on Go 1.26.3.

## [0.2.2] - 2026-05-13

### Added

- **`docs/ROADMAP.md`**: in-repo prioritized roadmap (semver bands **0.3.x**–**1.0.0 (future)**) plus **Shipped** for **0.2.0** / **0.2.1**; linked from **README** and **docs/README**.

### Changed

- **`docs/ROADMAP.md`**: rename milestone headings from fictional `v1.0.x` / `v2.0` style labels to **0.3.x**–**1.0.0 (future)** bands aligned with public releases starting at **0.2.0**.

### Fixed

- **`CHANGELOG.md`**: restore the **`[0.2.1]`** section (Removed / Changed) that had been folded under **`[Unreleased]`** by mistake, so published **0.2.1** release notes match the tagged release again.

## [0.2.1] - 2026-05-13

### Removed

- `daemonset.<namespace>/<name>` step kind in `pipelines.down` / `pipelines.up`. The v1 engine ran `kubectl scale daemonset/...`, which the Kubernetes API server rejects because DaemonSet has no `/scale` subresource (`Error from server (NotFound)`). Configs that reference `daemonset.*` now fail at parse time. **Migration:** replace those steps with a `custom: ./hooks/<name>.sh` step that runs `kubectl patch daemonset ... --type=strategic -p '{"spec":{"template":{"spec":{"nodeSelector":{"kzero.io/disabled":"true"}}}}}'` to drain the pods (and a matching script on `up` that removes the nodeSelector key). See `docs/SPECIFICATIONS.md` → *Supported workload kinds*.

### Changed

- `pipelines.{down,up}` reject unsupported step kinds at config load time via an explicit allow-list (`deployment`, `statefulset`, `release`). Previously, refs such as `cronjob.<ns>/<name>`, `job.<ns>/<name>`, or `service.<ns>/<name>` passed validation and failed only later in live mode with `unsupported pipeline resource type`. `kzero analyze` now surfaces the problem before any cluster mutation.

## [0.2.0] - 2026-05-13

### Added

- **GNUmakefile** (pgwd-style): `help`, `build`, `build-all`, `install`, `clean`, `test`, `cover`, `lint` (includes **gocyclo** ≤14), `tools`, `security` (govulncheck), `docker-build`, `docker-scan`, **`release-check`**, **`release`**, **`snapshot`**.
- **GoReleaser** (`.goreleaser.yaml`), **Dockerfile** / **Dockerfile.release**, **GitHub Actions**: `ci.yml`, `security.yml`, `release.yml`.
- **`SECURITY.md`**, **`CONTRIBUTING.md`**, **`tools/scan.sh`**, **`codecov.yml`**, **`kzero version`** command.
- **Linux packages**: GoReleaser **nfpm** `.deb` / `.rpm` (plus existing **`archives`** `tar.gz` / `zip`), **`contrib/deb/`** maintainer scripts, **`contrib/README.md`**.

### Changed

- **Makefile** is a FreeBSD-friendly stub that forwards to **gmake** / **GNUmakefile** (same pattern as pgwd).

[Unreleased]: https://github.com/hrodrig/kzero/compare/v0.5.7...HEAD
[0.5.7]: https://github.com/hrodrig/kzero/compare/v0.5.6...v0.5.7
[0.5.6]: https://github.com/hrodrig/kzero/compare/v0.5.5...v0.5.6
[0.5.5]: https://github.com/hrodrig/kzero/compare/v0.5.4...v0.5.5
[0.5.4]: https://github.com/hrodrig/kzero/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/hrodrig/kzero/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/hrodrig/kzero/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/hrodrig/kzero/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/hrodrig/kzero/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/hrodrig/kzero/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/hrodrig/kzero/compare/v0.2.3...v0.4.0
[0.2.3]: https://github.com/hrodrig/kzero/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/hrodrig/kzero/releases/tag/v0.2.2
[0.2.1]: https://github.com/hrodrig/kzero/releases/tag/v0.2.1
[0.2.0]: https://github.com/hrodrig/kzero/releases/tag/v0.2.0
