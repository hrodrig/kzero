#!/bin/sh
# Wait until a Helm release workload is ready (per-step post on release.* up steps).
# Requires: kubectl on PATH. helm status does not support --wait.
set -euo pipefail

name="${KZERO_RELEASE_NAME:-${KZERO_STEP_NAME:?release name not set}}"
ns="${KZERO_RELEASE_NAMESPACE:-${KZERO_STEP_NAMESPACE:?release namespace not set}}"
timeout="${KZERO_HELM_WAIT_TIMEOUT:-15m}"

if kubectl -n "$ns" get statefulset "$name" >/dev/null 2>&1; then
  kubectl -n "$ns" rollout status "statefulset/${name}" --timeout="$timeout"
  exit 0
fi

if kubectl -n "$ns" get deployment "$name" >/dev/null 2>&1; then
  kubectl -n "$ns" rollout status "deployment/${name}" --timeout="$timeout"
  exit 0
fi

# Helm 3 common instance label (Bitnami and many charts)
kubectl -n "$ns" wait --for=condition=ready pod \
  -l "app.kubernetes.io/instance=${name}" \
  --timeout="$timeout"
