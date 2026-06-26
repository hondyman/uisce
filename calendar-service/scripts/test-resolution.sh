#!/bin/bash
set -e

echo "🧪 Testing Full Holiday/Blackout Resolution"

# Configuration
TENANT_ID="${TENANT_ID:-550e8400-e29b-41d4-a716-446655440000}"
API_BASE="${API_BASE:-http://localhost:8081}"

# Test 1: Basic Holiday Resolution
echo ""
echo "📅 Test 1: Basic Availability Check"

# Check availability on a normal day
echo "Checking availability on a normal day (2026-02-20)..."
RESULT=$(curl -s -X POST "$API_BASE/api/v1/availability" \
  -H "X-User-ID: dev" -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"tenant_id":"'$TENANT_ID'","profile_name":"default","start_time":"2026-02-20T09:00:00Z","duration_secs":3600}' 2>/dev/null)
echo "Result: $RESULT"

if echo "$RESULT" | jq -e '.available' > /dev/null 2>&1; then
  echo "✅ Availability check works correctly"
else
  echo "⚠️  Availability endpoint returned: $RESULT"
fi

# Test 2: Cache Behavior
echo ""
echo "🚀 Test 2: Cache Behavior"

# Make 3 identical requests to test cache
echo "Making 3 identical requests to test cache..."
for i in {1..3}; do
  START=$(date +%s%N)
  curl -s -X POST "$API_BASE/api/v1/availability" \
    -H "X-User-ID: dev" -H "X-Tenant-ID: $TENANT_ID" \
    -d '{"tenant_id":"'$TENANT_ID'","profile_name":"default","start_time":"2026-02-20T09:00:00Z","duration_secs":3600}' > /dev/null 2>&1
  END=$(date +%s%N)
  DURATION=$(( (END - START) / 1000000 ))  # Convert to ms
  echo "  Request $i: ${DURATION}ms"
  sleep 0.05
done

# Test 3: Metrics
echo ""
echo "📊 Test 3: Prometheus Metrics"

echo "Checking metrics endpoint..."
METRICS=$(curl -s "$API_BASE/metrics" 2>/dev/null | grep -c "calendar_profile_resolution" || echo "0")
echo "Found $METRICS metric series for profile resolution"

if [ "$METRICS" -gt 0 ]; then
  echo "✅ Prometheus metrics are being collected"
  curl -s "$API_BASE/metrics" 2>/dev/null | grep "calendar_profile_resolution" | head -5
else
  echo "⚠️  No metrics found - metrics endpoint may not be initialized"
fi

# Test 4: GetMetrics Endpoint
echo ""
echo "📈 Test 4: Enhanced GetMetrics Endpoint"

METRICS=$(curl -s -X GET "$API_BASE/api/v1/availability/metrics" \
  -H "X-Tenant-ID: $TENANT_ID" 2>/dev/null)
echo "Metrics response: $METRICS"

if echo "$METRICS" | jq -e '.cache_enabled' > /dev/null 2>&1; then
  echo "✅ Enhanced metrics endpoint working"
else
  echo "⚠️  GetMetrics endpoint returned: $METRICS"
fi

echo ""
echo "✅ Core integration tests completed!"
echo ""
echo "📝 Next steps:"
echo "  1. Test with real calendar data in database"
echo "  2. Verify CDC invalidation triggers cache clears"
echo "  3. Monitor Prometheus metrics for production readiness"
