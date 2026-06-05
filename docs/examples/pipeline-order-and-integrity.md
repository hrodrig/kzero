# Pipeline order and integrity on `down`

This page complements [SPECIFICATIONS.md](../SPECIFICATIONS.md#current-engine-sequencing-retry-and-concurrency) with **concrete YAML** for operators who care about **correct ordering**, not only speed.

## What YAML order guarantees

`pipelines.down` is a **strictly sequential** list. Step *i+1* starts only after step *i* completes without error (fail-fast).

For a **workload** step (`deployment` / `statefulset`), the main action on **`down`** is: set **replicas to 0** (via `kubectl scale` or the native API). That call returning success means the API accepted the scale — **not** that all pods have terminated.

| Mechanism | On `down` | On `up` |
|-----------|-----------|---------|
| YAML list order | Previous step finished (including its `pre` / main / `post`) | Same |
| `wait_for_ready: true` | **Not used** (ignored for termination) | Waits for rollout / ready after scale-up |
| Per-step `post` | Runs after a **successful** scale; use to wait for pods to finish | Same semantics after scale-up |

There is **no** built-in `depends_on` field. Express “B may only change after A is fully down” with **step order** plus optional **`post`** / **`pre`** scripts.

## Example: two Deployments (consumer before producer)

Scenario: **`deployment.app/producer`** must not be scaled down until **`deployment.app/consumer`** has no running pods (for example, the consumer drains a queue the producer feeds).

```yaml
pipelines:
  down:
    - deployment.app/consumer:
        post: ./hooks/wait-deployment-scale-down.sh
    - deployment.app/producer
  up:
    - deployment.app/producer:
        replicas: 1
        wait_for_ready: true
        timeout: 10m
    - deployment.app/consumer:
        replicas: 1
        wait_for_ready: true
```

Copy the hook from [hooks/wait-deployment-scale-down.sh](hooks/wait-deployment-scale-down.sh) into your config directory (for example next to `kzero.yaml`), mark it executable (`chmod +x`), and point `post` at that path.

On **`down`**, kzero runs:

1. Scale `consumer` to 0.
2. **`post`**: `kubectl rollout status deployment/consumer` (fails the pipeline if pods do not finish terminating within the timeout).
3. Scale `producer` to 0.

If step 2 fails, **`producer` is never scaled** (fail-fast).

## Example: StatefulSet with work before scale (from sample profile)

Drain or export data **while the StatefulSet still has pods**, then scale to zero in the same step:

```yaml
pipelines:
  down:
    - statefulset.database/postgresql:
        pre: ./hooks/before-pg-scale.sh
        post: ./hooks/after-pg-scale.sh
```

- **`pre`**: runs with replicas still at their current count (exec, backup, etc.).
- **Main**: scale to 0.
- **`post`**: optional cleanup or verification (only if scale succeeded).

See [configs/kzero.sample.yml](../../configs/kzero.sample.yml) for this fragment in a full file.

## Example: optional guard on the dependent step

Assert the upstream Deployment has no ready replicas before scaling the next workload:

```yaml
    - deployment.app/producer:
        pre: ./hooks/assert-deployment-replicas-zero.sh
```

Example `assert-deployment-replicas-zero.sh` (checks the **previous** tier you name in the script or via env):

```sh
#!/bin/sh
set -euo pipefail
ns="${KZERO_STEP_NAMESPACE:?}"
upstream="${UPSTREAM_DEPLOYMENT:-consumer}"
ready="$(kubectl -n "$ns" get deployment "$upstream" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)"
ready="${ready:-0}"
if [ "$ready" != "0" ] && [ -n "$ready" ]; then
  echo "upstream deployment/${upstream} still has readyReplicas=${ready}" >&2
  exit 1
fi
```

Set `UPSTREAM_DEPLOYMENT` in the script or export it from a thin wrapper; kzero does not infer cross-step dependencies automatically.

## Hook environment (workload steps)

Per-step hooks receive (among others):

| Variable | Example |
|----------|---------|
| `KZERO_PHASE` | `down` |
| `KZERO_STEP_TYPE` | `deployment` |
| `KZERO_STEP_NAMESPACE` | `app` |
| `KZERO_STEP_NAME` | `consumer` |
| `KZERO_STEP_REF` | `deployment.app/consumer` |

Full table: [SPECIFICATIONS.md § Per-step pre/post](../SPECIFICATIONS.md#per-step-pre--post-behavior-live-mode).

## `analyze` output

`kzero analyze` lists steps in pipeline order and annotates map steps, for example:

```text
  0: deployment.app/consumer (post: ./hooks/wait-deployment-scale-down.sh)
  1: deployment.app/producer
```

Release steps on **`down`** show `helm uninstall --wait --ignore-not-found` in the plan (see [CHANGELOG](../../CHANGELOG.md)).

## Related

- [README — Per-step pre/post](../../README.md#per-step-pre-post-example)
- [SPEC — Current engine sequencing](../SPECIFICATIONS.md#current-engine-sequencing-retry-and-concurrency)
