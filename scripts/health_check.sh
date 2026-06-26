#!/usr/bin/env bash
set -euo pipefail

# Simple health check for local dev stack
# Checks: api-gateway (8000), hasura (8081), sample GraphQL via gateway

API_GATEWAY_URL="http://localhost:8000/health"
# Hasura may be exposed on 8081 (host) or 8080 depending on compose mapping; try 8081 first
HASURA_URL_PRIMARY="http://localhost:8081/healthz"
HASURA_URL_FALLBACK="http://localhost:8080/healthz"
GATEWAY_GRAPHQL="http://localhost:8000/api/graphql"
# Read admin secret from environment, fallback to repo default
ADMIN_SECRET="${HASURA_ADMIN_SECRET:-newadminsecretkey}"

echo "Checking api-gateway: $API_GATEWAY_URL"
if curl --fail -sS "$API_GATEWAY_URL" >/dev/null; then
  echo "api-gateway: OK"
else
  echo "api-gateway: FAIL" >&2
  exit 2
fi

echo "Checking hasura: trying $HASURA_URL_PRIMARY then $HASURA_URL_FALLBACK"
if curl --fail -sS -H "X-Hasura-Admin-Secret: $ADMIN_SECRET" "$HASURA_URL_PRIMARY" >/dev/null; then
  echo "hasura: OK ($HASURA_URL_PRIMARY)"
elif curl --fail -sS -H "X-Hasura-Admin-Secret: $ADMIN_SECRET" "$HASURA_URL_FALLBACK" >/dev/null; then
  echo "hasura: OK ($HASURA_URL_FALLBACK)"
else
  echo "hasura: FAIL" >&2
  exit 3
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
