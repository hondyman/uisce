#!/usr/bin/env bash
set -euo pipefail

HASURA_URL=${HASURA_URL:-http://localhost:8080/v1/graphql}
ADMIN_SECRET=${HASURA_ADMIN_SECRET:-secret}

echo "Waiting for Hasura to be ready at ${HASURA_URL}..."
for i in {1..30}; do
  status=$(curl -s -o /dev/null -w "%{http_code}" ${HASURA_URL}/healthz || true)
  if [ "$status" = "200" ]; then
    echo "Hasura is up"
    break
  fi
  sleep 1
done

echo "Running dynamic_insert action test (via Hasura)..."
read -r -d '' PAYLOAD <<'GRAPHQL'
mutation DynamicInsert($entity_type: String!, $object: jsonb!) {
  dynamic_insert(entity_type: $entity_type, object: $object) {
    success
    result
    error
  }
}
GRAPHQL

curl -s -X POST ${HASURA_URL} \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Admin-Secret: ${ADMIN_SECRET}" \
  -d '{"query":"mutation DynamicInsert($entity_type: String!, $object: jsonb!) { dynamic_insert(entity_type: $entity_type, object: $object) { success result error } }","variables":{"entity_type":"client_investors","object":{"type":"ClientInvestor","name":"Smoke Test","custom_fields":{}}}}' | jq .

echo "Smoke test completed."
