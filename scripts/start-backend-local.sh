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

# Start the backend. Prefer 'go run' in dev, fall back to built binary if present.
cd "$BACKEND_DIR"

if command -v go >/dev/null 2>&1; then
  info "Launching backend using 'go run' (logs -> $LOG_FILE)"
  PORT="$PORT" nohup go run ./cmd/server > "$LOG_FILE" 2>&1 &
  NEW_PID=$!
else
  if [ -x "./server" ]; then
    info "Launching built server binary (logs -> $LOG_FILE)"
    PORT="$PORT" nohup ./server > "$LOG_FILE" 2>&1 &
    NEW_PID=$!
  else
    err "Neither 'go' is available nor './server' binary exists in $BACKEND_DIR"
    exit 1
  fi
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
