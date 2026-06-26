#!/usr/bin/env bash
set -euo pipefail

# Simple migration runner wrapper: retry running the migrate binary inside the runner
# container until it succeeds (or until attempts exhausted). This helps when DB
# is still coming up.

COMPOSE_FILE=docker-compose.backend.yml
ATTEMPTS=${ATTEMPTS:-10}
SLEEP=${SLEEP:-3}

i=0
while [ $i -lt $ATTEMPTS ]; do
  i=$((i+1))
  echo "Attempt $i/$ATTEMPTS: running migrate..."
  if docker compose -f "$COMPOSE_FILE" run --rm runner "./migrate"; then
    echo "Migration succeeded"
    exit 0
  fi
  echo "Migration failed, sleeping $SLEEP seconds before retrying..."
  sleep $SLEEP
done

echo "Migration failed after $ATTEMPTS attempts"
exit 1
