#!/usr/bin/env bash
# Bump file-descriptor limit before invoking anything that walks the module
# graph (gopls, go build ./..., go test ./...). With the default macOS
# ulimit (256) the go toolchain hits "too many open files" once it has to
# scan a few hundred .go files, and downstream packages then appear as
# "no required module provides package ... — but the files exist" — which
# is exactly the failure mode this repo hits when gopls auto-indexes.
#
# The IDE problems panel only clears after both:
#   (a) ulimit -n is raised (this script), AND
#   (b) Go's build cache is invalidated (`go clean -cache` once).
#
# Re-run this script any time the Problems panel shows "missing package"
# errors for files that clearly exist on disk.

ulimit -n 16384 2>/dev/null || true
echo "[start-backend] ulimit -n now $(ulimit -n)"

cd "$(dirname "$0")"
SCRIPT_DIR="$(pwd)"

# Clear stale build cache so any half-built packages get rebuilt against the
# higher FD limit. This is a one-shot fix; subsequent builds use the cache.
if [ "${UISCE_CLEAN_CACHE:-1}" = "1" ]; then
  echo "[start-backend] invalidating Go build cache (set UISCE_CLEAN_CACHE=0 to skip)"
  go clean -cache 2>/dev/null || true
fi

# Actually start the backend server (this script's name implies it should).
echo "[start-backend] launching backend server via scripts/start-backend-local.sh"
exec "$SCRIPT_DIR/scripts/start-backend-local.sh"
