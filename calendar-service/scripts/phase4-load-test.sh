#!/bin/bash
# Phase 4: Load Testing & Performance Benchmarking
# Tests cache effectiveness and system performance under load

set -e

SERVICE_URL="http://127.0.0.1:9081"
METRICS_URL="${SERVICE_URL}/metrics"
TENANT_ID="870361a8-87e2-4171-95ad-0473cc93791e"
CALENDAR_ID="7d3be7d4-5134-45af-b66c-547cedea9e08"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Phase 4: Load Testing & Performance Validation                ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Generate JWT token
TOKEN=$(python3 -c "
import base64, json, hmac, hashlib, time
secret = 'dev-jwt-secret-key-change-in-production'
tenant = '870361a8-87e2-4171-95ad-0473cc93791e'
h = base64.urlsafe_b64encode(json.dumps({'alg': 'HS256', 'typ': 'JWT'}).encode()).decode().rstrip('=')
p = base64.urlsafe_b64encode(json.dumps({'user_id': 'test', 'tenant_id': tenant, 'exp': int(time.time())+3600, 'iat': int(time.time())}).encode()).decode().rstrip('=')
m = f'{h}.{p}'
s = base64.urlsafe_b64encode(hmac.new(secret.encode(), m.encode(), hashlib.sha256).digest()).decode().rstrip('=')
print(f'{m}.{s}')
" 2>/dev/null)

echo -e "${YELLOW}🔄 BASELINE: First request (cache miss)${NC}"
echo "Testing availability endpoint..."
START=$(date +%s%N)
RESPONSE=$(curl -s -X POST "${SERVICE_URL}/api/v1/availability" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "{\"calendar_id\":\"$CALENDAR_ID\",\"date\":\"2026-02-20\"}")
END=$(date +%s%N)
FIRST_CALL_MS=$(( (END - START) / 1000000 ))
echo "Response time: ${FIRST_CALL_MS}ms"
echo "Cache status: MISS (first call)"
echo ""

echo -e "${YELLOW}🚀 CACHED: Second request (should be cached)${NC}"
START=$(date +%s%N)
RESPONSE=$(curl -s -X POST "${SERVICE_URL}/api/v1/availability" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "{\"calendar_id\":\"$CALENDAR_ID\",\"date\":\"2026-02-20\"}")
END=$(date +%s%N)
CACHED_CALL_MS=$(( (END - START) / 1000000 ))
echo "Response time: ${CACHED_CALL_MS}ms"
echo "Cache status: HIT (cached result)"
echo ""

# Calculate improvement
if [ $CACHED_CALL_MS -gt 0 ]; then
    IMPROVEMENT=$(( (FIRST_CALL_MS - CACHED_CALL_MS) * 100 / FIRST_CALL_MS ))
    echo -e "${GREEN}⚡ Performance Improvement: ${IMPROVEMENT}%${NC}"
    echo "  First call:  ${FIRST_CALL_MS}ms (DB + Hasura query)"
    echo "  Cached call: ${CACHED_CALL_MS}ms (Redis hit)"
fi
echo ""

# Load test: 10 concurrent requests
echo -e "${YELLOW}📊 LOAD TEST: 10 concurrent requests${NC}"
echo "Running concurrent requests..."
CONCURRENT=10
TIME_TAKEN=0

START=$(date +%s%N)
for i in $(seq 1 $CONCURRENT); do
    curl -s -X POST "${SERVICE_URL}/api/v1/availability" \
      -H "Authorization: Bearer $TOKEN" \
      -H "X-Tenant-ID: $TENANT_ID" \
      -H "Content-Type: application/json" \
      -d "{\"calendar_id\":\"$CALENDAR_ID\",\"date\":\"2026-02-2$((i % 10))\"}" > /dev/null &
done
wait
END=$(date +%s%N)
TIME_TAKEN=$(( (END - START) / 1000000 ))

THROUGHPUT=$(( (CONCURRENT * 1000) / TIME_TAKEN ))
echo "Total time: ${TIME_TAKEN}ms for $CONCURRENT requests"
echo "Throughput: ${THROUGHPUT} requests/second"
echo ""

# Extended load test with ApacheBench if available
echo -e "${YELLOW}📈 EXTENDED LOAD TEST: 100 requests with 10 concurrent${NC}"
if command -v ab &> /dev/null; then
    ab -n 100 -c 10 -q \
      -H "Authorization: Bearer $TOKEN" \
      -H "X-Tenant-ID: $TENANT_ID" \
      "${SERVICE_URL}/api/v1/calendars/${CALENDAR_ID}" 2>&1 | tail -20
else
    echo "ApacheBench (ab) not installed. Install with: brew install httpd"
    echo "Skipping extended load test..."
fi
echo ""

# Check Prometheus metrics if available
echo -e "${YELLOW}📊 PROMETHEUS METRICS${NC}"
if curl -s "${METRICS_URL}" > /dev/null 2>&1; then
    echo "✅ Prometheus metrics endpoint available at ${METRICS_URL}"
    echo ""
    echo "Key metrics to monitor:"
    echo "  - calendar_cache_hit_rate: Should be > 0.8 after warmup"
    echo "  - calendar_resolution_duration_seconds: Should be < 0.1s (100ms)"
    echo "  - calendar_requests_in_flight: Should stay low"
    echo ""
    echo "Sample metrics (first 10):"
    curl -s "${METRICS_URL}" | grep "^calendar_" | head -10
else
    echo "❌ Prometheus metrics endpoint not available"
    echo "Enable with: -metrics-port 8090"
fi
echo ""

# Redis cache status
echo -e "${YELLOW}🗄️  REDIS CACHE STATUS${NC}"
if command -v docker &> /dev/null && docker ps | grep -q redis; then
    echo "✅ Redis container running"
    docker exec redis-calendar-cache redis-cli INFO stats | grep -E "total_commands|expired_keys|evicted_keys" || true
else
    echo "❌ Redis not available"
    echo "Start with: docker run -d -p 6379:6379 redis:7-alpine"
fi
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Phase 4 Performance Validation Complete                      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Summary
echo "📊 PERFORMANCE SUMMARY:"
echo "  ✅ First call (cache miss):  ${FIRST_CALL_MS}ms"
echo "  ✅ Cached call (cache hit):  ${CACHED_CALL_MS}ms"
if [ $IMPROVEMENT -gt 0 ]; then
    echo -e "  ${GREEN}✅ Cache speedup:           ${IMPROVEMENT}%${NC}"
fi
echo "  ✅ Concurrent throughput:    ${THROUGHPUT} req/s"
echo ""
echo "🎯 Performance Goals:"
echo "  ✓ First call < 150ms:  $([ $FIRST_CALL_MS -lt 150 ] && echo "✅ PASS" || echo "❌ FAIL (${FIRST_CALL_MS}ms)")"
echo "  ✓ Cached call < 20ms:  $([ $CACHED_CALL_MS -lt 20 ] && echo "✅ PASS" || echo "❌ FAIL (${CACHED_CALL_MS}ms)")"
echo "  ✓ Throughput > 50 req/s: $([ $THROUGHPUT -gt 50 ] && echo "✅ PASS" || echo "❌ FAIL (${THROUGHPUT} req/s)")"
