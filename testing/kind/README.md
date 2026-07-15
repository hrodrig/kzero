# Kind integration (product CI, #34)

Minimal **kind** exercise of **`kzero`** on a real API: one Deployment, **`run.execution: native`**, live **`down`** then **`up`**.

Full multi-workload lab (PVC, Postgres, counter UI): **[kzero-selfhosted `testing/kind`](https://github.com/hrodrig/kzero-selfhosted/tree/main/testing/kind)** — not a PR gate here.

Companion fast gate without a cluster: [../smoke/README.md](../smoke/README.md) (**#45**).

## Run locally

Prerequisites: **Docker**, **kind**, **kubectl**, **Go** (for `make build`).

```bash
bash testing/kind/kind-e2e.sh
```

| Variable | Default | Purpose |
|----------|---------|---------|
| **`KZERO_KIND_CLUSTER`** | `kzero-ci` | kind cluster name |
| **`KZERO_KIND_ROLLOUT_TIMEOUT`** | `120s` | `kubectl rollout status` budget |
| **`KZERO_BIN`** | `bin/kzero` after `make build` | Path to binary |
| **`KZERO_KIND_NO_CLEANUP`** | unset | Keep cluster after failure/success |

## Runtime budget

| Gate | Budget |
|------|--------|
| GitHub Actions job **`integration-kind`** | **`timeout-minutes: 20`** |
| Local script | kind wait **120s** + rollout **120s** + down/up waits **≤ ~10 min** typical |

Job must **not** use `continue-on-error`. Failures block **`develop`** / PR CI.

## Flake policy

| Class | Action |
|-------|--------|
| **kind create / node NotReady** | Script **retries once** (delete cluster, sleep 5s, recreate). Logged as `warn:`. |
| **Image pull** for `registry.k8s.io/pause` | No automatic retry; re-run the job. Prefer pause (small, usually cached on kind nodes). |
| **Assert / analyze / kzero step failure** | **No retry** — treat as product or fixture bug; fix and re-push. |
| **Manual re-run** | Operators may re-run the failed GHA job once for infrastructure flakes (create/pull). Two consecutive identical assert failures → investigate code. |

Diagnostics on failure: `kubectl -n kzero-kind get deploy,pods -o wide` (script prints this on replica asserts). Set **`KZERO_KIND_NO_CLEANUP=1`** to inspect.

## Files

| File | Purpose |
|------|---------|
| **`workloads.yaml`** | Namespace **`kzero-kind`**, Deployment **`web`** (pause ×2) |
| **`kzero.yaml`** | Live native pipelines for that Deployment |
| **`kind-e2e.sh`** | Create kind → apply → analyze → down → up → assert → delete |
