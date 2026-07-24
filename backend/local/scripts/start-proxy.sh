#!/usr/bin/env bash
set -euo pipefail

# Start the local proxy in background and write pidfile
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PIDFILE="$ROOT_DIR/proxy.pid"

if [ -f "$PIDFILE" ]; then
  echo "pidfile exists at $PIDFILE - proxy may already be running"
  exit 1
fi

echo "Starting proxy..."
cd "$ROOT_DIR"
nohup go run ./cmd/proxy > proxy.log 2>&1 &
echo $! > "$PIDFILE"
echo "proxy started (pid $(cat $PIDFILE)), logs: $ROOT_DIR/proxy.log"
