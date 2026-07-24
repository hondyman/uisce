#!/bin/bash
# Load test script for compliance engine

BASE_URL="http://localhost:8090/api/v1/compliance/submit"
NUM_REQUESTS=10000
CONCURRENCY=100

echo "🚀 Load Testing Compliance Engine"
echo "   Requests: $NUM_REQUESTS"
echo "   Concurrency: $CONCURRENCY"
echo ""

# Create payload file
cat > /tmp/trade_payload.json <<EOF
{
  "id": "TXN-\$RANDOM",
  "tradeDate": "2025-12-29",
  "amount": 500000,
  "currency": "USD",
  "orderType": "LIMIT",
  "limitPrice": 150.0
}
EOF

# Function to send request
send_request() {
  local id="TXN-$RANDOM-$1"
  curl -s -X POST $BASE_URL \
    -H "Content-Type: application/json" \
    -d "{
      \"id\": \"$id\",
      \"tradeDate\": \"2025-12-29\",
      \"amount\": 500000,
      \"currency\": \"USD\",
      \"orderType\": \"LIMIT\",
      \"limitPrice\": 150.0
    }" > /dev/null
}

export -f send_request
export BASE_URL

echo "Starting load test..."
START_TIME=$(date +%s)

# Run concurrent requests
seq 1 $NUM_REQUESTS | xargs -P $CONCURRENCY -I {} bash -c 'send_request {}'

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
RPS=$((NUM_REQUESTS / DURATION))

echo ""
echo "✅ Load Test Complete"
echo "   Duration: ${DURATION}s"
echo "   Requests/sec: $RPS"
echo ""
echo "📊 Verify results:"
echo "   Redpanda Pandaproxy (HTTP): http://localhost:8082 (if host ports bound) - check topic(s) for compliance.post_trade"
echo "   Postgres: docker exec -it semlayer-postgres psql -U postgres -d alpha -c 'SELECT COUNT(*) FROM compliance_events;'"
echo "   StarRocks: docker exec -it starrocks-fe mysql -uroot -P9030 -h127.0.0.1 -e 'SELECT COUNT(*) FROM alpha.compliance_audit;'"
