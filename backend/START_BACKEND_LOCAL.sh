#!/usr/bin/env bash
# Local development start script — runs the Go backend on :8080 (the port
# the frontend vite proxy targets). No cross-compilation, no remote deploy.
#
# Required env (fail-fast with named error if missing):
#   DATABASE_URL, JWT_SECRET, API_TOKEN_ENCRYPTION_KEY
# All three must be set in backend/.env (which is gitignored).
#
# TEMPORAL_HOST and TEMPORAL_RETRY_ATTEMPTS have working defaults (non-secrets;
# extend with :? if your setup needs them mandatory).
#
# Usage:
#   ./START_BACKEND_LOCAL.sh
#   ./START_BACKEND_LOCAL.sh   # safe to re-run while server is up (kill + restart)

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

source .env 2>/dev/null || true

# Bare `export NAME` only exports names .env actually defined (no empty-string
# false positives). Without this, `source` leaves them as shell-local vars and
# the server silently loads no JWKS keys, rejecting every Keycloak token.
# Deliberately NOT `set -a`: that would export all of .env, including flags
# (e.g. tenant-header fallbacks) that must stay opt-in.
export KEYCLOAK_JWKS_URL KEYCLOAK_JWKS_REFRESH_INTERVAL

export DATABASE_URL="${DATABASE_URL:?DATABASE_URL missing — set it in backend/.env}"
export JWT_SECRET="${JWT_SECRET:?JWT_SECRET missing from backend/.env}"
export API_TOKEN_ENCRYPTION_KEY="${API_TOKEN_ENCRYPTION_KEY:?API_TOKEN_ENCRYPTION_KEY missing}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-100.84.50.65:7233}"
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-2}"

# Kill anything already on 8080 — SIGKILL the compiled binary directly.
# (SIGTERM alone doesn't reliably kill a Go http.Server in reasonable time.)
pkill -9 -f "/tmp/uisce-server" 2>/dev/null || true
sleep 1

echo "🚀 Starting backend locally on :8080"
go build -o /tmp/uisce-server ./cmd/server
exec /tmp/uisce-server
