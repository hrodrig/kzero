# Automation and CI/CD pipelines

How to run **kzero** non-interactively from cron, shell scripts, or CI jobs. Examples use **placeholder paths**—replace with your kubeconfig, binary, and profile directory.

## kzero has no built-in confirmation prompt

The **`kzero`** binary does **not** ask for `YES` before mutating a cluster. In **`run.mode: live`**, it executes the pipeline as soon as the command starts (after config load and the usual **`Kubernetes target:`** block).

If your environment adds a **wrapper script** (for example one that runs **`kzero target`** and prompts an operator), that gate is **outside** kzero—you must satisfy it in automation (see [Wrapper with a manual gate](#wrapper-with-a-manual-gate) below).

## Set live mode

Pick one approach:

| Approach | Example |
|----------|---------|
| YAML | `run.mode: "live"` in the profile |
| Environment | `export KZERO_RUN_MODE=live` (overrides YAML via `KZERO_*`) |
| Wrapper flag | `./run-operator --live down` (if your wrapper patches mode into a temp config) |

Stay on **`dry-run`** until **`kzero analyze`** and a dry-run **`down`** / **`up`** match expectations.

## Direct invocation (recommended for CI)

No confirmation step—only your own review process before enabling **`live`**.

```bash
export KUBECONFIG=/path/to/kubeconfig
export KZERO_RUN_MODE=live          # optional if already set in YAML

kzero analyze --config /path/to/profile.yaml
kzero down  --config /path/to/profile.yaml
```

Pinned binary (release artifact or `make build`):

```bash
export KUBECONFIG=/path/to/kubeconfig
export KZERO_BIN=/path/to/kzero      # optional; default: kzero on PATH

"${KZERO_BIN:-kzero}" down --config /path/to/profile.yaml
```

Working directory matters for **relative** paths in the profile (`helm.workspace`, hook scripts, `custom:` steps). Either `cd` to the profile directory or use absolute paths in YAML.

```bash
export KUBECONFIG=/path/to/kubeconfig
cd /path/to/profile-directory
kzero down --config ./profile.yaml
```

### Other useful overrides

```bash
export KZERO_RUN_KUBECONFIG=/path/to/kubeconfig   # same effect as KUBECONFIG for kzero/kubectl children
export KZERO_CLIENT_ID=my-pipeline-runner         # overrides client.id from YAML
export KZERO_COLOR=never                          # plain logs in CI
# Structured logs for log aggregators:
# kzero down --config ./profile.yaml --log-format json
```

**Notifications:** configure **`notify.*`** in YAML or via **`KZERO_NOTIFY_*`** env vars; run **`kzero notify test`** before the first live pipeline. See [notifications.md](notifications.md).

## Wrapper with a manual gate

Some teams ship a **host wrapper** around kzero that:

1. Verifies prerequisites (`kubectl`, `helm`, config file).
2. Runs **`kzero target`** and prints the cluster.
3. Requires typing **`YES`** before **`down`**, **`up`**, or **`reset`** in live mode.

For **automated** runs (cron, GitHub Actions, GitLab CI), pipe the expected answer on stdin:

```bash
export KUBECONFIG=/path/to/kubeconfig
export KZERO_BIN=/path/to/kzero          # if the wrapper respects KZERO_BIN
cd /path/to/profile-directory

printf 'YES\n' | ./run-operator --live down
```

Same pattern for **`up`** or **`reset`**:

```bash
printf 'YES\n' | ./run-operator --live up
printf 'YES\n' | ./run-operator --live reset
```

**`analyze`** and **`target`** typically do not need **`YES`** (read-only / no cluster mutation).

### Logging wrapper output

If the wrapper tees to a log file (for example `.logs/kzero-down-<timestamp>.log`), keep that directory on the runner and archive logs as CI artifacts when useful.

## Suggested pipeline order

1. **`kzero analyze`** — validate schema, print plan, optional API workload checks.
2. **`kzero down`** or **`up`** with **`run.mode: dry-run`** — review `[dry-run]` lines (and `client_id=` when `client.id` is set).
3. **`kzero <phase>`** with **`run.mode: live`** — only after review, on the intended kubeconfig.

Example CI skeleton (pseudo-YAML):

```yaml
steps:
  - run: |
      export KUBECONFIG="$KUBECONFIG_PATH"
      kzero analyze --config "$PROFILE_PATH"
  - run: |
      export KUBECONFIG="$KUBECONFIG_PATH"
      export KZERO_RUN_MODE=dry-run
      kzero down --config "$PROFILE_PATH"
  - when: manual   # or protected branch / environment approval
    run: |
      export KUBECONFIG="$KUBECONFIG_PATH"
      export KZERO_RUN_MODE=live
      kzero down --config "$PROFILE_PATH"
```

If you use a **YES-gated wrapper** on the live step, replace the last `kzero` line with:

```bash
printf 'YES\n' | ./run-operator --live down
```

## Exit codes and failures

- Non-zero exit if config validation fails, a pipeline step fails, or **`reset`** aborts because **`down`** failed.
- In **live** mode, a failed step stops the phase; later steps do not run.
- Capture stdout/stderr in CI; kzero prints **`Kubernetes target:`** at the start of pipeline commands—use it to verify the job hit the right cluster.

## Safety checklist

- **Never** auto-confirm live runs against production without an approval gate (environment protection, manual job, or change ticket).
- Store kubeconfig as a **secret**; restrict RBAC to the minimum needed for your pipeline steps.
- Prefer **`kzero target`** or **`analyze`** in a prior job to fail fast on wrong context.
- Document who may run **`printf 'YES\n' | …`** wrappers and rotate credentials when people leave.

## Related

- [SPECIFICATIONS.md](../SPECIFICATIONS.md) — config schema, `KZERO_*` overrides, run modes
- [pipeline-order-and-integrity.md](pipeline-order-and-integrity.md) — ordering and per-step hooks on `down`
