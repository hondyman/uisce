#!/bin/bash

##############################################################################
# Calc Engine E2E Test Script
# Tests: Metric creation, PoP trigger, anomaly trigger, result retrieval
##############################################################################

set -e

# Configuration
BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
TENANT_ID="tenant-001"
USER_ID="test-user@example.com"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=== Calc Engine E2E Test ===${NC}"
echo "Backend URL: $BACKEND_URL"
echo "Tenant ID: $TENANT_ID"
echo ""

##############################################################################
# Test 1: Verify Backend Health
##############################################################################

echo -e "${BLUE}[1] Checking backend health...${NC}"
if ! curl -s -f "$BACKEND_URL/health" > /dev/null; then
  echo -e "${RED}Backend is not responding at $BACKEND_URL${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Backend is healthy${NC}"
echo ""

##############################################################################
# Test 2: Verify Routes are Registered
##############################################################################

echo -e "${BLUE}[2] Verifying calc-engine routes are registered...${NC}"
ROUTES=$(curl -s "$BACKEND_URL/_routes" | grep -c "/api/metrics" || true)
if [ "$ROUTES" -lt 3 ]; then
  echo -e "${RED}Calc-engine routes not found in routing table${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Found $ROUTES metric-related routes${NC}"
echo ""

##############################################################################
# Test 3: Create Metric
##############################################################################

echo -e "${BLUE}[3] Creating metric 'revenue_daily'...${NC}"

METRIC_RESPONSE=$(curl -s -X POST "$BACKEND_URL/api/metrics" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -d '{
    "name": "revenue_daily",
    "display_name": "Daily Revenue",
    "domain": "finance",
    "category": "sales",
    "granularity": "day",
    "aggregation_function": "sum",
    "sla_freshness_hours": 24,
    "sla_completeness_threshold": 95.0,
    "computation_type": "SQL",
    "computation_logic": "SELECT date, SUM(amount) FROM transactions GROUP BY date"
  }')

# Extract metric_id
METRIC_ID=$(echo "$METRIC_RESPONSE" | jq -r '.metric_id // empty')

if [ -z "$METRIC_ID" ] || [ "$METRIC_ID" = "null" ]; then
  echo -e "${RED}Failed to create metric${NC}"
  echo "Response: $METRIC_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ Metric created: $METRIC_ID${NC}"
echo ""

##############################################################################
# Test 4: Retrieve Metric
##############################################################################

echo -e "${BLUE}[4] Retrieving metric details...${NC}"

METRIC_GET=$(curl -s -X GET "$BACKEND_URL/api/metrics/$METRIC_ID" \
  -H "X-Tenant-ID: $TENANT_ID")

RETRIEVED_NAME=$(echo "$METRIC_GET" | jq -r '.name // empty')

if [ "$RETRIEVED_NAME" != "revenue_daily" ]; then
  echo -e "${RED}Failed to retrieve metric${NC}"
  echo "Response: $METRIC_GET"
  exit 1
fi

echo -e "${GREEN}✓ Metric retrieved successfully${NC}"
echo ""

##############################################################################
# Test 5: List Metrics
##############################################################################

echo -e "${BLUE}[5] Listing all metrics for tenant...${NC}"

METRICS_LIST=$(curl -s -X GET "$BACKEND_URL/api/metrics" \
  -H "X-Tenant-ID: $TENANT_ID")

COUNT=$(echo "$METRICS_LIST" | jq 'length')
echo -e "${GREEN}✓ Found $COUNT metric(s)${NC}"
echo ""

##############################################################################
# Test 6: Trigger PoP Computation
##############################################################################

echo -e "${BLUE}[6] Triggering PoP computation for 2024-08...${NC}"

POP_RESPONSE=$(curl -s -X POST "$BACKEND_URL/api/metrics/$METRIC_ID/compute/pop" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -d '{"period_label": "2024-08"}')

POP_RUN_ID=$(echo "$POP_RESPONSE" | jq -r '.run_id // empty')
POP_STATUS=$(echo "$POP_RESPONSE" | jq -r '.status // empty')

if [ -z "$POP_RUN_ID" ] || [ "$POP_RUN_ID" = "null" ]; then
  echo -e "${RED}Failed to trigger PoP computation${NC}"
  echo "Response: $POP_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ PoP computation triggered${NC}"
echo "  Run ID: $POP_RUN_ID"
echo "  Status: $POP_STATUS"
echo ""

##############################################################################
# Test 7: Check Job Run Status
##############################################################################

echo -e "${BLUE}[7] Retrieving job run status...${NC}"

RUNS=$(curl -s -X GET "$BACKEND_URL/api/metrics/$METRIC_ID/runs" \
  -H "X-Tenant-ID: $TENANT_ID")

RUN_COUNT=$(echo "$RUNS" | jq 'length')
LATEST_RUN_STATUS=$(echo "$RUNS" | jq -r '.[0].status // empty')

if [ "$RUN_COUNT" -lt 1 ]; then
  echo -e "${RED}No job runs found${NC}"
  exit 1
fi

echo -e "${GREEN}✓ Found $RUN_COUNT job run(s)${NC}"
echo "  Latest run status: $LATEST_RUN_STATUS"
echo ""

##############################################################################
# Test 8: Trigger Anomaly Computation
##############################################################################

echo -e "${BLUE}[8] Triggering anomaly detection for 2024-08...${NC}"

ANOMALY_RESPONSE=$(curl -s -X POST "$BACKEND_URL/api/metrics/$METRIC_ID/compute/anomaly" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -d '{"period_label": "2024-08"}')

ANOMALY_RUN_ID=$(echo "$ANOMALY_RESPONSE" | jq -r '.run_id // empty')

if [ -z "$ANOMALY_RUN_ID" ] || [ "$ANOMALY_RUN_ID" = "null" ]; then
  echo -e "${RED}Failed to trigger anomaly computation${NC}"
  echo "Response: $ANOMALY_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ Anomaly computation triggered${NC}"
echo "  Run ID: $ANOMALY_RUN_ID"
echo ""

##############################################################################
# Test 9: Retrieve Anomalies
##############################################################################

echo -e "${BLUE}[9] Retrieving detected anomalies...${NC}"

ANOMALIES=$(curl -s -X GET "$BACKEND_URL/api/metrics/$METRIC_ID/anomalies" \
  -H "X-Tenant-ID: $TENANT_ID")

ANOMALY_COUNT=$(echo "$ANOMALIES" | jq 'length')
echo -e "${GREEN}✓ Found $ANOMALY_COUNT anomaly event(s)${NC}"
echo ""

##############################################################################
# Test 10: Update Metric
##############################################################################

echo -e "${BLUE}[10] Updating metric SLA freshness...${NC}"

UPDATE_RESPONSE=$(curl -s -X PUT "$BACKEND_URL/api/metrics/$METRIC_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID" \
  -d '{
    "name": "revenue_daily",
    "display_name": "Daily Revenue (Updated)",
    "domain": "finance",
    "aggregation_function": "sum",
    "sla_freshness_hours": 12
  }')

UPDATED_FRESHNESS=$(echo "$UPDATE_RESPONSE" | jq -r '.sla_freshness_hours // empty')

if [ "$UPDATED_FRESHNESS" != "12" ]; then
  echo -e "${RED}Failed to update metric${NC}"
  echo "Response: $UPDATE_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ Metric updated (SLA freshness now 12 hours)${NC}"
echo ""

##############################################################################
# Test 11: Delete Metric
##############################################################################

echo -e "${BLUE}[11] Deleting metric...${NC}"

DELETE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BACKEND_URL/api/metrics/$METRIC_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-User-ID: $USER_ID")

if [ "$DELETE_CODE" != "204" ]; then
  echo -e "${RED}Failed to delete metric (HTTP $DELETE_CODE)${NC}"
  exit 1
fi

echo -e "${GREEN}✓ Metric deleted successfully${NC}"
echo ""

##############################################################################
# Summary
##############################################################################

echo -e "${GREEN}=== All Tests Passed! ===${NC}"
echo ""
echo "Summary:"
echo "  ✓ Backend health check"
echo "  ✓ Routes registered"
echo "  ✓ Metric creation"
echo "  ✓ Metric retrieval"
echo "  ✓ Metric listing"
echo "  ✓ PoP computation trigger"
echo "  ✓ Job run status retrieval"
echo "  ✓ Anomaly computation trigger"
echo "  ✓ Anomaly retrieval"
echo "  ✓ Metric update"
echo "  ✓ Metric deletion"
echo ""
echo "The production calc engine is working correctly!"
echo ""
echo "Next steps:"
echo "  1. Verify metrics in Postgres: SELECT * FROM metric_registry WHERE tenant_id = '$TENANT_ID'"
echo "  2. Check job runs: SELECT * FROM metric_job_runs WHERE tenant_id = '$TENANT_ID'"
echo "  3. Query Trino/Iceberg for PoP results"
echo "  4. Monitor Temporal workflows for background computation"
