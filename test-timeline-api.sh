#!/bin/bash
# Timeline API Test Script
# This script demonstrates how to interact with the Incident Timeline API
# 
# Prerequisites:
#   - Server running on localhost:8080
#   - jq installed for JSON formatting (optional)

BASE_URL="http://localhost:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== SemLayer Incident Timeline API Tests ===${NC}\n"

# Test 1: Get Recent Timeline (Last 1 hour)
echo -e "${YELLOW}Test 1: Get timeline for last 1 hour${NC}"
echo "GET /admin/ops/timeline?since=1h&limit=100"
curl -s "${BASE_URL}/admin/ops/timeline?since=1h&limit=100" | jq '.' | head -50
echo -e "\n"

# Test 2: Get Recent Timeline (Last 24 hours) with fewer results
echo -e "${YELLOW}Test 2: Get timeline for last 24 hours (limit 50)${NC}"
echo "GET /admin/ops/timeline?since=24h&limit=50"
curl -s "${BASE_URL}/admin/ops/timeline?since=24h&limit=50" | jq '.total'
echo -e "\n"

# Test 3: Filter for only critical events
echo -e "${YELLOW}Test 3: Get critical events (frontend filtering)${NC}"
echo "GET /admin/ops/timeline?since=24h&limit=100"
curl -s "${BASE_URL}/admin/ops/timeline?since=24h&limit=100" | jq '.events[] | select(.severity=="critical")'
echo -e "\n"

# Test 4: Get first incident from timeline (if exists)
echo -e "${YELLOW}Test 4: Get first incident details${NC}"
FIRST_INCIDENT=$(curl -s "${BASE_URL}/admin/ops/timeline?since=24h&limit=1" | jq -r '.events[0]?.incident_id // empty')

if [ -z "$FIRST_INCIDENT" ]; then
    echo -e "${RED}No incidents found in recent timeline${NC}"
    echo "You may need to:"
    echo "  1. Generate some test events first"
    echo "  2. Increase the time range (e.g., ?since=7d)"
else
    echo "GET /admin/ops/incidents/${FIRST_INCIDENT}"
    curl -s "${BASE_URL}/admin/ops/incidents/${FIRST_INCIDENT}" | jq '.'
fi
echo -e "\n"

# Test 5: Count events by severity
echo -e "${YELLOW}Test 5: Count events by severity${NC}"
echo "GET /admin/ops/timeline?since=7d&limit=1000"
echo "Event counts by severity:"
curl -s "${BASE_URL}/admin/ops/timeline?since=7d&limit=1000" | jq '[.events[] | .severity] | group_by(.) | map({severity: .[0], count: length}) | sort_by(.count) | reverse[]'
echo -e "\n"

# Test 6: Count events by type
echo -e "${YELLOW}Test 6: Count events by type${NC}"
echo "Event counts by type:"
curl -s "${BASE_URL}/admin/ops/timeline?since=7d&limit=1000" | jq '[.events[] | .event_type] | group_by(.) | map({type: .[0], count: length}) | sort_by(.count) | reverse[]'
echo -e "\n"

# Test 7: Find all open incidents
echo -e "${YELLOW}Test 7: List of events with open incidents${NC}"
echo "GET /admin/ops/timeline?since=24h&limit=100"
curl -s "${BASE_URL}/admin/ops/timeline?since=24h&limit=100" | jq '.events[] | select(.incident_id != null) | {incident_id, event_type, severity, title}' | head -30
echo -e "\n"

# Test 8: Test close incident endpoint (requires valid incident ID)
if [ -n "$FIRST_INCIDENT" ]; then
    echo -e "${YELLOW}Test 8: Close incident (example)${NC}"
    echo "POST /admin/ops/incidents/${FIRST_INCIDENT}/close"
    echo "Request body:"
    cat << EOF
{
  "summary": "Root cause identified and fixed",
  "root_cause": "Database connection pool exhaustion due to tenant configuration"
}
EOF
    echo ""
    echo "To actually execute this, uncomment the curl command below:"
    echo "# curl -X POST \"${BASE_URL}/admin/ops/incidents/${FIRST_INCIDENT}/close\" \\"
    echo "#   -H \"Content-Type: application/json\" \\"
    echo "#   -d '{\"summary\":\"Root cause identified\",\"root_cause\":\"Connection pool issue\"}'"
    echo ""
fi

echo -e "${GREEN}=== Tests Complete ===${NC}\n"

# Summary of endpoints
echo -e "${YELLOW}=== API Endpoint Summary ===${NC}"
echo "GET  /admin/ops/timeline?since=1h&limit=200"
echo "     Query recent events by time range and severity"
echo "     Parameters:"
echo "       - since: Duration string (e.g., '1h', '24h', '7d')"
echo "       - limit: Max results (1-1000)"
echo ""
echo "GET  /admin/ops/incidents/{incidentID}"
echo "     Get incident details with all related events"
echo "     Parameters:"
echo "       - incidentID: UUID of incident"
echo ""
echo "POST /admin/ops/incidents/{incidentID}/close"
echo "     Close an incident with optional analysis"
echo "     Request body:"
echo "       {"
echo "         \"summary\": \"What was the issue\","
echo "         \"root_cause\": \"Why did it happen\""
echo "       }"
echo ""
