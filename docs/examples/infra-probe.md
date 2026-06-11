# Infra probe cookbook

Use **`infra_probe`** to run a **throwaway mini-pipeline** before a destructive **`kzero reset`** or **`down`**. The goal is not to test your application workloads—it is to confirm that **platform prerequisites still work** right before you wipe real PVCs and core releases.

## What a probe is meant to validate

In **live** mode, a successful probe run is evidence that (for **your** chosen probe chart):

| Layer | What failure looks like | Typical probe signal |
|-------|-------------------------|----------------------|
| **Helm / OCI registry** | Chart pull timeout, 401/403, wrong version | `infra probe: up` fails on `helm upgrade --install` |
| **Container registry** | Image pull backoff, missing tag, rate limit | Pod `ImagePullBackOff` during probe up (`release_ready` fails) |
| **Image pull credentials** | Private registry without `imagePullSecrets` | Same as image pull; fix in **your** values |
| **Storage / volumes** | StorageClass missing, CSI down, quota | `pvc_bound` stays `Pending` or check fails |
| **Helm install path** | RBAC, namespace, webhook rejects release | Probe up script exits non-zero (`release_ready` fails) |

**kzero** orchestrates **up → checks → down** and optional **gate** before the main pipeline. It does **not** embed cloud-specific logic, registry URLs, or PVC cleanup rules—you maintain the probe chart under **`helm.workspace`** (or `custom:` scripts).

## Use our reference example—or bring your own

kzero ships an **anonymous reference** (public Bitnami Redis 8) in **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)**:

| File | Purpose |
|------|---------|
| [run/examples/infra-probe/kzero-probe-redis.sh](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/infra-probe/kzero-probe-redis.sh) | Install script → `<helm.workspace>/probe-redis.sh` (**shell** path) |
| [run/examples/infra-probe/kzero-probe-redis.yaml](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/infra-probe/kzero-probe-redis.yaml) | Chart manifest → `<helm.workspace>/probe-redis.yaml` (**native** path) |
| [run/examples/infra-probe/kzero-probe-redis-values.yaml](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/infra-probe/kzero-probe-redis-values.yaml) | Values (image, PVC size, retention on uninstall) |
| [run/examples/infra-probe/kzero-infra-probe-redis.sample.yaml](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/infra-probe/kzero-infra-probe-redis.sample.yaml) | YAML fragment (**shell**) |
| [run/examples/infra-probe/kzero-infra-probe-redis-native.sample.yaml](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/infra-probe/kzero-infra-probe-redis-native.sample.yaml) | YAML fragment (**native** / distroless) |

**Copy and edit** those files, or ignore them entirely and wire `infra_probe` to whatever you prefer:

- **MariaDB / PostgreSQL / RabbitMQ** — another `release.<ns>/<name>` + values in your workspace
- **Minimal busybox + PVC** — `custom:` apply/delete scripts
- **Same chart as production** — usually **avoid** (probe should be disposable and isolated)

Align **`infra_probe.checks`** (`pvc_bound`, `release_ready`) with **your** chart’s PVC name and install semantics.

## Minimal config (generic)

```yaml
infra_probe:
  enabled: true
  before: ["reset"]          # add "down" to gate both commands
  fail_fast: true
  cache_ttl: 30m
  pipeline:
    up:
      - release.probe-ns/probe-storage
    down:
      - release.probe-ns/probe-storage
  checks:
    - pvc_bound: probe-ns/<your-pvc-name>
    - release_ready: true

run:
  mode: live
  probe_cache_dir: /var/lib/kzero/probe-cache   # optional; shared path for cron/CI
```

Place **`<release>.sh`** under **`helm.workspace`** (basename = release **name** in `release.<ns>/<name>`).

### Reference Redis wiring (after copying example files)

```yaml
infra_probe:
  pipeline:
    up:
      - release.probe-ns/probe-redis
    down:
      - release.probe-ns/probe-redis
  checks:
    - pvc_bound: probe-ns/redis-data-probe-redis-master-0
    - release_ready: true
```

## Native path (distroless / in-cluster)

For **distroless** Jobs or hosts without **`/bin/sh`**, set **`run.execution: native`** (or **`auto`**) and use **Helm SDK** for probe **`release.*`** steps — no **`<release>.sh`** script.

1. Copy **`kzero-probe-redis-values.yaml`** and **`kzero-probe-redis.yaml`** (chart manifest) into **`helm.workspace`** — see [docs/examples/infra-probe/](infra-probe/) in the product repo and [kzero-selfhosted run/examples/infra-probe/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/infra-probe).
2. Merge [kzero-infra-probe-native.sample.yaml](infra-probe/kzero-infra-probe-native.sample.yaml) (or selfhosted **`kzero-infra-probe-redis-native.sample.yaml`**).
3. Grant RBAC for Helm release secrets/configmaps, **`pods`** (checks), and optional **`pvc`** / **`exec`** in the probe namespace — see [kzero-selfhosted run/in-cluster/](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/in-cluster).

Optional **`pvc.*`** on probe **`down`** or **`exec.*`** write/read checks can extend the mini-pipeline when chart retention does not delete PVCs automatically.

```yaml
run:
  execution: native
  kubeconfig: ""   # in-cluster Job

infra_probe:
  pipeline:
    up:
      - release.probe-ns/probe-redis
    down:
      - release.probe-ns/probe-redis
```

## Probe chart teardown (operator-owned)

**kzero** runs **`helm uninstall`** on probe **`down`**; it does **not** delete PVCs unless **your** chart/values arrange it.

Configure cleanup in the **probe chart values** you maintain, for example:

- **Bitnami Redis / PostgreSQL / RabbitMQ (StatefulSet):** PVC retention when the StatefulSet is deleted on uninstall:

```yaml
master:
  persistentVolumeClaimRetentionPolicy:
    enabled: true
    whenScaled: Retain
    whenDeleted: Delete
```

- **Generic Helm templates you control:** `helm.sh/resource-policy: delete` where applicable (not always effective on StatefulSet PVCs).

- **Custom manifests:** retention policy or explicit PVC delete in your scripts.

Pick the pattern that matches **your** probe chart.

## Commands

| Command | Behavior |
|---------|----------|
| **`kzero probe`** | Always runs up → checks → down (respects **`run.mode`**) |
| **`kzero reset`** / **`down`** | Runs probe first when **`enabled`** and command ∈ **`before`** (**live** only) |

Dry-run main pipelines do **not** run the gate; use **`kzero probe`** with **`run.mode: dry-run`** to preview steps.

Recommended workflow:

1. **`kzero probe`** in **dry-run** — plan only  
2. **`kzero probe`** in **live** — validate registry, credentials, volumes  
3. Set **`infra_probe.enabled: true`** — gate **`reset`** / **`down`** in automation  

## Cache

After a successful probe in **live** mode, **`cache_ttl`** skips re-running probe steps when the fingerprint (cluster + pipeline + checks) matches. Set **`run.probe_cache_dir`** on shared hosts so cache survives process restarts.

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `infra probe: up:` … chart fetch | OCI/registry network, chart version typo |
| `ImagePullBackOff` on probe pod | Image tag, registry auth, firewall |
| `pvc_bound … phase Pending` | StorageClass, CSI, quota |
| `release_ready` | Probe up did not finish (`--wait`, pod not Ready) |
| PVC left after probe down | Teardown not configured in **your** chart values |
| Gate skipped | **`enabled: false`**, command not in **`before`**, or **`run.mode: dry-run`** |

## Related

- [SPECIFICATIONS.md § infra_probe](../SPECIFICATIONS.md)
- [notifications.md](notifications.md) — optional alerts on pipeline failure after probe passes
