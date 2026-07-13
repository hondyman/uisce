#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# LOCAL BACKEND DEVELOPMENT WITH INFISICAL
# ==============================================================================
# Run backend Go server locally on MacBook with Infisical secrets injection
# Usage: ./scripts/start-backend-local.sh
#
# Prerequisites:
#   1. Install Infisical CLI: brew install infisical
#   2. Copy .env.infisical.example to .env.infisical and add your credentials
#   3. Or export INFISICAL_TOKEN before running
#
# This runs Go directly on your MacBook - Docker is only on the remote server.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_DIR="$SCRIPT_DIR/logs"
PIDFILE="$SCRIPT_DIR/.backend.pid"
PORT="${PORT:-8080}"
TIMESTAMP="$(date '+%Y%m%d_%H%M%S')"
LOG_FILE="$LOG_DIR/backend_${TIMESTAMP}.log"

mkdir -p "$LOG_DIR"

info() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*"; }
err() { echo "[ERROR] $*"; }

# ==============================================================================
# INFISICAL CONFIGURATION
# ==============================================================================
INFISICAL_ENV_FILE="$SCRIPT_DIR/.env.infisical"
if [ -f "$INFISICAL_ENV_FILE" ]; then
  info "Loading Infisical config from $INFISICAL_ENV_FILE..."
  set -a
  source "$INFISICAL_ENV_FILE"
  set +a
elif [ -n "${INFISICAL_TOKEN:-}" ]; then
  info "Using Infisical from environment..."
else
  err "No Infisical config found!"
  echo "   Copy .env.infisical.example to .env.infisical and add your credentials"
  echo "   Or export INFISICAL_TOKEN"
  exit 1
fi

INFISICAL_DOMAIN="${INFISICAL_DOMAIN:-http://100.84.50.65:8085}"
INFISICAL_ENVIRONMENT="${INFISICAL_ENVIRONMENT:-dev}"

export INFISICAL_DOMAIN
export INFISICAL_TOKEN
export INFISICAL_PROJECT_ID
export INFISICAL_ENVIRONMENT

echo ""
info "Infisical Domain: $INFISICAL_DOMAIN"
info "Infisical Project: ${INFISICAL_PROJECT_ID:-not set}"
info "Infisical Environment: $INFISICAL_ENVIRONMENT"
echo ""

# Verify infisical CLI
if ! command -v infisical >/dev/null 2>&1; then
  err "'infisical' CLI not found"
  echo "   Install: brew install infisical"
  exit 1
fi

# Check required vars
if [ -z "${INFISICAL_TOKEN:-}" ] || [ -z "${INFISICAL_PROJECT_ID:-}" ]; then
  err "INFISICAL_TOKEN and INFISICAL_PROJECT_ID are required"
  exit 1
fi

# Test access
info "Testing Infisical access..."
if infisical secrets get --projectId="${INFISICAL_PROJECT_ID}" --env="${INFISICAL_ENVIRONMENT}" --path="/core" DATABASE_URL >/dev/null 2>&1; then
  info "✅ Infisical token is valid"
else
  err "Cannot access secrets - check token permissions"
  exit 1
fi

# Kill stale process
if [ -f "$PIDFILE" ]; then
  OLD_PID=$(cat "$PIDFILE" 2>/dev/null || echo "")
  if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
    info "Killing stale backend (PID: $OLD_PID)"
    kill "$OLD_PID" 2>/dev/null || true
    sleep 1
  fi
  rm -f "$PIDFILE"
fi

# Kill process on port
if lsof -ti:"$PORT" >/dev/null 2>&1; then
  info "Port $PORT in use, killing..."
  lsof -ti:"$PORT" | xargs kill 2>/dev/null || true
  sleep 1
fi

# ==============================================================================
# START BACKEND
# ==============================================================================

cd "$BACKEND_DIR"

info "🚀 Starting Uisce Backend on port $PORT..."
info "   Frontend should connect to: http://localhost:$PORT"
info "   Press Ctrl+C to stop"
echo ""

exec infisical run \
  --projectId="${INFISICAL_PROJECT_ID}" \
  --env="${INFISICAL_ENVIRONMENT}" \
  --command="go run ./cmd/server/main.go"
