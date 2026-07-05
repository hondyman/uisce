#!/usr/bin/env bash
set -euo pipefail

# Local helper: start the backend in the foreground or background with a pidfile
# Usage:
#   ./scripts/start-backend-local.sh            # starts backend using PORT (default 8080)
#   PORT=29080 ./scripts/start-backend-local.sh # start on different port

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_DIR="$SCRIPT_DIR/logs"
PIDFILE="$SCRIPT_DIR/.backend.pid"
PORT=${PORT:-8080}
TIMESTAMP="$(date '+%Y%m%d_%H%M%S')"
LOG_FILE="$LOG_DIR/backend_${TIMESTAMP}.log"

mkdir -p "$LOG_DIR"

info() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*"; }
err() { echo "[ERROR] $*"; }

info "Starting backend (PORT=$PORT)"

# Sanity-check that Postgres is reachable before trying to start the server.
# The backend will crash immediately if it cannot connect to the database.
# Try to read the DSN and host from the project's config.yaml; fall back to the Docker Compose default.
CONFIG_FILE="$SCRIPT_DIR/config.yaml"
DATABASE_URL="${DATABASE_URL:-}"
if [ -z "$DATABASE_URL" ] && [ -f "$CONFIG_FILE" ] && command -v sed >/dev/null 2>&1; then
  DATABASE_URL="$(sed -n 's/^dsn: *"\(.*\)".*/\1/p' "$CONFIG_FILE" | head -n1)"
fi

PG_HOST="${PGHOST:-}"
if [ -z "$PG_HOST" ] && [ -n "$DATABASE_URL" ]; then
  PG_HOST="$(echo "$DATABASE_URL" | sed -n 's/.*@\([^:]*\):.*/\1/p')"
fi
PG_HOST="${PG_HOST:-100.84.50.65}"
PG_PORT="${PGPORT:-5432}"

if command -v nc >/dev/null 2>&1; then
  if ! nc -z "$PG_HOST" "$PG_PORT" >/dev/null 2>&1; then
    err "Postgres is not reachable at $PG_HOST:$PG_PORT."
    err "Start the full stack with ./START_FULL_SYSTEM.sh, or start Postgres manually."
    exit 1
  fi
else
  warn "'nc' not found; skipping Postgres readiness check"
fi

export DATABASE_URL

# Pull JWT secret from config.yaml if one is configured; otherwise the backend
# falls back to its development default.
JWT_SECRET="${JWT_SECRET:-}"
if [ -z "$JWT_SECRET" ] && [ -f "$CONFIG_FILE" ] && command -v sed >/dev/null 2>&1; then
  JWT_SECRET="$(sed -n 's/^jwt_secret: *"\(.*\)".*/\1/p' "$CONFIG_FILE" | head -n1)"
fi
if [ -n "$JWT_SECRET" ]; then
  export JWT_SECRET
fi

# Keycloak JWKS endpoint for validating RS256 tokens from the browser.
export KEYCLOAK_JWKS_URL="${KEYCLOAK_JWKS_URL:-https://100.84.50.65:8443/realms/uisce/protocol/openid-connect/certs}"

# Don't block server startup for minutes waiting on Temporal in local dev.
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-1}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-temporal:7233}"

# Kill pidfile-managed process if present
if [ -f "$PIDFILE" ]; then
  OLD_PID=$(cat "$PIDFILE" 2>/dev/null || echo "")
  if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" >/dev/null 2>&1; then
    info "Killing stale backend process (PID: $OLD_PID)"
    kill -9 "$OLD_PID" >/dev/null 2>&1 || true
    sleep 1
  fi
  rm -f "$PIDFILE" >/dev/null 2>&1 || true
fi

# Kill any process listening on the requested port (helpful for quick dev restarts)
if command -v lsof >/dev/null 2>&1 && lsof -ti:"$PORT" >/dev/null 2>&1; then
  info "Killing any process listening on port $PORT"
  lsof -ti:"$PORT" | xargs -r kill -9 2>/dev/null || true
  sleep 1
fi

# Start the backend. Build once and run the binary — this avoids the output-
# redirection quirks of 'go run' and starts reliably under nohup.
cd "$BACKEND_DIR"

if ! command -v go >/dev/null 2>&1; then
  if [ -x "./server" ]; then
    info "Launching existing built server binary (logs -> $LOG_FILE)"
    PORT="$PORT" nohup ./server > "$LOG_FILE" 2>&1 &
    NEW_PID=$!
  else
    err "Neither 'go' is available nor './server' binary exists in $BACKEND_DIR"
    exit 1
  fi
else
  info "Building backend binary (logs -> $LOG_FILE)"
  if ! go build -o ./server ./cmd/server >> "$LOG_FILE" 2>&1; then
    err "Backend build failed. See $LOG_FILE for details."
    exit 1
  fi
  info "Launching built server binary (logs -> $LOG_FILE)"
  PORT="$PORT" nohup ./server > "$LOG_FILE" 2>&1 &
  NEW_PID=$!
fi

# Record pidfile at repo root so other scripts can find/kill it
echo "$NEW_PID" > "$PIDFILE"
info "Backend started (PID: $NEW_PID)"
info "Log file: $LOG_FILE"

# Tail the log file in foreground when not run from a terminal background
# If the script was invoked interactively, follow the log; otherwise exit.
if [ -t 1 ]; then
  echo "Tailing log (press Ctrl-C to stop)"
  tail -f "$LOG_FILE"
else
  echo "Started in background"
fi
