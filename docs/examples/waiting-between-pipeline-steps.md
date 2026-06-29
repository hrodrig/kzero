# Waiting between pipeline steps

How to make step *i+1* start only after step *i* is **fully ready**—not merely submitted. Examples use placeholder names (`platform`, `postgresql`, `./helm-assets/`).

kzero always runs pipeline steps **sequentially** (fail-fast). What you must add is a **readiness gate** when the main action returns before the cluster is healthy (typical for Helm without `--wait`).

## What list order already gives you

```yaml
pipelines:
  up:
    - release.platform/postgresql
    - release.platform/rabbitmq
    - release.platform/redis-eviction
```

Step 2 starts only after step 1’s **whole unit** succeeds: optional `pre` → main action → optional `post`. If step 1 fails, step 2 never runs.

**Gap:** a `release.*` **up** step runs `<helm.workspace>/<name>.sh`. If that script’s `helm upgrade --install` returns without `--wait`, the next release may start while Postgres is still starting.

Use one or more of the three patterns below.

---

## 1. `--wait` in the Helm install script

Add **`--wait`** (and usually **`--timeout`**) to each `helm upgrade --install` in your release script so the script itself blocks until Helm considers the release ready.

**`helm-assets/postgresql.sh`** (fragment):

```bash
#!/bin/sh
set -euo pipefail

helm upgrade --install postgresql oci://registry.example.com/helm/postgresql \
  --namespace platform \
  --wait \
  --timeout 15m \
  -f ./postgresql-values.yaml
```

**`helm-assets/rabbitmq.sh`** — same pattern:

```bash
helm upgrade --install rabbitmq oci://registry.example.com/helm/rabbitmq \
  --namespace platform \
  --wait \
  --timeout 15m \
  -f ./rabbitmq-values.yaml
```

| Pros | Cons |
|------|------|
| One place per chart; no YAML change | Every `.sh` must be updated |
| Works with plain `release.ns/name` steps | Timeout tuning per chart |

**Pilot profile** (`kzero.yaml` unchanged):

```yaml
pipelines:
  up:
    - release.platform/postgresql
    - release.platform/rabbitmq
```

kzero runs `postgresql.sh`, waits for it to exit, then runs `rabbitmq.sh`.

---

## 2. Per-step `post:` hook after each `release.*`

Keep scripts as-is (or combine with §1) and add a **`post`** hook that waits on the release before the next YAML step runs.

**`kzero.yaml`:**

```yaml
helm:
  workspace: "./helm-assets"

pipelines:
  up:
    - release.platform/postgresql:
        post: ./hooks/wait-helm-release-ready.sh
    - release.platform/rabbitmq:
        post: ./hooks/wait-helm-release-ready.sh
    - release.platform/redis-eviction:
        post: ./hooks/wait-helm-release-ready.sh
```

Copy the reference hook from [hooks/wait-helm-release-ready.sh](hooks/wait-helm-release-ready.sh), `chmod +x`, and place it next to your config (or use an absolute path).

**What runs for Postgres:**

1. **Main:** `helm-assets/postgresql.sh up` (via kzero → `/bin/sh postgresql.sh up`)
2. **`post`:** `kubectl rollout status statefulset/postgresql` (or `deployment/` / `kubectl wait` on `app.kubernetes.io/instance` — see the hook)

RabbitMQ does not start until both succeed.

**Note:** `helm status` has **no** `--wait` flag. The reference hook uses **`kubectl rollout status`** or **`kubectl wait`** on pods labeled `app.kubernetes.io/instance=<release>`.

**Hook environment** (release steps):

| Variable | Example |
|----------|---------|
| `KZERO_PHASE` | `up` |
| `KZERO_STEP_TYPE` | `release` |
| `KZERO_RELEASE_NAME` | `postgresql` |
| `KZERO_RELEASE_NAMESPACE` | `platform` |
| `KZERO_CLIENT_ID` | your `client.id` when set |

Optional: `export KZERO_HELM_WAIT_TIMEOUT=20m` before `kzero up` to override the hook default.

| Pros | Cons |
|------|------|
| Readiness policy in YAML, one shared hook | Extra process per release step |
| Composable with §1 (belt and suspenders) | Requires `helm` on PATH |

**`analyze` plan** shows the hook:

```text
  0: release.platform/postgresql (script: ./helm-assets/postgresql.sh, post: ./hooks/wait-helm-release-ready.sh)
  1: release.platform/rabbitmq (script: ./helm-assets/rabbitmq.sh, post: ./hooks/wait-helm-release-ready.sh)
```

---

## 3. `wait_for_ready` on `deployment` / `statefulset` (up only)

For workload steps (not `release.*`), use map form with **`wait_for_ready: true`** so kzero waits for rollout/readiness after scale-up (`kubectl rollout status` or native API poll).

```yaml
pipelines:
  up:
    - release.platform/postgresql:
        post: ./hooks/wait-helm-release-ready.sh
    - release.platform/rabbitmq:
        post: ./hooks/wait-helm-release-ready.sh
    - deployment.platform/config-service:
        replicas: 1
        wait_for_ready: true
        timeout: 10m
    - deployment.platform/webui:
        replicas: 3
        wait_for_ready: true
        timeout: 15m
    - statefulset.platform/data-extractor-slave:
        replicas: 4
        wait_for_ready: true
        timeout: 20m
```

| Field | Applies to |
|-------|------------|
| `wait_for_ready: true` | **`up`** only — after scale-up |
| `timeout` | Rollout wait budget (per step) |
| `replicas` | Target count on **`up`** (default 1) |

**Not used on `down`** for pod drain; use per-step **`post`** hooks (see [pipeline-order-and-integrity.md](pipeline-order-and-integrity.md)).

---

## 4. Slaves wait for master (`pre` + `wait_for_ready`)

When **masters** (`deployment/…`) scale earlier in the list and **slaves** (`statefulset/…-slave`) run later, add:

- **`wait_for_ready: true`** on each master `deployment` when it is scaled on **`up`**
- **`pre: ./hooks/wait-master-ready.sh`** on each `*-slave` step before scale-up (derives `deployment/<name>` from `…-slave`)

```yaml
    - deployment.platform/data-extractor:
        replicas: 1
        wait_for_ready: true
    # ... other steps ...
    - statefulset.platform/data-extractor-slave:
        pre: ./hooks/wait-master-ready.sh
        replicas: 4
        wait_for_ready: true
```

Reference: [hooks/wait-master-ready.sh](hooks/wait-master-ready.sh).

---

## Combining all three (typical infra + apps `up`)

```yaml
pipelines:
  up:
    # §1 inside each .sh (--wait) + §2 post hook for validation/audit
    - release.platform/postgresql:
        post: ./hooks/wait-helm-release-ready.sh
    - release.platform/rabbitmq:
        post: ./hooks/wait-helm-release-ready.sh
    - release.platform/jobstore-data-extractor:
        post: ./hooks/wait-helm-release-ready.sh
    # §3 workload rollout wait
    - deployment.platform/vaultunsealer:
        replicas: 1
        wait_for_ready: true
        timeout: 10m
    - deployment.platform/config-service:
        replicas: 1
        wait_for_ready: true
    - deployment.platform/webui:
        replicas: 3
        wait_for_ready: true
        timeout: 15m
```

**Dry-run:** `kzero analyze` lists order and annotations; `kzero up` in `dry-run` logs planned hooks and `wait_for_ready` without mutating the cluster.

**Live validation:** run `kzero analyze`, then `kzero up` in `dry-run`, then `live` when the plan matches. See [kzero-selfhosted automation guide](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/docs/automation-and-pipelines.md).

---

## Related

- [pipeline-order-and-integrity.md](pipeline-order-and-integrity.md) — `down` ordering, drain with `post`, StatefulSet `pre`/`post`
- [hooks/wait-deployment-scale-down.sh](hooks/wait-deployment-scale-down.sh) — Deployment drain on `down`
- [hooks/wait-helm-release-ready.sh](hooks/wait-helm-release-ready.sh) — Helm release wait on `up`
- [SPECIFICATIONS.md § Supported workload kinds](../../SPECIFICATIONS.md#supported-workload-kinds) — `release` down uses `helm uninstall --wait`
