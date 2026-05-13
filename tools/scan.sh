#!/usr/bin/env bash
# Security scans from repo root (same idea as pgwd). Run: ./tools/scan.sh
set -e
cd "$(dirname "$0")/.."

GOVULNCHECK_FAIL=0

if command -v govulncheck >/dev/null 2>&1; then
  echo "=== govulncheck ./... ==="
  if ! govulncheck ./...; then
    GOVULNCHECK_FAIL=1
  fi
else
  echo "govulncheck not found; install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
  GOVULNCHECK_FAIL=1
fi

if command -v grype >/dev/null 2>&1; then
  echo "=== grype (current dir) ==="
  grype . || true
else
  echo "Grype not found (optional); see https://github.com/anchore/grype#installation"
fi

exit "$GOVULNCHECK_FAIL"
