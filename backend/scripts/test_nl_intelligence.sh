#!/bin/bash

# NL Intelligence Smoke Test Script
# This script tests the core endpoints of the NL Intelligence Layer.

API_BASE="http://localhost:8080/api"
TENANT_ID="default"

echo "=== NL Intelligence Smoke Test ==="

# 1. Test Interpretation (Intent Classification)
echo -n "Testing Interpretation... "
INTERPRET_RESP=$(curl -s -X POST "$API_BASE/nl/interpret" \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What are the top 5 tables by row count?",
    "tenant_scope": ["default"]
  }')

INTENT=$(echo $INTERPRET_RESP | grep -o '"intent":"[^"]*' | cut -d'"' -f4)
if [ "$INTENT" == "COLLECT_METRICS" ] || [ "$INTENT" == "DATA_QUERY" ]; then
    echo "SUCCESS (Intent: $INTENT)"
else
    echo "FAILED (Resp: $INTERPRET_RESP)"
fi

# 2. Test AI-Driven API Design
echo -n "Testing AI API Design... "
API_DESIGN_RESP=$(curl -s -X POST "$API_BASE/api-studio/endpoints/ai" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Create an endpoint to fetch active users for tenant acme",
    "tenant_id": "default"
  }')

ENDPOINT_NAME=$(echo $API_DESIGN_RESP | grep -o '"name":"[^"]*' | cut -d'"' -f4)
if [ -n "$ENDPOINT_NAME" ]; then
    echo "SUCCESS (Endpoint: $ENDPOINT_NAME)"
else
    echo "FAILED (Resp: $API_DESIGN_RESP)"
fi

# 3. Test Incident Explanation
echo -n "Testing Incident Explanation... "
EXPLAIN_RESP=$(curl -s -X POST "$API_BASE/nl/explain-incident" \
  -H "Content-Type: application/json" \
  -d '{
    "incident_id": "INC-123",
    "graph_context": {}
  }')

NARRATIVE=$(echo $EXPLAIN_RESP | grep -o '"narrative":"[^"]*' | cut -d'"' -f4)
if [ -n "$NARRATIVE" ]; then
    echo "SUCCESS (Narrative exists)"
else
    echo "FAILED (Resp: $EXPLAIN_RESP)"
fi

echo "=== Smoke Test Complete ==="
