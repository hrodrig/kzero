# PVC / StatefulSet data strategy (pipeline patterns)

**kzero** can **delete** named PersistentVolumeClaims (`pvc.<namespace>/<name>`) and **scale** StatefulSets. It does **not** invent volume snapshots, CSI backups, or database-aware wipe logic. Those stay in **operator scripts** (`custom:`, per-step hooks) or external tools.

This cookbook shows **patterns** you compose in YAML for reset / maintenance. Contract: [SPECIFICATIONS.md](../../SPECIFICATIONS.md) (`pvc`, `statefulset`, `exec`). Related: [pipeline-order-and-integrity.md](pipeline-order-and-integrity.md), [infra-probe.md](infra-probe.md).

## What the engine does

| Step | Live action |
|------|-------------|
| `statefulset.ns/name` on **down** | Scale replicas to **0** (no wait for pod exit unless you add **`post`**) |
| `pvc.ns/claim` | API **delete** claim (`DeletePropagationBackground`, ignore-not-found) — **same on up and down**; typically list on **`down`** after scale |
| `exec.ns/pod` | Run a command in a container (wipe files, `TRUNCATE`, …) while the pod still exists |

**Important:** Deleting a PVC while pods still mount it often **fails** or leaves odd states. Always **scale (and wait) first**, then delete claims.

Shell scripts for hooks/`custom:` must be **POSIX `/bin/sh`**-safe (Ubuntu **dash**): see [SPEC — Hook and script interpreter](../../SPECIFICATIONS.md#hook-and-script-interpreter-binsh).

---

## Pattern A — scale → wait → delete PVC (fresh volumes on up)

Goal: next **`up`** (Helm/operator) recreates empty PVCs.

```yaml
pipelines:
  down:
    - statefulset.database/postgresql:
        post: ./hooks/wait-statefulset-scale-down.sh
    - pvc.database/data-postgresql-0
    - pvc.database/data-postgresql-1
  up:
    - release.database/postgresql:
        # wait in chart script or post hook
    - statefulset.database/postgresql:
        replicas: 3
        wait_for_ready: true
        timeout: 15m
```

- **`post`** after scale: typically `kubectl rollout status` / wait until pods gone (same idea as [wait-deployment-scale-down](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/hooks/wait-deployment-scale-down.sh)).
- List **every claim** you intend to wipe; naming is usually chart-specific (`data-<sts>-0`, …).
- On **up**, chart/operator must create PVCs again (normal install path).

---

## Pattern B — wipe in place (`exec`, no PVC delete)

Goal: keep the PVC object; clear application data while the pod is still up (or after scale-down to a maintenance pod).

```yaml
pipelines:
  down:
    - statefulset.app/api:
        post: ./hooks/wait-statefulset-scale-down.sh
    # Example: truncate via a still-running admin pod, or a Job/custom script.
    - exec.database/postgresql-0:
        container: postgresql
        command: ["psql", "-U", "postgres", "-c", "TRUNCATE TABLE jobs;"]
    - statefulset.database/postgresql
```

Use **`exec`** only when the command is safe for your app. Prefer app-native tools over `rm -rf` on mounted paths unless you own the data layout.

---

## Pattern C — snapshot / restore before destructive wipe

**kzero has no snapshot step.** Compose externally:

1. **`custom:`** / hook: VolumeSnapshot (or cloud snapshot CLI) **before** PVC delete.
2. Pattern A (scale → wait → `pvc.*`).
3. On failure recovery: restore from snapshot outside kzero (or a separate `up`/`custom:` playbook).

```yaml
pipelines:
  down:
    - custom: ./hooks/snapshot-pg-volumes.sh
    - statefulset.database/postgresql:
        post: ./hooks/wait-statefulset-scale-down.sh
    - pvc.database/data-postgresql-0
```

Keep snapshot scripts **operator-owned**; pass credentials via env / secret manager (not committed YAML).

---

## Pattern D — init Job (or one-shot custom) after empty PVCs

Goal: after Pattern A + Helm install, run a **migration / seed** before opening the app.

```yaml
pipelines:
  up:
    - release.database/postgresql
    - statefulset.database/postgresql:
        replicas: 1
        wait_for_ready: true
    - custom: ./hooks/run-schema-migrate.sh
    - deployment.app/api:
        replicas: 2
        wait_for_ready: true
```

Or apply a Kubernetes Job via `custom:` (`kubectl apply -f migrate-job.yaml` + wait). First-class **`job`** steps are deferred to **1.1.0** ([plan-1.1.0.md](../plan-1.1.0.md) **#29**).

---

## Helm uninstall vs explicit `pvc.*`

Uninstalling a chart (**`release.*`**) does **not** always delete StatefulSet PVCs (chart retention policies, Bitnami flags, etc.). See [infra-probe.md](infra-probe.md) teardown notes.

| Approach | When |
|----------|------|
| Chart values retain PVCs + you want data gone | Pattern A: explicit **`pvc.*`** after scale |
| Chart deletes PVCs on uninstall | Confirm once; still **scale apps** that use those volumes first |
| Probe mini-pipeline | Use probe to validate StorageClass **before** wiping production claims |

---

## Ordering checklist (before live `reset`)

1. **`analyze`** / **`doctor`** — refs exist; RBAC can delete PVCs in the namespace.
2. **`infra_probe`** (optional) — storage/registry OK before destructive main pipeline.
3. **Consumers before producers** — [pipeline-order-and-integrity.md](pipeline-order-and-integrity.md).
4. **Scale + wait** before **`pvc.*`**.
5. Hooks are **POSIX `/bin/sh`**.
6. Prefer **bastion** for destructive resets ([deployment-models.md](../deployment-models.md)).

---

## Minimal generic example

```yaml
schema_version: "1.0"
pipelines:
  down:
    - deployment.app/worker:
        post: ./hooks/wait-deployment-scale-down.sh
    - statefulset.data/cache:
        post: ./hooks/wait-statefulset-scale-down.sh
    - pvc.data/data-cache-0
  up:
    - statefulset.data/cache:
        replicas: 1
        wait_for_ready: true
    - deployment.app/worker:
        replicas: 1
        wait_for_ready: true
run:
  mode: dry-run
```

Replace names with your cluster graph. Full multi-release platform example: [kzero-selfhosted full-reset-example](https://github.com/hrodrig/kzero-selfhosted/tree/main/run/examples/full-reset-example).
