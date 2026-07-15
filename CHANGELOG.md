# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Exit codes (#42):** stable process codes **0–4** via [`internal/exitcode`](internal/exitcode) (groot-style wrap): config **1**, Kubernetes **2**, executor **3**, notify delivery **4**; SPEC §5 + man `EXIT STATUS`.
- **Docs (#33):** [docs/examples/pvc-statefulset-data-strategy.md](docs/examples/pvc-statefulset-data-strategy.md) — PVC / StatefulSet data patterns (scale→wait→delete, wipe, snapshot/`custom:`, init); linked from SPEC/README.
- **Planning:** [docs/plan-1.0.0.md](docs/plan-1.0.0.md) — **1.0.0** stable-contract band (**#32–#34**, **#42**); PR order and success criteria.
- **Planning:** [docs/plan-1.1.0.md](docs/plan-1.1.0.md) — bounded post-1.0 band (**#56** hook interpreter, **#29**, **#57** resume-from-step); parked ideas listed explicitly.

### Changed

- **SPEC:** document that hooks / **`custom:`** / shell **`release.*`** scripts always run as **`/bin/sh <path>`** (shebang ignored); must be POSIX-safe (Ubuntu **dash** vs bashisms such as **`pipefail`**).

### Fixed

- **CLI tests:** drop `t.Parallel` from completion tests that call `newRootCmd` — global Viper/`cobra.OnInitialize` raced under CI `go test -race`.

## [0.9.2] - 2026-07-11

### Added

- **JSON Schema (#51):** **[configs/kzero.schema.json](configs/kzero.schema.json)** for editor autocomplete (`schema_version` **1.0**); sample YAML Language Server hint; SPEC/README notes (runtime validation remains the Go loader).
- **Shell completion (#53):** **`kzero completion <bash|zsh|fish|powershell>`** — strict shell arg validation (invalid/missing → non-zero), script on stdout; README one-liners and man page entry.
- **`kubectl-kzero` plugin (#52):** second release binary (same `cmd/kzero`) for **`kubectl kzero …`**; GoReleaser archives/packages/Homebrew; **`make install-kubectl-plugin`**; version line shows entry-point label.
- **`kzero doctor` (#49):** config + binary PATH checks + API ping + pipeline workload existence + SelfSubjectAccessReview RBAC hints; **`--output text|json`**; non-zero exit on errors.

### Changed

- **Retry backoff (#50):** live per-step retry waits use **full jitter** uniformly in **`[0, exponential]`** (still capped at **2m**) so concurrent runs do not align on the same delay.
- **Docs (#48):** README trim + Cosign verify examples; retry table documents full jitter.

## [0.9.1] - 2026-07-11

### Security

- **Dependencies:** bump transitive `oras.land/oras-go/v2` to **v2.6.2** (Helm OCI) — closes Dependabot **#6** / [GHSA-fxhp-mv3v-67qp](https://github.com/advisories/GHSA-fxhp-mv3v-67qp) ([CVE-2026-50163](https://www.cve.org/CVERecord?id=CVE-2026-50163)); Grype oras ignore removed (patched).
- **`make security`:** document **GO-2026-5932** (Helm transitive `openpgp`) in `.govulncheck-ignore.yaml`; containerd v2-only entries unchanged.

### Changed

- **Go toolchain:** bump minimum Go to **1.26.5** (`go.mod`, `Dockerfile`) — addresses [GO-2026-4970](https://pkg.go.dev/vuln/GO-2026-4970) and [GO-2026-5856](https://pkg.go.dev/vuln/GO-2026-5856) (stdlib) reported by Grype on `golang:1.26.4` images.
- **Docker:** final stage `gcr.io/distroless/static-debian13:nonroot` (`Dockerfile`, `Dockerfile.release`) — Debian 12 base EOL.

### Fixed

- **Validate client factory:** replace mutable **`DefaultClientFactory`** with thread-safe **`ClientFactoryDefault()`** / **`SwapDefaultClientFactory()`**.
- **Engine preflight tests:** inject **`Engine.PreflightFactory`** so parallel live **`RunDown`** tests do not clobber the process-wide factory before the pipeline step starts.

## [0.9.0] - 2026-07-08

### Added

- **E2E smoke in CI (#45):** `testing/smoke/smoke.sh` — build, `analyze`, dry-run `down`, `--print-sample-config`; GitHub Actions job `e2e-smoke`.
- **Watchdog mid-wait tests (#46):** engine integration tests with fake API server (`watchdog_midwait_test.go`, `live_progress_test.go`) — API unreachable during blocking step; throttled progress on context cancel.
- **SPEC contract index (#47):** operator-facing implemented / out-of-scope / deferred table in [SPECIFICATIONS.md](SPECIFICATIONS.md) §2.1.
- **Graceful shutdown (0.9.x #44):** **`SIGINT`** / **`SIGTERM`** cancel **`down`** / **`up`** / **`reset`** / **`probe`** pipeline context; engine logs last hook or step on user interrupt (distinct from API watchdog **`pipeline.stalled`**).
- **`kzero target --output slug` (0.9.x):** print a filesystem-safe cluster slug for wrapper log filenames (`kzero-<cmd>-<slug>-<timestamp>.log`).
- **`notify.require_delivery` engine wiring (0.9.x #43):** when **`true`**, failed **`pipeline.error`** or **`pipeline.stalled`** notify POST fails the pipeline (wraps the original error); removed from **`kzero analyze`** Deferred summary.
- **Operator docs:** [docs/deployment-models.md](docs/deployment-models.md) — bastion-first (out-of-band) deployment model; in-cluster Job documented as optional, not recommended for destructive **`reset`** when API reliability is uncertain.
- **Planning:** [docs/plan-0.9.x.md](docs/plan-0.9.x.md) — **0.9.x** band (**#43–#51**) bastion-first hardening; ROADMAP strategic direction updated.

### Changed

- **Man page:** `contrib/man/man1/kzero.1` `.TH` version synced to **v0.9.0**; `make release-check` fails on mismatch with **`VERSION`**.
- **ROADMAP / SPEC / README:** cross-links to deployment models; **`run.execution: native`** framed for bastions and optional in-cluster packaging.
- **docs/deployment-models.md:** quick decision matrix, stronger in-cluster warning, bastion **`docker run`** example, decision flowchart.
- **docs/plan-0.9.x.md:** priority tiers (0.9.0 vs 0.9.1), no-regression criterion for in-cluster, PR order (**#44** before **#43**).
- **Dependencies:** bump transitive `oras.land/oras-go/v2` to **v2.6.1** (Helm OCI); Grype ignore for **GHSA-fxhp-mv3v-67qp** until upstream **v2.6.2** (see `.grype.yaml`).
- **`make security`:** govulncheck known false positives (containerd v2-only CRI checkpoint on v1 module) filter to **PASS** with pending-upstream note (`.govulncheck-ignore.yaml`).
- **CI / release gates:** Grype uses `.grype.yaml`; `security.yml` passes `-c .grype.yaml`; job `e2e-smoke` in `ci.yml`.

### Fixed

- **API watchdog client:** probe `/healthz` via `rest.HTTPClientFor` and a direct GET (previously `rest.RESTClientFor` required `GroupVersion` / `NegotiatedSerializer` and disabled the watchdog silently with a normal kubeconfig).

## [0.8.1] - 2026-06-29

### Fixed

- **`kzero analyze` deferred warning**: remove incorrect Deferred message for **`run.api_watchdog.enabled`** now that the engine watchdog ships in **v0.8.0**; **`notify.require_delivery`** warning text updated to match current behavior (**#35** logs **`[ERR]`**; pipeline fail-fast on delivery error still deferred).

### Changed

- **Docs:** post-**v0.8.0** sync for [pipeline-network-loss.md](docs/examples/pipeline-network-loss.md), [docs/README.md](docs/README.md), and related operator cookbooks (no longer “until **0.8.0**”).

## [0.8.0] - 2026-06-29

### Added

- **API watchdog** (`run.api_watchdog`): periodic Kubernetes API reachability check during live `down`/`up`/`reset`. Configurable `interval`, `fail_after`. When cumulative unreachability exceeds `fail_after`, the running pipeline step is cancelled and `pipeline.stalled` is dispatched (**#36**, **#39**).
- **`pipeline.stalled` notify event**: distinct event type for API watchdog trips (vs step failure). Separate Slack color (dark orange) and title. Support in `kzero notify test --event stalled` (**#41**).
- **Reset phase-boundary preflight**: after `down` completes and before `up` begins in `kzero reset`, re-run the API preflight check (**#37**).
- **Throttled progress logs**: during long waits (rollout wait, Helm install/uninstall, release scripts), emit `[INF]` lines every 30s: step ref, elapsed time (**#38**).
- **`run.api_watchdog` and `notify.require_delivery` schema**: YAML parsing + `KZERO_RUN_API_WATCHDOG_*` / `KZERO_NOTIFY_REQUIRE_DELIVERY` env overrides. Surfaced in `kzero analyze` Deferred summary until the engine fully wires `require_delivery` (**#39**).

### Fixed

- **Notify dispatch failures now logged**: `_ = notify.Dispatch(...)` replaced with error handling across all three call sites (CLI `EventStart`/`EventSuccess`, engine `EventError`). Failed notify POSTs produce `[ERR]` log lines with redacted webhook URLs (**#35**).

## [0.7.4] - 2026-06-16

### Added

- **`kzero --print-sample-config`** — writes sample YAML to stdout (same as `configs/kzero.sample.yml`); documented in README and SPECIFICATIONS.
- **Planning:** [docs/plan-0.8.x.md](docs/plan-0.8.x.md) — **0.8.x** band (API watchdog, notify delivery visibility, reset phase preflight) from production network-loss incident learnings; [docs/examples/pipeline-network-loss.md](docs/examples/pipeline-network-loss.md) operator mitigations until **0.8.0**.

### Changed

- **README:** install examples and Docker pin updated to **v0.7.4**; document **Helm SDK** / **`run.execution: native`**, **`pvc`** / **`exec`** steps, and link to **kzero-selfhosted** [full-reset-example](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/full-reset-example).
- **ROADMAP:** **0.8.x** band **#35–#41**; current focus shifted from **1.0.0** prep.

### Fixed

- **Docker image build:** include **`configs/`** in build context so embedded sample YAML compiles in CI **`docker-scan`**.

## [0.7.3] - 2026-06-12

### Added

- **Text log levels:** timestamped lines `YYYY/MM/DD HH:MM:SS: kzero - [DBG|INF|WRN|ERR] - …`; filter with **`--log-level`** (default **`info`**); documented in **SPECIFICATIONS** (Microsoft LogLevel subset).
- **Slack notify attachments:** colored sidebar (blue start, green completed, red error, yellow test), title **`kzero {action}`**, fields (`Cluster`, `Client`, `Time`, `Context`, `User`, `Mode`, `Duration` on success), footer **`kzero vX.Y.Z`** from build metadata; **`kube_context`** in generic webhook JSON payloads.
- **`KZERO_NOTIFY_*` env binding** for all notify channel keys (Slack, Discord, Teams, PagerDuty, webhook).
- **`notify.AppVersion`** ldflags in Makefile, Dockerfile, and GoReleaser for footer/version stamping.

### Changed

- **Helm SDK OCI:** registry login before chart pull; improved credential resolution for private OCI charts.

## [0.7.2] - 2026-06-10

### Added

- **Helm SDK (#25):** when **`run.execution`** is **`native`** or **`auto`**, **`release.*`** steps use **`helm.sh/helm/v3`** for uninstall (down) and **`upgrade --install`** (up) from **`<helm.workspace>/<release>.yaml`** or step **`chart`** / **`version`** overrides — no host **`helm`** binary or **`.sh`** install script on the SDK path.
- **`pvc` step (#24):** compact ref **`pvc.<namespace>/<claim>`** deletes a named PersistentVolumeClaim via the Kubernetes API (`DeletePropagationBackground`, ignore-not-found); always native regardless of **`run.execution`**.
- **`exec` step (#23):** compact ref **`exec.<namespace>/<pod>`** with required **`container`** and **`command`** runs a command in the pod via **remotecommand**; optional **`stdin`** and per-step **`timeout`**; always native regardless of **`run.execution`**.
- **Infra probe (native #26):** reference Redis probe for **`run.execution: native`** using Helm SDK chart manifest (**`probe-redis.yaml`**) instead of **`.sh`** — see [docs/examples/infra-probe/](docs/examples/infra-probe/).
- **`pods_schedulable` verify check (#27):** optional post-up check for pods **Pending** with **`Unschedulable`** scheduling failures (affinity, taints, node selectors) in namespaces from **`pipelines.up`**.
- **`pods_schedulable` infra probe check:** same scheduling sanity for **`infra_probe.pipeline.up`** namespaces.
- **OCI registry login:** **`helm.registries`** with **`host`**, **`username`**, and **`password`** or **`password_env`** — Helm SDK logs in before **`oci://`** chart pulls (**0.7.x #25** follow-up).
- **Release `script:` override (#31):** optional non-flat install script path relative to **`helm.workspace`** (default **`<name>.sh`** unchanged).
- **`custom:` parity (#30):** main custom pipeline scripts receive **`KZERO_PHASE`**, **`KZERO_PIPELINE_STEP_INDEX`**, **`KZERO_STEP_HOOK=main`**, **`KZERO_STEP_CUSTOM`**, and related step metadata (same as per-step hooks).
- Dependency: **`helm.sh/helm/v3`** v3.21.x; **`k8s.io/*`** client modules aligned to **v0.35.1** for Helm SDK compatibility.

### Changed

- **0.7.x band close** on **`develop`**: bundles PR3–PR7 (**Helm SDK**, **`pvc`**, **`exec`**, probe native, scheduling, OCI auth, helm path ergonomics, custom env parity). **`#29`** (`job`/`cronjob`) remains open on the roadmap.

## [0.7.1] - 2026-06-10

### Added

- **In-cluster auth:** when **`run.kubeconfig`** is empty, **`LoadRESTConfig`** tries default kubeconfig discovery, then **`rest.InClusterConfig()`** (Pod service account). **`Kubernetes target:`** shows an **`in-cluster`** audit block; pipeline step namespaces still come from compact refs.
- **Secret redaction (#17):** `internal/redact` scrubs bearer tokens, webhook URLs, and common `*_TOKEN` / `*_KEY` env patterns in engine logs, notify error payloads, and subprocess output.
- **`run.no_env_passthrough`** and **`--no-env-passthrough`** on pipeline commands — hooks and kubectl subprocesses receive only **`KZERO_*`**, optional **`KUBECONFIG`**, and correlation fields.

## [0.7.0] - 2026-06-10

### Added

- **Supply chain (GoReleaser):** **Cosign** keyless signing for **`checksums.txt`** and **`ghcr.io/hrodrig/kzero`** images; **SPDX** and **CycloneDX** SBOMs per release (Syft, source catalog). Release workflow installs **cosign** and **syft** and grants **`id-token: write`** for OIDC signing (same pattern as [groot](https://github.com/hrodrig/groot)).

## [0.6.2] - 2026-06-10

### Changed

- **Product vs operator split:** operator deployment docs (cron/CI, reference hook scripts, infra-probe assets) moved to **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)**; stubs and links remain in this repo. README adds **Operator deployment** table; new **[AGENTS.md](AGENTS.md)** documents scope (same pattern as pgwd / pgwd-selfhosted).

## [0.6.1] - 2026-06-07

### Added

- **`contrib/man/man1/kzero.1`**: manual page for BSD ports, Linux packages, release tarballs, and **`man kzero`** after install.
- **Homebrew cask** published to **[hrodrig/homebrew-kzero](https://github.com/hrodrig/homebrew-kzero)** on release (`brew install hrodrig/kzero/kzero`); requires **`HOMEBREW_TAP_TOKEN`** in CI (same pattern as pgwd).

### Changed

- Release **archives** and **`.deb`/`.rpm`** include **`share/man/man1/kzero.1`** (packages install **`/usr/share/man/man1/kzero.1.gz`**).
- FreeBSD and OpenBSD port skeletons install the man page from distfiles.

## [0.6.0] - 2026-06-03

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
- **Ports:** sync FreeBSD and OpenBSD port Makefiles to **0.6.0**.

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

[Unreleased]: https://github.com/hrodrig/kzero/compare/v0.9.2...HEAD
[0.9.2]: https://github.com/hrodrig/kzero/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/hrodrig/kzero/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/hrodrig/kzero/compare/v0.8.1...v0.9.0
[0.8.1]: https://github.com/hrodrig/kzero/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/hrodrig/kzero/compare/v0.7.4...v0.8.0
[0.7.4]: https://github.com/hrodrig/kzero/compare/v0.7.3...v0.7.4
[0.7.3]: https://github.com/hrodrig/kzero/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/hrodrig/kzero/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/hrodrig/kzero/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/hrodrig/kzero/compare/v0.6.2...v0.7.0
[0.6.2]: https://github.com/hrodrig/kzero/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/hrodrig/kzero/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/hrodrig/kzero/compare/v0.5.7...v0.6.0
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
