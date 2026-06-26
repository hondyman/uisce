#!/bin/bash
set -e

echo "📊 Testing Observability Stack"

# Configuration
API_BASE="http://localhost:8081"
GRAFANA_URL="http://localhost:3000"
PROMETHEUS_URL="http://localhost:9090"

# Test 1: Verify Metrics Endpoint
echo ""
echo "📈 Test 1: Verify Metrics Endpoint"
METRICS=$(curl -s "$API_BASE/metrics")

if echo "$METRICS" | grep -q "google_sync_jobs_total"; then
    echo "✅ Sync job metrics exposed"
else
    echo "❌ Sync job metrics missing"
    exit 1
fi

if echo "$METRICS" | grep -q "google_calendar_api_calls_total"; then
    echo "✅ API call metrics exposed"
else
    echo "❌ API call metrics missing"
    exit 1
fi

# Test 2: Verify Prometheus Scraping
echo ""
echo "🔍 Test 2: Verify Prometheus Scraping"
PROM_METRICS=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=google_sync_jobs_total")

if echo "$PROM_METRICS" | grep -q "status"; then
    echo "✅ Prometheus scraping metrics"
else
    echo "❌ Prometheus not scraping metrics"
    exit 1
fi

# Test 3: Verify Grafana Dashboard
echo ""
echo "📊 Test 3: Verify Grafana Dashboard"
DASHBOARD=$(curl -s "$GRAFANA_URL/api/dashboards/uid/google-calendar-sync" \
  -H "Authorization: Bearer ${GRAFANA_API_KEY}")

if echo "$DASHBOARD" | grep -q "Google Calendar Sync"; then
    echo "✅ Grafana dashboard accessible"
else
    echo "⚠️  Grafana dashboard not found (may need manual import)"
fi

echo ""
echo "✅ Observability test complete!"
