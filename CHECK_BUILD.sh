#!/usr/bin/env bash
# Quick compile-status check for the Go backend.
# Bumps FD limit, clears stale cache, then runs go build + go vet.
set -uo pipefail

ulimit -n 16384 2>/dev/null || true
echo "[check-build] ulimit -n now $(ulimit -n)"

cd "$(dirname "$0")/backend"

if [ "${UISCE_CLEAN_CACHE:-1}" = "1" ]; then
  echo "[check-build] clearing Go cache..."
  go clean -cache 2>/dev/null || true
fi

echo "[check-build] go build ./cmd/server ..."
if go build -o /tmp/uisce_backend_server ./cmd/server; then
  echo "[check-build] ✅ compile OK ($(du -h /tmp/uisce_backend_server | cut -f1))"
else
  echo "[check-build] ❌ compile FAILED"; exit 1
fi

echo "[check-build] go vet ./... ..."
go vet ./... 2>&1 | tee /tmp/uisce_vet.log
vet_lines=$(wc -l < /tmp/uisce_vet.log)
echo "[check-build] vet printed ${vet_lines} warning lines (vet exit non-zero only on actual problems)"

rm -f /tmp/uisce_backend_server
