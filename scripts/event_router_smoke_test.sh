#!/usr/bin/env bash
set -euo pipefail

# Simple smoke test for Event Router
# - Creates a Redpanda topic
# - Posts an event to the Event Router /events endpoint with route_queue set to that topic
# - Consumes the message from the Redpanda topic to verify delivery

EVENT_ROUTER_URL=${EVENT_ROUTER_URL:-http://localhost:8081/events}
REDPANDA_CONTAINER=${REDPANDA_CONTAINER:-semlayer-redpanda}
TOPIC="eventrouter-smoke-$(date +%s)"

echo "Smoke test: topic=${TOPIC}, event-router=${EVENT_ROUTER_URL}, redpanda container=${REDPANDA_CONTAINER}"

# Create topic
echo "Creating topic ${TOPIC}..."
docker exec ${REDPANDA_CONTAINER} rpk topic create "${TOPIC}" >/dev/null

# Insert a temporary config directly in the database so the Event Router will route to our topic
DB_URL=${DB_URL:-postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable}

echo "Inserting temporary event_config pointing to topic ${TOPIC}..."
CONFIG_ID=$(psql "$DB_URL" -t -A -c "
INSERT INTO event_configs (tenant_id, bo_type, event_type, filter_json, route_queue)
VALUES ('00000000-0000-0000-0000-000000000000', 'test', 'fieldchange', '{}', '${TOPIC}')
RETURNING id;
")

if [ -z "$CONFIG_ID" ]; then
  echo "Failed to insert event_config." >&2
  exit 1
fi

echo "Inserted config id: $CONFIG_ID"

cleanup_config() {
  echo "Cleaning up event_config $CONFIG_ID"
  psql "$DB_URL" -c "DELETE FROM event_configs WHERE id = '${CONFIG_ID}';" >/dev/null || true
}

# Post event to event-router
payload=$(cat <<EOF
{
  "tenant_id": "00000000-0000-0000-0000-000000000000",
  "bo_type": "test",
  "bo_id": "test-001",
  "event_type": "fieldchange",
  "field_name": "status",
  "old_value": "pending",
  "new_value": "processed",
  "changed_by": "smoke-test",
  "custom_data": {}
}
EOF
)

# Try host POST first
echo "Posting event to Event Router (host)..."
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$payload" "$EVENT_ROUTER_URL" || true)

if [ "$http_code" -eq 200 ]; then
  echo "Event Router (host) accepted event"
else
  echo "Host POST returned $http_code; trying inside container..."
  # Find event router container name
  ER_CONTAINER=$(docker ps --format '{{.Names}} {{.Image}}' | grep -i event-router | awk '{print $1}' | head -n1 || true)
  if [ -n "$ER_CONTAINER" ]; then
    echo "Found event-router container: $ER_CONTAINER — posting inside container"
    # Create payload file inside container and POST using wget (some containers don't have curl)
    docker exec "$ER_CONTAINER" sh -c 'cat > /tmp/_payload.json <<"JSON"\n'"$payload"'\nJSON\n; wget --header="Content-Type: application/json" --post-file=/tmp/_payload.json -q -O - http://localhost:8080/events; EXIT_CODE=$?; echo "EXIT:$EXIT_CODE"'
    # Check exit code from wget inside container
    if [ $? -ne 0 ]; then
      echo "Container POST failed" >&2
      cleanup_config
      exit 1
    fi
  else
    echo "No event-router container found and host POST failed ($http_code)." >&2
    cleanup_config
    exit 1
  fi
fi

# Try to consume from topic (retry loop)
echo "Consuming from topic ${TOPIC}..."
for i in $(seq 1 12); do
  out=$(docker exec ${REDPANDA_CONTAINER} rpk topic consume "${TOPIC}" -o start -n 1 -f '%k %v\n' 2>/dev/null || true)
  if [ -n "${out}" ]; then
    echo "Received: ${out}"
    echo "Event Router smoke test PASSED"
    if [ -n "$CONFIG_ID" ]; then
      cleanup_config
    fi
    exit 0
  fi
  echo "Attempt $i: no message yet, retrying..."
  sleep 1
done

echo "Event Router smoke test FAILED: no message consumed from ${TOPIC}" >&2
# Cleanup config on failure
if [ -n "$CONFIG_ID" ]; then
  cleanup_config
fi
exit 1
