#!/usr/bin/env bash
# Dev helper: wait for API gateway then start frontend dev server
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
GATEWAY_HEALTH_URL="${GATEWAY_HEALTH_URL:-http://localhost:8001/api/_debug/headers}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-60}"

echo "Waiting for API gateway at $GATEWAY_HEALTH_URL (timeout ${WAIT_TIMEOUT_SECONDS}s)"
end=$((SECONDS + WAIT_TIMEOUT_SECONDS))
while :; do
  if curl -fsS "$GATEWAY_HEALTH_URL" >/dev/null 2>&1; then
    echo "API gateway is healthy"
    break
  fi
  if [ $SECONDS -ge $end ]; then
    echo "Timed out waiting for API gateway after ${WAIT_TIMEOUT_SECONDS}s" >&2
    exit 1
  fi
  sleep 1
done

cd "$FRONTEND_DIR/frontend"
echo "Starting frontend dev server in $FRONTEND_DIR/frontend"
exec npm run dev
#!/usr/bin/env bash
# Dev helper: wait for API gateway then start frontend dev server
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
GATEWAY_HEALTH_URL="${GATEWAY_HEALTH_URL:-http://localhost:8001/api/_debug/headers}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-60}"

echo "Waiting for API gateway at $GATEWAY_HEALTH_URL (timeout ${WAIT_TIMEOUT_SECONDS}s)"
end=$((SECONDS + WAIT_TIMEOUT_SECONDS))
while :; do
  if curl -fsS "$GATEWAY_HEALTH_URL" >/dev/null 2>&1; then
    echo "API gateway is healthy"
    break
  fi
  if [ $SECONDS -ge $end ]; then
    echo "Timed out waiting for API gateway after ${WAIT_TIMEOUT_SECONDS}s" >&2
    exit 1
  fi
  sleep 1
done

cd "$FRONTEND_DIR"
echo "Starting frontend dev server in $FRONTEND_DIR"
exec npm run dev
