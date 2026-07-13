#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# LOCAL FRONTEND DEVELOPMENT
# ==============================================================================
# Run React frontend locally on MacBook
# Usage: ./scripts/start-frontend.sh
#
# Prerequisites:
#   1. npm install in frontend/frontend directory
#   2. Backend should be running on localhost:8080
#
# This runs React directly on your MacBook - Docker is only on the remote server.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/frontend/frontend"

# Check if frontend dir exists
if [ ! -d "$FRONTEND_DIR" ]; then
  echo "[ERROR] Frontend directory not found: $FRONTEND_DIR"
  exit 1
fi

# Check if node_modules exists
if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  echo "[INFO] node_modules not found, running npm install..."
  cd "$FRONTEND_DIR"
  npm install
fi

cd "$FRONTEND_DIR"

echo ""
echo "[INFO] Starting React frontend..."
echo "[INFO] Frontend will connect to backend at: http://localhost:8080"
echo "[INFO] Frontend dev server URL: http://localhost:5173 (or next available)"
echo ""

exec npm run dev
