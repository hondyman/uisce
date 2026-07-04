#!/bin/bash

# Test script for Global Ops Cockpit API endpoints

BASE_URL="http://localhost:8080"
AUTH_HEADER="Authorization: Bearer test-token-$(uuidgen)"

echo "Testing Global Ops Cockpit endpoints..."
echo ""

# Test 1: List alerts
echo "1. Testing GET /admin/alerts"
curl -s -X GET "$BASE_URL/admin/alerts" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" | jq '.' || echo "Request failed"
echo ""

# Test 2: Get eval alerts  
echo "2. Testing POST /admin/alerts/evaluate"
curl -s -X POST "$BASE_URL/admin/alerts/evaluate" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" | jq '.' || echo "Request failed"
echo ""

# Test 3: Get tenant health
echo "3. Testing GET /admin/tenants/{tenantID}/health"
TENANT_ID=$(uuidgen)
curl -s -X GET "$BASE_URL/admin/tenants/$TENANT_ID/health" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" | jq '.' || echo "Request failed"
echo ""

# Test 4: List endpoint health
echo "4. Testing GET /admin/endpoints/health"
curl -s -X GET "$BASE_URL/admin/endpoints/health" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" | jq '.' || echo "Request failed"
echo ""

# Test 5: Get latency heatmap
echo "5. Testing GET /admin/latency/heatmap"
curl -s -X GET "$BASE_URL/admin/latency/heatmap?window=3600" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" | jq '.' || echo "Request failed"
echo ""

# Test 6: Get error fingerprints  
echo "6. Testing GET /admin/errors/fingerprints"
curl -s -X GET "$BASE_URL/admin/errors/fingerprints?limit=20" \
  -H "$AUTH_HEADER" \
  -H "Content-Type: application/json" | jq '.' || echo "Request failed"
echo ""

echo "✅ All endpoint tests complete!"
