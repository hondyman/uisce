#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/e2e_run.sh --scenario basic-mvp
BASEURL="${E2E_BASEURL:-http://localhost:8080}"
API_KEY="${API_KEY:-test-api-key}"

function usage() {
  echo "e2e_run.sh --scenario basic-mvp"
  exit 1
}

if [ "${1:-}" != "--scenario" ]; then
  usage
fi
SCENARIO="${2:-}"
if [ -z "${SCENARIO}" ]; then usage; fi

echo "Running E2E scenario: ${SCENARIO}"

if [ "${SCENARIO}" = "basic-mvp" ]; then
  TENANT="test_t1"
  CLIENT="test_c1"
  
  # 1) create tenant, client, seed holdings and upcoming dividend via test fixtures
  echo "Seeding test tenant/client/holdings..."
  curl -sS -H "X-Api-Key: ${API_KEY}" -H "Content-Type: application/json" \
    -X POST "${BASEURL}/internal/test/seed" \
    -d "{\"tenant_id\":\"${TENANT}\",\"client_id\":\"${CLIENT}\",\"fixtures\":[{\"type\":\"holdings\",\"symbol\":\"MSFT\",\"shares\":100},{\"type\":\"dividend_event\",\"symbol\":\"MSFT\",\"date\":\"$(date -d '+3 days' --iso-8601=seconds 2>/dev/null || date -v+3d -Iseconds)\"}]}" \
    > /tmp/seed_resp.json
  echo "✓ Seed response: $(cat /tmp/seed_resp.json)"

  # 2) call feed and assert card present
  echo "Calling feed..."
  feed_resp=$(curl -sS -H "X-Api-Key: ${API_KEY}" "${BASEURL}/v1/feed?tenant_id=${TENANT}&client_id=${CLIENT}&limit=20")
  echo "Feed response: ${feed_resp}"
  if ! echo "${feed_resp}" | grep -q '"card_id":"card_dividend_income"'; then
    echo "✗ FAIL: expected card_dividend_income in feed"
    exit 2
  fi
  echo "✓ Card found in feed"

  # 3) invoke CTA reinvest
  cta_id=$(echo "${feed_resp}" | jq -r '.cards[] | select(.card_id=="card_dividend_income") | .ctas[0].id')
  echo "Invoking CTA ${cta_id}"
  cta_resp=$(curl -sS -H "X-Api-Key: ${API_KEY}" -H "Content-Type: application/json" \
    -X POST "${BASEURL}/v1/feed/card_dividend_income/cta" \
    -d "{\"tenant_id\":\"${TENANT}\",\"client_id\":\"${CLIENT}\",\"cta_id\":\"${cta_id}\",\"payload\":{}}")
  echo "CTA resp: ${cta_resp}"

  # 4) verify UAR chain head exists and is valid via verify endpoint
  uar_hash=$(echo "${cta_resp}" | jq -r '.uar_hash')
  if [ -z "${uar_hash}" ] || [ "${uar_hash}" = "null" ]; then
    echo "✗ FAIL: no uar_hash"
    exit 3
  fi
  echo "✓ UAR hash from CTA: ${uar_hash}"

  verify_resp=$(curl -sS -H "X-Api-Key: ${API_KEY}" -H "Content-Type: application/json" \
    -X POST "${BASEURL}/v1/uar/verify" \
    -d "{\"tenant_id\":\"${TENANT}\"}")
  echo "Verify response: ${verify_resp}"
  if ! echo "${verify_resp}" | jq -e '.verified == true' >/dev/null 2>&1; then
    echo "✗ FAIL: UAR verification failed"
    exit 4
  fi
  echo "✓ UAR chain verified"

  echo ""
  echo "=========================================="
  echo "✓ E2E basic-mvp scenario PASSED"
  echo "=========================================="
  exit 0
else
  echo "✗ Unknown scenario: ${SCENARIO}"
  exit 1
fi
