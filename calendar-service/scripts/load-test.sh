#!/bin/bash
set -e

echo "🚀 Starting Load & RLS Stress Testing"
echo "======================================"

# Configuration
BACKEND_URL="${BACKEND_URL:-http://host.docker.internal:8081}"
HASURA_URL="${HASURA_URL:-http://host.docker.internal:8080}"
TEST_DURATION="${TEST_DURATION:-5m}"
VUS_MAX="${VUS_MAX:-100}"
TENANT_COUNT="${TENANT_COUNT:-10}"

echo "Backend URL: $BACKEND_URL"
echo "Hasura URL: $HASURA_URL"
echo "Test Duration: $TEST_DURATION"
echo "Max VUs: $VUS_MAX"
echo "Tenant Count: $TENANT_COUNT"

# Verify backend is accessible from Docker
echo ""
echo "=== Verifying Backend Connectivity ==="
docker run --rm --add-host=host.docker.internal:host-gateway \
  alpine/curl \
  curl -s -o /dev/null -w "%{http_code}" "$BACKEND_URL/health" || {
    echo "❌ Backend not accessible from Docker container"
    echo "Make sure backend is running and listening on 0.0.0.0"
    exit 1
  }
echo "✅ Backend accessible"

# Run k6 load tests
echo ""
echo "=== Running Load Tests ==="
docker run --rm --add-host=host.docker.internal:host-gateway \
  -v $(pwd)/k6:/scripts \
  -e BACKEND_URL="$BACKEND_URL" \
  -e HASURA_URL="$HASURA_URL" \
  -e TENANT_COUNT="$TENANT_COUNT" \
  grafana/k6:latest \
  run /scripts/api-load-test.js \
  --duration "$TEST_DURATION" \
  --vus-max "$VUS_MAX" \
  --out json=/scripts/results/load-test-$(date +%Y%m%d_%H%M%S).json

# Run RLS validation tests
echo ""
echo "=== Running RLS Validation Tests ==="
docker run --rm --add-host=host.docker.internal:host-gateway \
  -v $(pwd)/k6:/scripts \
  -e BACKEND_URL="$BACKEND_URL" \
  -e HASURA_URL="$HASURA_URL" \
  grafana/k6:latest \
  run /scripts/rls-validation-test.js \
  --out json=/scripts/results/rls-test-$(date +%Y%m%d_%H%M%S).json

# Generate report
echo ""
echo "=== Generating Test Report ==="
./scripts/generate-test-report.sh

echo ""
echo "✅ Load testing complete!"
echo "📊 Results saved to: k6/results/"
