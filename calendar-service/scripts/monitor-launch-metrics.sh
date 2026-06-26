#!/bin/bash

echo "📊 Launch Day Performance Monitor"
echo "=================================="
echo "Time: $(date)"
echo ""

# API Metrics
echo "=== API Performance ==="
P95_LATENCY=$(curl -s https://prometheus.yourcompany.com/api/v1/query \
  -d "query=histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{environment=\"production\"}[5m]))" \
  | jq -r '.data.result[0].value[1]')

ERROR_RATE=$(curl -s https://prometheus.yourcompany.com/api/v1/query \
  -d "query=rate(http_requests_total{status=~\"5..\", environment=\"production\"}[5m]) / rate(http_requests_total{environment=\"production\"}[5m])" \
  | jq -r '.data.result[0].value[1]')

REQUEST_RATE=$(curl -s https://prometheus.yourcompany.com/api/v1/query \
  -d "query=rate(http_requests_total{environment=\"production\"}[5m])" \
  | jq -r '.data.result[0].value[1]')

echo "p95 Latency:    ${P95_LATENCY}s (target: <0.5s)"
echo "Error Rate:     ${ERROR_RATE} (target: <0.01)"
echo "Request Rate:   ${REQUEST_RATE}/sec"

# Check thresholds
echo ""
echo "=== Threshold Checks ==="
if (( $(echo "$P95_LATENCY > 0.5" | bc -l) )); then
    echo "⚠️  WARNING: p95 latency above 500ms"
else
    echo "✅ p95 latency OK"
fi

if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
    echo "⚠️  WARNING: Error rate above 1%"
else
    echo "✅ Error rate OK"
fi

# Database Metrics
echo ""
echo "=== Database Performance ==="
psql $PROD_DATABASE_URL -c "
SELECT 
    COUNT(*) as active_connections,
    COUNT(*) * 100.0 / (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as connection_utilization
FROM pg_stat_activity 
WHERE state = 'active';"

# Cache Metrics
echo ""
echo "=== Cache Performance ==="
docker exec -it calendar-redis-dev redis-cli INFO stats | grep -E "keyspace_hits|keyspace_misses"

# Sync Metrics
echo ""
echo "=== Sync Performance ==="
SYNC_SUCCESS=$(curl -s https://prometheus.yourcompany.com/api/v1/query \
  -d "query=rate(sync_jobs_total{status=\"completed\", environment=\"production\"}[5m]) / rate(sync_jobs_total{environment=\"production\"}[5m])" \
  | jq -r '.data.result[0].value[1]')

echo "Sync Success Rate: ${SYNC_SUCCESS} (target: >0.95)"

echo ""
echo "Last updated: $(date)"
echo "Next check: 5 minutes"
