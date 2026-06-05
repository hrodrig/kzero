#!/bin/sh
# Wait for the matching master Deployment before scaling a *-slave StatefulSet.
# Naming: statefulset/foo-slave -> deployment/foo (same namespace).
# kzero sets KZERO_STEP_NAMESPACE, KZERO_STEP_NAME on per-step pre/post hooks.
set -euo pipefail

ns="${KZERO_STEP_NAMESPACE:?KZERO_STEP_NAMESPACE not set}"
slave="${KZERO_STEP_NAME:?KZERO_STEP_NAME not set}"
timeout="${KZERO_MASTER_WAIT_TIMEOUT:-10m}"

case "$slave" in
  *-slave) master="${slave%-slave}" ;;
  *)
    echo "wait-master-ready: step ${slave} is not a *-slave workload" >&2
    exit 1
    ;;
esac

kubectl -n "$ns" rollout status "deployment/${master}" --timeout="$timeout"
