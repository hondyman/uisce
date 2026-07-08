#!/usr/bin/env bash
set -euo pipefail

# Local helper: start the backend in the foreground or background with a pidfile
# Usage:
#   ./scripts/start-backend-local.sh            # starts backend using PORT (default 8080)
#   PORT=29080 ./scripts/start-backend-local.sh # start on different port
#
# Secrets are injected via Infisical (project 9af25976-bafc-4895-a057-2effec13c620, env dev).
# If Infisical is not installed or auth fails, the script falls back to whatever is
# already in the environment — useful in CI or Docker contexts where secrets are
# baked in via other means.
#
# Override Infisical settings:
#   INFISICAL_API_URL   - default http://100.84.50.65:8085
#   INFISICAL_PROJECT   - default 9af25976-bafc-4895-a057-2effec13c620
#   INFISICAL_ENV       - default dev
#   SKIP_INFISICAL=1    - bypass Infisical entirely (use env as-is)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_DIR="$SCRIPT_DIR/logs"
PIDFILE="$SCRIPT_DIR/.backend.pid"
PORT=${PORT:-8080}
TIMESTAMP="$(date '+%Y%m%d_%H%M%S')"
LOG_FILE="$LOG_DIR/backend_${TIMESTAMP}.log"

# ── Infisical config ────────────────────────────────────────────────────────
INFISICAL_API_URL="${INFISICAL_API_URL:-http://100.84.50.65:8085}"
INFISICAL_PROJECT="${INFISICAL_PROJECT:-9af25976-bafc-4895-a057-2effec13c620}"
INFISICAL_ENV="${INFISICAL_ENV:-dev}"
SKIP_INFISICAL="${SKIP_INFISICAL:-0}"

# ── Hardened defaults for remote DB/services ───────────────────────────────
# These are set BEFORE Infisical runs so Infisical values always win,
# but they protect against a missing secret leaving the binary blind.
export DATABASE_URL="${DATABASE_URL:-postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-100.84.50.65:7233}"
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-1}"
export KEYCLOAK_JWKS_URL="${KEYCLOAK_JWKS_URL:-https://100.84.50.65:8443/realms/uisce/protocol/openid-connect/certs}"

# =====================================================================
# AUTOMATIC PORT EVICTION (ANTI-ZOMBIE GUARD)
# =====================================================================
# Runs BEFORE go build / infisical run to prevent `bind: address already in use`.
# macOS-native: uses lsof + kill -9 to cleanly evict any stuck listeners.
TARGET_PORT="${PORT:-8080}"

echo "⏳ Checking for zombie services occupying port :${TARGET_PORT}..."

# Check if anything is listening on the target TCP port on macOS
# -t = terse: returns only the raw PID(s), no headers/usernames.
ZOMBIE_PID=$(lsof -t -i tcp:"${TARGET_PORT}" 2>/dev/null || true)

# Fixed: `[ -not -z ]` is invalid bash; use `[ -n ]` for "non-zero length".
if [ -n "${ZOMBIE_PID}" ]; then
    echo "⚠️  Found stale process(es) [PID: ${ZOMBIE_PID}] occupying port :${TARGET_PORT}."
    echo "🔥 Forcing eviction of ghost processes..."

    # Forcefully terminate the process handles hogging the port
    echo "${ZOMBIE_PID}" | xargs kill -9 2>/dev/null || true

    # Brief pause to let the network socket release cleanly in the macOS kernel
    sleep 1
    echo "✅ Port :${TARGET_PORT} successfully cleared and unblocked!"
else
    echo "✅ Port :${TARGET_PORT} is vacant and ready for binding."
fi
# =====================================================================

mkdir -p "$LOG_DIR"

info() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*"; }
err()  { echo "[ERROR] $*"; }

info "Starting backend (PORT=$PORT)"

# ── Postgres reachability check ────────────────────────────────────────────
# Parse host from DATABASE_URL for the nc check.
PG_HOST="${PGHOST:-}"
if [ -z "$PG_HOST" ] && [ -n "${DATABASE_URL:-}" ]; then
  PG_HOST="$(echo "$DATABASE_URL" | sed -n 's/.*@\([^:/?]*\).*/\1/p')"
fi
PG_HOST="${PG_HOST:-100.84.50.65}"
PG_PORT="${PGPORT:-5432}"

if command -v nc >/dev/null 2>&1; then
  if ! nc -z "$PG_HOST" "$PG_PORT" >/dev/null 2>&1; then
    err "Postgres is not reachable at $PG_HOST:$PG_PORT."
    err "Ensure the remote host is up, or run ./START_FULL_SYSTEM.sh."
    exit 1
  fi
  info "Postgres reachable at $PG_HOST:$PG_PORT ✓"
else
  warn "'nc' not found; skipping Postgres readiness check"
fi

# ── Pull JWT secret from config.yaml if not already set ───────────────────
CONFIG_FILE="$SCRIPT_DIR/config.yaml"
JWT_SECRET="${JWT_SECRET:-}"
if [ -z "$JWT_SECRET" ] && [ -f "$CONFIG_FILE" ] && command -v sed >/dev/null 2>&1; then
  JWT_SECRET="$(sed -n 's/^jwt_secret: *"\(.*\)".*/\1/p' "$CONFIG_FILE" | head -n1)"
fi
if [ -n "$JWT_SECRET" ]; then
  export JWT_SECRET
fi

# ── Kill stale pidfile-managed process ────────────────────────────────────
if [ -f "$PIDFILE" ]; then
  OLD_PID=$(cat "$PIDFILE" 2>/dev/null || echo "")
  if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" >/dev/null 2>&1; then
    info "Killing stale backend process (PID: $OLD_PID)"
    kill -9 "$OLD_PID" >/dev/null 2>&1 || true
    sleep 1
  fi
  rm -f "$PIDFILE" >/dev/null 2>&1 || true
fi

# ── Build the server binary ───────────────────────────────────────────────
cd "$BACKEND_DIR"

if command -v go >/dev/null 2>&1; then
  info "Building backend binary..."
  if ! go build -o ./server ./cmd/server >> "$LOG_FILE" 2>&1; then
    err "Backend build failed. See $LOG_FILE for details."
    exit 1
  fi
  info "Build complete ✓"
elif [ ! -x "./server" ]; then
  err "Neither 'go' is available nor './server' binary exists in $BACKEND_DIR"
  exit 1
fi

# ── Compose the launch command ────────────────────────────────────────────
# We always pass DATABASE_URL explicitly on the command line so it wins
# over any stale env var, even when Infisical is skipped.
LAUNCH_CMD="PORT=$PORT DATABASE_URL=\"$DATABASE_URL\" TEMPORAL_HOST=\"$TEMPORAL_HOST\" nohup ./server > \"$LOG_FILE\" 2>&1"

if [ "$SKIP_INFISICAL" = "1" ]; then
  warn "SKIP_INFISICAL=1 — launching without Infisical secret injection"
  info "Launching backend (logs -> $LOG_FILE)"
  eval "$LAUNCH_CMD" &
  NEW_PID=$!

elif command -v infisical >/dev/null 2>&1; then
  info "Injecting secrets via Infisical (project=$INFISICAL_PROJECT env=$INFISICAL_ENV)"
  INFISICAL_API_URL="$INFISICAL_API_URL" \
    infisical run \
      --projectId "$INFISICAL_PROJECT" \
      --env      "$INFISICAL_ENV" \
      -- sh -c "$LAUNCH_CMD" &
  NEW_PID=$!

else
  warn "Infisical CLI not found — launching without secret injection."
  warn "Install with: brew install infisical/get-cli/infisical"
  warn "Falling back to hardened defaults (DATABASE_URL=$DATABASE_URL)"
  eval "$LAUNCH_CMD" &
  NEW_PID=$!
fi

# ── Record pidfile ─────────────────────────────────────────────────────────
echo "$NEW_PID" > "$PIDFILE"
info "Backend started (PID: $NEW_PID)"
info "Log: $LOG_FILE"
info "Port: $PORT"

# Brief pause so startup errors surface immediately
sleep 2
if ! kill -0 "$NEW_PID" >/dev/null 2>&1; then
  err "Backend process died immediately after launch. Check $LOG_FILE"
  tail -20 "$LOG_FILE" || true
  exit 1
fi
info "Backend is alive ✓"

# ── Tail log if running interactively ─────────────────────────────────────
if [ -t 1 ]; then
  echo "Tailing log (press Ctrl-C to stop tailing — server keeps running)"
  tail -f "$LOG_FILE"
else
  echo "Started in background"
fi
