#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PIDFILE="$ROOT_DIR/proxy.pid"

if [ ! -f "$PIDFILE" ]; then
  echo "no pidfile at $PIDFILE"
  exit 1
fi

PID=$(cat "$PIDFILE")
echo "Stopping proxy pid $PID"
kill "$PID" || true
rm -f "$PIDFILE"
echo "stopped"
