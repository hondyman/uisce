#!/usr/bin/env bash
set -euo pipefail
url=${1:-http://localhost:8087/ready}
max_attempts=${2:-60}
interval=${3:-5}

for i in $(seq 1 "$max_attempts"); do
  if curl -sSf "$url" > /dev/null 2>&1; then
    echo "READY: $url"
    exit 0
  fi
  echo "waiting for ready... ($i/$max_attempts)"
  sleep "$interval"
done

echo "ERROR: service did not become ready: $url"
curl -sS "$url" || true
exit 1
