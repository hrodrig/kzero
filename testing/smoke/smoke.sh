#!/usr/bin/env bash
# CI / local smoke: build kzero, analyze, dry-run down, print-sample-config.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
CFG="$ROOT/testing/smoke/kzero-smoke.yaml"

# Isolate from developer kubeconfig so analyze does not hit a real cluster.
SMOKE_HOME="$(mktemp -d)"
trap 'rm -rf "$SMOKE_HOME"' EXIT
export HOME="$SMOKE_HOME"
unset KUBECONFIG

echo "Building kzero..."
make build
export PATH="$ROOT/bin:$PATH"

echo "kzero version:"
kzero version

echo "kzero analyze..."
analyze_out="$(mktemp)"
trap 'rm -f "$analyze_out"' EXIT
kzero analyze --config "$CFG" >"$analyze_out" 2>&1
for needle in '[down]' '[up]' 'Run mode:' 'Pipeline steps:'; do
  grep -q "$needle" "$analyze_out" || {
    echo "error: analyze missing: $needle"
    cat "$analyze_out"
    exit 1
  }
done

echo "kzero down (dry-run)..."
down_out="$(mktemp)"
trap 'rm -f "$analyze_out" "$down_out"' EXIT
kzero down --config "$CFG" >"$down_out" 2>&1
grep -q '\[dry-run\]' "$down_out" || {
  echo "error: down missing dry-run marker"
  cat "$down_out"
  exit 1
}

echo "kzero --print-sample-config..."
sample_out="$(mktemp)"
trap 'rm -f "$analyze_out" "$down_out" "$sample_out"' EXIT
kzero --print-sample-config >"$sample_out" 2>&1
grep -q 'schema_version:' "$sample_out" || {
  echo "error: print-sample-config missing schema_version"
  cat "$sample_out"
  exit 1
}

echo "smoke passed"
