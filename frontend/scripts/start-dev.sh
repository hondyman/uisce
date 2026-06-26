#!/usr/bin/env bash
set -euo pipefail

# Script to ensure port 5173 is free and start the Vite dev server.
# Works on macOS/Linux/Alpine.

# Change to the frontend directory (where this script is located, go up one level)
cd "$(dirname "$0")/.."

PORT=${PORT:-3000}

echo "[start-dev] ensuring port ${PORT} is free..."

# Find any PIDs listening on the port
# Try multiple methods for better compatibility
PIDS=""

# Method 1: Try lsof (works on macOS/Linux)
if command -v lsof &> /dev/null; then
  PIDS=$(lsof -nP -iTCP:${PORT} -sTCP:LISTEN 2>/dev/null | awk 'NR>1 {print $2}' || true)
fi

# Method 2: If lsof didn't work or isn't available, try fuser (Alpine compatible)
if [ -z "${PIDS}" ] && command -v fuser &> /dev/null; then
  PIDS=$(fuser ${PORT}/tcp 2>/dev/null || true)
fi

if [ -n "${PIDS}" ]; then
  echo "[start-dev] found processes on ${PORT}: ${PIDS}"
  for p in ${PIDS}; do
    # Validate that p is actually a number (PID)
    if [ -n "$p" ] && echo "$p" | grep -qE '^[0-9]+$'; then
      echo "[start-dev] killing PID ${p}..."
      kill -9 "${p}" 2>/dev/null || true
    fi
  done
  # brief pause to allow socket to be released
  sleep 1
else
  echo "[start-dev] no process listening on ${PORT}."
fi

# Double-check the port is free (if lsof available)
if command -v lsof &> /dev/null; then
  if lsof -nP -iTCP:${PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo "[start-dev] port ${PORT} still in use after kill; aborting." >&2
    exit 1
  fi
elif command -v fuser &> /dev/null; then
  if fuser ${PORT}/tcp >/dev/null 2>&1; then
    echo "[start-dev] port ${PORT} still in use after kill; aborting." >&2
    exit 1
  fi
fi

echo "[start-dev] starting Vite on port ${PORT}..."

# Prefer local node_modules binary to avoid relying on global installs
VITE_BIN="./node_modules/.bin/vite"
# Also check the monorepo-root installation path used by our Docker image
VITE_BIN_ROOT="/app/node_modules/.bin/vite"
if [ -x "${VITE_BIN}" ]; then
  exec env PORT=${PORT} "${VITE_BIN}" -- --host 0.0.0.0
elif [ -x "${VITE_BIN_ROOT}" ]; then
  echo "[start-dev] found Vite at ${VITE_BIN_ROOT}, using that"
  exec env PORT=${PORT} "${VITE_BIN_ROOT}" -- --host 0.0.0.0
else
  echo "[start-dev] ${VITE_BIN} not found or not executable; falling back to npx";
  # Use --yes to avoid interactive prompts in CI/container environments
  exec env PORT=${PORT} npx --yes vite -- --host 0.0.0.0
fi
