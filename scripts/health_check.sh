#!/usr/bin/env bash
set -euo pipefail

# Simple health check for local dev stack
# Checks: api-gateway (8000), sample GraphQL via gateway

API_GATEWAY_URL="http://localhost:8000/health"
GATEWAY_GRAPHQL="http://localhost:8000/api/graphql"

echo "Checking api-gateway: $API_GATEWAY_URL"
if curl --fail -sS "$API_GATEWAY_URL" >/dev/null; then
  echo "api-gateway: OK"
else
  echo "api-gateway: FAIL" >&2
  exit 2
fi

# Run a lightweight GraphQL introspection via gateway
echo "Running sample GraphQL query via gateway: $GATEWAY_GRAPHQL"
read -r resp <<EOF
$(curl -sS -H "Content-Type: application/json" -X POST "$GATEWAY_GRAPHQL" -d '{"query":"{ __typename }"}')
EOF

if echo "$resp" | grep -q "__typename"; then
  echo "graphQL via gateway: OK"
else
  echo "graphQL via gateway: FAIL" >&2
  echo "Response: $resp" >&2
  exit 4
fi

echo "All health checks passed."
