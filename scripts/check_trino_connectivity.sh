#!/usr/bin/env bash
# Simple connectivity check to Trino HTTP endpoint
set -euo pipefail

if [ -z "${TRINO_HOST:-}" ]; then
  echo "TRINO_HOST not set"
  exit 2
fi
PORT=${TRINO_PORT:-8080}
URL="http://${TRINO_HOST}:${PORT}/v1/info"

echo "Checking Trino at $URL"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$URL")
if [ "$HTTP_STATUS" -ne 200 ]; then
  echo "Trino endpoint returned HTTP $HTTP_STATUS"
  exit 1
fi

echo "Trino endpoint reachable (HTTP 200)"
