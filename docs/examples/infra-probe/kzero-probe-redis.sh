#!/bin/bash
# Reference probe install script for infra_probe (operator-maintained; copy into helm.workspace).
# Release name must match pipelines: release.<namespace>/probe-redis → probe-redis.sh
#
# Validates (live probe up): Helm/OCI chart pull, container image pull, StorageClass/PVC bind.
# Anonymous example — public Bitnami chart + bitnamilegacy image from Docker Hub.
# Replace namespace, chart version, image, values, or use an entirely different probe chart.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PROBE_NAMESPACE="${PROBE_NAMESPACE:-probe-ns}"

helm upgrade \
  --install probe-redis oci://registry-1.docker.io/bitnamicharts/redis \
  --namespace="${PROBE_NAMESPACE}" \
  --create-namespace \
  --version="25.5.2" \
  --wait \
  --timeout=10m \
  -f ./kzero-probe-redis-values.yaml
