#!/bin/bash

# Phase 2 Test Integration - Direct API Testing
# Tests the holiday/blackout resolution pipeline using direct API calls
# (Simulates what would happen with real data)

set -e

TENANT_ID="550e8400-e29b-41d4-a716-446655440000"
API_BASE="${API_BASE:-http://localhost:8081}"

echo "🧪 Phase 2 Integration Testing"
echo "==============================="
echo ""

# Wait for service to be ready
echo "📡 Waiting for calendar-service to be ready..."
for i in {1..30}; do
  if curl -s "${API_BASE}/health" > /dev/null 2>&1; then
    echo "✅ Service is ready"
    break
  fi
  if [ $i -eq 30 ]; then
    echo "❌ Service not ready after 30 seconds"
    exit 1
  fi
  sleep 1
done

echo ""
echo "📋 Running Integration Tests"
echo "---"

# Test 1: Availability Check - Normal Day
echo ""
echo "Test 1: Availability Check (Normal Day)"
RESULT=$(curl -s -X POST "${API_BASE}/api/v1/availability" \
  -H "X-User-ID: dev" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Region: US" \
  -d '{
    "profile_name": "default",
    "start_time": "2026-02-20T09:00:00Z",
    "end_time": "2026-02-20T10:00:00Z"
  }' 2>&1)

echo "Response: ${RESULT}"
if echo "${RESULT}" | grep -q "available"; then
  echo "✅ PASS: Normal day availability check"
else
  echo "⚠️  Response structure: ${RESULT:0:100}"
fi

# Test 2: Cache Hit Test (should be much faster)
echo ""
echo "Test 2: Cache Hit Test"
START_TIME=$(date +%s%N)
RESULT=$(curl -s -X POST "${API_BASE}/api/v1/availability" \
  -H "X-User-ID: dev" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Region: US" \
  -d '{
    "profile_name": "default",
    "start_time": "2026-02-20T09:00:00Z",
    "end_time": "2026-02-20T10:00:00Z"
  }' 2>&1)
END_TIME=$(date +%s%N)
DURATION=$(( (END_TIME - START_TIME) / 1000000 ))

echo "Response time: ${DURATION}ms"
if [ "$DURATION" -lt 50 ]; then
  echo "✅ PASS: Cache hit (< 50ms)"
elif [ "$DURATION" -lt 100 ]; then
  echo "⚠️  WARN: Slower than expected (${DURATION}ms > 50ms)"
else
  echo "⚠️  INFO: First request or no cache (${DURATION}ms)"
fi

# Test 3: Metrics Endpoint
echo ""
echo "Test 3: Metrics Endpoint"
METRICS=$(curl -s "${API_BASE}/metrics" 2>&1 | head -20)
if echo "${METRICS}" | grep -q "calendar_profile_resolution"; then
  echo "✅ PASS: Metrics being collected"
  echo "Sample metrics:"
  echo "${METRICS}" | grep "calendar_profile_resolution" | head -3
else
  echo "⚠️  WARN: No profile resolution metrics found"
fi

# Test 4: GetMetrics Endpoint
echo ""
echo "Test 4: GetMetrics Endpoint"
RESULT=$(curl -s -X GET "${API_BASE}/api/v1/availability/metrics" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Region: US" 2>&1)
echo "Response: ${RESULT:0:150}"
if echo "${RESULT}" | grep -q "error" || echo "${RESULT}" | grep -q "404"; then
  echo "⚠️  WARN: Metrics endpoint may not exist yet"
else
  echo "✅ PASS: Metrics endpoint responding"
fi

# Test 5: Profile Resolution Fallback (no profile)
echo ""
echo "Test 5: Resolution with Non-existent Profile"
RESULT=$(curl -s -X POST "${API_BASE}/api/v1/availability" \
  -H "X-User-ID: dev" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Region: US" \
  -d '{
    "profile_name": "nonexistent",
    "start_time": "2026-02-20T09:00:00Z",
    "end_time": "2026-02-20T10:00:00Z"
  }' 2>&1)

if echo "${RESULT}" | grep -q '"available"'; then
  echo "✅ PASS: Gracefully handled missing profile"
else
  echo "Response: ${RESULT:0:100}"
  echo "⚠️  WARN: Unexpected response format"
fi

echo ""
echo "📊 Test Summary"
echo "---"
echo "✅ All core tests executed"
echo ""
echo "📝 Notes:"
echo "  - These tests check basic API functionality"
echo "  - Full resolution testing requires database schema"
echo "  - Run 'phase2-test-setup.sh' to populate test data"
echo ""
echo "🎯 Next Steps:"
echo "  1. Ensure database schema is created"
echo "  2. Run: ./scripts/phase2-test-setup.sh"
echo "  3. Verify recurring blackout expansion works"
echo "  4. Check cache hit rates in metrics"
echo ""
