#!/usr/bin/env bash
set -euo pipefail

# Usage:
# ASSERT_FEED_CARD_DELIVERY_BASEURL=https://api.example.com \
# TENANT=test_t1 CLIENT=test_c1 CARD_ID=card_dividend_income \
# ./scripts/assert_feed_card_delivery.sh

BASEURL="${ASSERT_FEED_CARD_DELIVERY_BASEURL:-http://localhost:8080}"
TENANT="${TENANT:?TENANT required}"
CLIENT="${CLIENT:?CLIENT required}"
CARD_ID="${CARD_ID:?CARD_ID required}"
API_KEY="${API_KEY:-test-api-key}"

echo "Checking feed for tenant=${TENANT} client=${CLIENT} expecting card=${CARD_ID}"

resp=$(curl -sS -H "X-Api-Key: ${API_KEY}" "${BASEURL}/v1/feed?tenant_id=${TENANT}&client_id=${CLIENT}&limit=20")
if echo "${resp}" | grep -q "\"card_id\":\"${CARD_ID}\""; then
  echo "✓ Card ${CARD_ID} FOUND in feed response"
else
  echo "✗ ERROR: Card ${CARD_ID} NOT found in feed response"
  echo "Response: ${resp}"
  exit 2
fi

# Find first instance of card_id and invoke default CTA if present
cta_id=$(echo "${resp}" | jq -r --arg CID "${CARD_ID}" '.cards[] | select(.card_id==$CID) | .ctas[0].id' | head -n1)

if [ -z "${cta_id}" ] || [ "${cta_id}" == "null" ]; then
  echo "No CTA present for ${CARD_ID}, done."
  exit 0
fi

echo "Invoking CTA ${cta_id} for card ${CARD_ID}"
cta_payload='{}'

cta_resp=$(curl -sS -H "X-Api-Key: ${API_KEY}" -H "Content-Type: application/json" \
  -d "{\"tenant_id\":\"${TENANT}\",\"client_id\":\"${CLIENT}\",\"cta_id\":\"${cta_id}\",\"payload\":${cta_payload}}" \
  "${BASEURL}/v1/feed/${CARD_ID}/cta")

echo "CTA response: ${cta_resp}"
if echo "${cta_resp}" | jq -e '.status == "accepted"' >/dev/null 2>&1; then
  echo "✓ CTA accepted"
  uar_hash=$(echo "${cta_resp}" | jq -r '.uar_hash')
  if [ -n "${uar_hash}" ] && [ "${uar_hash}" != "null" ]; then
    echo "✓ UAR hash returned: ${uar_hash}"
    exit 0
  else
    echo "✗ ERROR: no uar_hash in response"
    exit 3
  fi
else
  echo "✗ ERROR: CTA not accepted: ${cta_resp}"
  exit 4
fi
