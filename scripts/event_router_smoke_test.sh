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

# Insert a temporary config in Hasura so the Event Router will route to our topic
HASURA_URL=${HASURA_URL:-http://localhost:8080/v1/graphql}
HASURA_ADMIN_SECRET=${HASURA_ADMIN_SECRET:-}

echo "Inserting temporary event_config in Hasura pointing to topic ${TOPIC}..."
GRAPHQL_PAYLOAD=$(cat <<JSON
{"query":"mutation InsertConfig($objects: [event_configs_insert_input!]!) { insert_event_configs(objects: $objects) { returning { id } } }","variables":{"objects":[{"tenant_id":"00000000-0000-0000-0000-000000000000","bo_type":"test","event_type":"fieldchange","filter_json":"{}","route_queue":"${TOPIC}"}]}}
JSON
)

if [ -n "$HASURA_ADMIN_SECRET" ]; then
  resp=$(curl -s -X POST -H "Content-Type: application/json" -H "x-hasura-admin-secret: $HASURA_ADMIN_SECRET" -d "$GRAPHQL_PAYLOAD" "$HASURA_URL")
else
  resp=$(curl -s -X POST -H "Content-Type: application/json" -d "$GRAPHQL_PAYLOAD" "$HASURA_URL")
fi

HASURA_ID=$(echo "$resp" | jq -r '.data.insert_event_configs.returning[0].id // empty')
if [ -z "$HASURA_ID" ]; then
  echo "Failed to insert Hasura config. Response: $resp" >&2
  exit 1
fi

echo "Inserted config id: $HASURA_ID"

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
      # Cleanup Hasura config
      echo "Cleaning up Hasura config $HASURA_ID"
      DELETE_PAYLOAD=$(jq -n --arg id "$HASURA_ID" '{"query":"mutation DeleteConfig($id: uuid!){delete_event_configs_by_pk(id: $id){id}}","variables":{"id":$id}}')
      if [ -n "$HASURA_ADMIN_SECRET" ]; then
        curl -s -X POST -H "Content-Type: application/json" -H "x-hasura-admin-secret: $HASURA_ADMIN_SECRET" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
      else
        curl -s -X POST -H "Content-Type: application/json" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
      fi
      exit 1
    fi
  else
    echo "No event-router container found and host POST failed ($http_code)." >&2
    # Cleanup Hasura config
    echo "Cleaning up Hasura config $HASURA_ID"
    DELETE_PAYLOAD=$(jq -n --arg id "$HASURA_ID" '{"query":"mutation DeleteConfig($id: uuid!){delete_event_configs_by_pk(id: $id){id}}","variables":{"id":$id}}')
    if [ -n "$HASURA_ADMIN_SECRET" ]; then
      curl -s -X POST -H "Content-Type: application/json" -H "x-hasura-admin-secret: $HASURA_ADMIN_SECRET" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
    else
      curl -s -X POST -H "Content-Type: application/json" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
    fi
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
    # Cleanup Hasura config
    if [ -n "$HASURA_ID" ]; then
      echo "Cleaning up Hasura config $HASURA_ID"
      DELETE_PAYLOAD=$(jq -n --arg id "$HASURA_ID" '{"query":"mutation DeleteConfig($id: uuid!){delete_event_configs_by_pk(id: $id){id}}","variables":{"id":$id}}')
      if [ -n "$HASURA_ADMIN_SECRET" ]; then
        curl -s -X POST -H "Content-Type: application/json" -H "x-hasura-admin-secret: $HASURA_ADMIN_SECRET" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
      else
        curl -s -X POST -H "Content-Type: application/json" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
      fi
    fi
    exit 0
  fi
  echo "Attempt $i: no message yet, retrying..."
  sleep 1
done

echo "Event Router smoke test FAILED: no message consumed from ${TOPIC}" >&2
# Cleanup Hasura config on failure
if [ -n "$HASURA_ID" ]; then
  echo "Cleaning up Hasura config $HASURA_ID"
  DELETE_PAYLOAD=$(jq -n --arg id "$HASURA_ID" '{"query":"mutation DeleteConfig($id: uuid!){delete_event_configs_by_pk(id: $id){id}}","variables":{"id":$id}}')
  if [ -n "$HASURA_ADMIN_SECRET" ]; then
    curl -s -X POST -H "Content-Type: application/json" -H "x-hasura-admin-secret: $HASURA_ADMIN_SECRET" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
  else
    curl -s -X POST -H "Content-Type: application/json" -d "$DELETE_PAYLOAD" "$HASURA_URL" >/dev/null || true
  fi
fi
exit 1
