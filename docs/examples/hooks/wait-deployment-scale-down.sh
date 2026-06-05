#!/bin/sh
# Wait until a Deployment finished scaling (use as per-step post on down, or pre on the next step).
# Requires: kubectl on PATH; KZERO_STEP_NAMESPACE and KZERO_STEP_NAME set by kzero for workload steps.
set -euo pipefail

ns="${KZERO_STEP_NAMESPACE:?KZERO_STEP_NAMESPACE not set}"
name="${KZERO_STEP_NAME:?KZERO_STEP_NAME not set}"
timeout="${KZERO_ROLLOUT_TIMEOUT:-10m}"

kubectl -n "$ns" rollout status "deployment/${name}" --timeout="$timeout"
