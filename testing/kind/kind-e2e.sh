#!/usr/bin/env bash
# Product-repo kind integration (#34): one Deployment, native live down/up.
# Flake policy and budget: testing/kind/README.md
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

CLUSTER="${KZERO_KIND_CLUSTER:-kzero-ci}"
KIND_DIR="$ROOT/testing/kind"
CFG="$KIND_DIR/kzero.yaml"
WORKLOADS="$KIND_DIR/workloads.yaml"
ROLLOUT_TIMEOUT="${KZERO_KIND_ROLLOUT_TIMEOUT:-120s}"
KZERO_BIN="${KZERO_BIN:-}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: $1 required on PATH" >&2
    exit 1
  }
}

need kind
need kubectl
need docker

docker info >/dev/null 2>&1 || {
  echo "error: Docker daemon not reachable (start Docker Desktop / dockerd)" >&2
  exit 1
}

if [[ -z "$KZERO_BIN" ]]; then
  echo "Building kzero..."
  make build
  KZERO_BIN="$ROOT/bin/kzero"
fi
[[ -x "$KZERO_BIN" ]] || {
  echo "error: KZERO_BIN not executable: $KZERO_BIN" >&2
  exit 1
}

cleanup() {
  if [[ -n "${KZERO_KIND_NO_CLEANUP:-}" ]]; then
    echo "KZERO_KIND_NO_CLEANUP set; leaving cluster $CLUSTER"
    return 0
  fi
  kind delete cluster --name "$CLUSTER" >/dev/null 2>&1 || true
}
trap cleanup EXIT

create_cluster() {
  if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
    kind delete cluster --name "$CLUSTER"
  fi
  kind create cluster --name "$CLUSTER" --wait 120s
}

# Flake policy: one retry only for kind create / node readiness failures.
echo "Creating kind cluster $CLUSTER..."
if ! create_cluster; then
  echo "warn: kind create failed; one retry after delete" >&2
  kind delete cluster --name "$CLUSTER" >/dev/null 2>&1 || true
  sleep 5
  create_cluster
fi

kubectl config use-context "kind-${CLUSTER}"

echo "Applying workloads..."
kubectl apply -f "$WORKLOADS"
kubectl -n kzero-kind rollout status deployment/web --timeout="$ROLLOUT_TIMEOUT"

replicas_ready() {
  local want="$1"
  local got
  got="$(kubectl -n kzero-kind get deploy web -o jsonpath='{.status.readyReplicas}')"
  [[ "${got:-0}" == "$want" ]]
}

echo "Assert replicas=2 before down..."
replicas_ready 2 || {
  echo "error: expected 2 ready replicas before down" >&2
  kubectl -n kzero-kind get deploy,pods -o wide >&2 || true
  exit 1
}

echo "kzero analyze..."
"$KZERO_BIN" analyze --config "$CFG" | tee /tmp/kzero-kind-analyze.txt
grep -q 'Cluster validation:' /tmp/kzero-kind-analyze.txt
grep -q 'OK  deployment.kzero-kind/web' /tmp/kzero-kind-analyze.txt

echo "kzero down (live)..."
"$KZERO_BIN" down --config "$CFG"

echo "Wait scale to 0..."
kubectl -n kzero-kind wait --for=jsonpath='{.spec.replicas}'=0 deployment/web --timeout=120s
got="$(kubectl -n kzero-kind get deploy web -o jsonpath='{.spec.replicas}')"
[[ "$got" == "0" ]] || {
  echo "error: expected spec.replicas=0 after down, got $got" >&2
  exit 1
}

echo "kzero up (live)..."
"$KZERO_BIN" up --config "$CFG"

echo "Assert replicas=2 after up..."
replicas_ready 2 || {
  echo "error: expected 2 ready replicas after up" >&2
  kubectl -n kzero-kind get deploy,pods -o wide >&2 || true
  exit 1
}

echo "kind e2e passed"
