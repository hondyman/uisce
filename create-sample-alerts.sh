#!/bin/bash

# Create sample alerts for testing the Ops Cockpit

BASE_URL="http://localhost:8080"
AUTH_HEADER="Authorization: Bearer test-token"

echo "Creating sample alerts..."
echo ""

# Alert 1: High Error Rate
echo "1. Creating 'High Error Rate' alert..."
curl -s -X POST "$BASE_URL/admin/alerts" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Error Rate",
    "scope": "global",
    "metric": "error_rate",
    "threshold": 0.05,
    "comparison": ">",
    "window_secs": 300,
    "enabled": true
  }' | jq '.'
echo ""

# Alert 2: High P95 Latency
echo "2. Creating 'High P95 Latency' alert..."
curl -s -X POST "$BASE_URL/admin/alerts" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High P95 Latency",
    "scope": "global",
    "metric": "latency_p95",
    "threshold": 500,
    "comparison": ">",
    "window_secs": 300,
    "enabled": true
  }' | jq '.'
echo ""

# Alert 3: Low Availability
echo "3. Creating 'Low Availability' alert..."
curl -s -X POST "$BASE_URL/admin/alerts" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Low Availability",
    "scope": "global",
    "metric": "availability",
    "threshold": 95,
    "comparison": "<",
    "window_secs": 600,
    "enabled": true
  }' | jq '.'
echo ""

# Alert 4: High Rate Limit
echo "4. Creating 'High Rate Limit' alert..."
curl -s -X POST "$BASE_URL/admin/alerts" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Rate Limit Hits",
    "scope": "global",
    "metric": "rate_limited",
    "threshold": 100,
    "comparison": ">",
    "window_secs": 300,
    "enabled": true
  }' | jq '.'
echo ""

echo "✅ Sample alerts created!"
echo ""
echo "Next: Run tests with:"
echo "  bash test-ops-api.sh"
