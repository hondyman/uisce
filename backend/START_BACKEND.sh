#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Cleaning up orphan backend processes ==="

# 1. Kill any process listening on port 8080
PID_8080=$(lsof -ti :8080 2>/dev/null || true)
if [ -n "$PID_8080" ]; then
    echo "Killing process listening on port 8080 (PID: $PID_8080)"
    kill -9 $PID_8080 2>/dev/null || true
    sleep 1
fi

# 2. Kill stray uisce binaries by process name / binary path
pkill -9 -f "uisce-backend" 2>/dev/null || true
pkill -9 -f "cmd/server" 2>/dev/null || true
sleep 1

echo "=== Building backend ==="
go build -buildvcs=false -o /tmp/uisce-backend ./cmd/server

echo "=== Starting backend ==="
if [ -f "$SCRIPT_DIR/.env" ]; then
    echo "Loading environment from .env..."
    set -a
    source "$SCRIPT_DIR/.env"
    set +a
fi
export JWT_SECRET="${JWT_SECRET:-test-secret}"
export PORT="${PORT:-8080}"
export DATABASE_URL="${DATABASE_URL:-postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable}"

/tmp/uisce-backend > /tmp/uisce-backend.log 2>&1 &
BACKEND_PID=$!

echo "Backend started (PID: $BACKEND_PID)"
echo "Logs streaming to /tmp/uisce-backend.log"

# Wait for server readiness
for i in {1..10}; do
    if curl -s http://localhost:${PORT}/health >/dev/null 2>&1; then
        echo "✅ Backend is healthy and responding on http://localhost:${PORT}"
        break
    fi
    sleep 1
done

wait $BACKEND_PID
