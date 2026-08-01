# `kzero diff` — plan vs live cluster

**`kzero diff`** compares the **desired** state implied by one pipeline phase (`pipelines.up` or `pipelines.down`) to the **live** Kubernetes cluster. It is **read-only** (no scale, delete, or Helm mutations).

Contract: [SPECIFICATIONS.md — `kzero diff`](../../SPECIFICATIONS.md#kzero-diff). Related: [analyze](../../SPECIFICATIONS.md#kzero-analyze), [doctor](../../SPECIFICATIONS.md#kzero-doctor), [pvc-statefulset-data-strategy.md](pvc-statefulset-data-strategy.md).

## When to use it

| Situation | Command |
|-----------|---------|
| Before **`reset`** — confirm apps look “up” already (or intentionally drifted) | `kzero diff --phase up` |
| After **`down`** — confirm scale 0, CronJobs suspended, PVCs/Jobs gone | `kzero diff --phase down` |
| Cron wrapper — fail the job if the cluster does not match expected phase | `kzero diff … \|\| exit 2` |

**`analyze`** prints the plan and optionally checks that objects **exist**. **`diff`** checks that live **values** match the phase (replicas, suspend, present/absent).

## Quick start

```bash
# Install / build kzero, point at your profile
export KUBECONFIG=~/.kube/config   # or rely on run.kubeconfig in YAML

# Default phase is up
kzero diff --config ./kzero.yaml

# After a maintenance window
kzero diff --config ./kzero.yaml --phase down
```

Exit codes: **0** match, **1** config/`--phase`, **2** drift or API error.

## Sample pipeline and expected diff

```yaml
schema_version: "1.0"
pipelines:
  down:
    - deployment.app/api
    - cronjob.batch/nightly
    - pvc.db/data-postgresql-0
    - job.batch/migrate-db
  up:
    - job.batch/migrate-db:
        manifest: ./jobs/migrate-db.yaml
    - cronjob.batch/nightly
    - deployment.app/api:
        replicas: 2
        wait_for_ready: true
run:
  mode: live
```

### After a healthy `kzero up`

```bash
kzero diff --phase up
```

Expect roughly:

```text
Diff (phase=up):
  OK     job.batch/migrate-db       present
  OK     cronjob.batch/nightly      suspend=false (live=false)
  OK     deployment.app/api         replicas=2 (live=2)
```

If someone scaled the Deployment to 0 by hand:

```text
  DRIFT  deployment.app/api         replicas=2 (live=0)
```

→ exit **2**.

### After a healthy `kzero down`

```bash
kzero diff --phase down
```

Expect:

```text
Diff (phase=down):
  OK     deployment.app/api         replicas=0 (live=0)
  OK     cronjob.batch/nightly      suspend=true (live=true)
  OK     pvc.db/data-postgresql-0   absent
  OK     job.batch/migrate-db       absent
```

**Note:** `pvc.*` steps always **delete** the claim, so desired is **absent** on both phases when that step appears in the phase list.

## Gate a reset

Only run a live reset when the cluster already matches the **up** desired state (no unexpected drift):

```bash
#!/bin/sh
set -eu
CFG=/etc/kzero/kzero.yaml
kzero diff --config "$CFG" --phase up
kzero reset --config "$CFG"
```

Or tolerate drift and always reset (skip the gate):

```bash
kzero reset --config "$CFG"
```

## What is not compared (MVP)

- **`exec.*`** / **`custom:`** — printed as **`SKIP`** (no durable cluster state).
- Helm **chart values** / revision digests — only release **presence** (Helm v3 secret labels `owner=helm,name=<release>`).
- Rollout readiness (`ReadyReplicas`) — use **`kzero verify`** after up.
- Generic CRD patch state (**#29b**).

## Tips

1. Run **`kzero analyze`** first if you are unsure which refs are in the plan.
2. Use **`kzero doctor`** if **`diff`** fails with forbidden / missing client.
3. Pair **`--phase down`** with post-`down` automation; **`--phase up`** with post-`up` or pre-`reset` checks.
