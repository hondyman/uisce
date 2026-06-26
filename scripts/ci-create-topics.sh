#!/usr/bin/env bash
set -euo pipefail

RETRY=${RETRY:-10}
SLEEP=${SLEEP:-2}
REDPANDA_COMPOSE_FILE="docker-compose.integration.yml"

for i in $(seq 1 "$RETRY"); do
  # Try to create the topic (may fail if it already exists)
  docker compose -f "$REDPANDA_COMPOSE_FILE" exec -T redpanda rpk topic create approval.events --partitions 1 --replicas 1 >/dev/null 2>&1 || true

  # Verify topic exists
  if docker compose -f "$REDPANDA_COMPOSE_FILE" exec -T redpanda rpk topic describe approval.events >/dev/null 2>&1; then
    echo "topic approval.events ready"
    echo "--- topic describe ---"
    docker compose -f "$REDPANDA_COMPOSE_FILE" exec -T redpanda rpk topic describe approval.events || true
    echo "----------------------"
    exit 0
  fi

  echo "retrying topic create (approval.events) ($i/$RETRY)"
  sleep "$SLEEP"
done

echo "failed to ensure topic approval.events"
docker compose -f "$REDPANDA_COMPOSE_FILE" logs redpanda --no-log-prefix --tail 50 || true
exit 1
