#!/bin/bash
# Comprehensive Integration Test Suite
# Tests all major platform features end-to-end

set -e

BASE_URL="${API_URL:-http://localhost:8080}"
CLIENT_ID="${TEST_CLIENT_ID:-test-client-123}"
TENANT_ID="${TEST_TENANT_ID:-test-tenant-456}"

echo "==================================="
echo "Platform Integration Test Suite"
echo "==================================="
echo "Testing against: $BASE_URL"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

passed=0
failed=0

test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    
    echo -n "Testing $name... "
    
    if [ "$method" == "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" == "200" ] || [ "$http_code" == "201" ]; then
        echo -e "${GREEN}PASS${NC} ($http_code)"
        ((passed++))
    else
        echo -e "${RED}FAIL${NC} ($http_code)"
        ((failed++))
    fi
}

echo "1. BUSINESS CALENDAR TESTS"
echo "-----------------------------------"
test_endpoint "Get Business Calendars" GET "/api/calendar"
test_endpoint "Check Trading Day" GET "/api/calendar/NYSE/is-business-day?date=2025-01-15"
test_endpoint "Add Business Days" GET "/api/calendar/NYSE/add-business-days?start_date=2025-01-15&days=5"

echo ""
echo "2. DIRECT INDEXING TESTS"
echo "-----------------------------------"
test_endpoint "Get Harvest Opportunities" GET "/api/direct-indexing/opportunities?client_id=$CLIENT_ID"
test_endpoint "Execute Harvest" POST "/api/direct-indexing/execute" '{"opportunity_id":"opp-123","execute":true}'

echo ""
echo "3. CLIENT ONBOARDING TESTS"
echo "-----------------------------------"
test_endpoint "Create Onboarding Session" POST "/api/onboarding/sessions" "{\"client_id\":\"$CLIENT_ID\",\"tenant_id\":\"$TENANT_ID\"}"
test_endpoint "Get Session Progress" GET "/api/onboarding/sessions/session-123/progress"
test_endpoint "Upload Document" POST "/api/onboarding/documents" '{"session_id":"session-123","document_type":"DRIVERS_LICENSE"}'

echo ""
echo "4. CLIENT PORTAL TESTS"
echo "-----------------------------------"
test_endpoint "Get Portal Preferences" GET "/api/portal/preferences"
test_endpoint "Update Preferences" PUT "/api/portal/preferences" '{"theme":"dark"}'
test_endpoint "Track Analytics" POST "/api/portal/analytics" "{\"event_type\":\"PAGE_VIEW\",\"client_id\":\"$CLIENT_ID\"}"
test_endpoint "Get Engagement Metrics" GET "/api/portal/metrics?days=30"

echo ""
echo "5. SECURE MESSAGING TESTS"
echo "-----------------------------------"
test_endpoint "Get Message Threads" GET "/api/messages/threads"
test_endpoint "Send Message" POST "/api/messages/threads/thread-123/messages" '{"message_content":"Test message"}'
test_endpoint "Mark Read" POST "/api/messages/threads/thread-123/mark-read" '{}'

echo ""
echo "6. ALTERNATIVE INVESTMENTS TESTS"
echo "-----------------------------------"
test_endpoint "Get Investments" GET "/api/alternative-investments"
test_endpoint "Get Capital Calls" GET "/api/alternative-investments/capital-calls?status=PENDING"
test_endpoint "Get Performance" GET "/api/alternative-investments/inv-123/performance"

echo ""
echo "7. ESG INTELLIGENCE TESTS"
echo "-----------------------------------"
test_endpoint "Get ESG Metrics" GET "/api/esg/portfolio-metrics"
test_endpoint "Get SDG Impact" GET "/api/esg/sdg-impact"
test_endpoint "Get Violations" GET "/api/esg/violations?status=OPEN"
test_endpoint "Check Compliance" POST "/api/esg/check-compliance" "{\"client_id\":\"$CLIENT_ID\"}"

echo ""
echo "8. ADVANCED TAX PLANNING TESTS"
echo "-----------------------------------"
test_endpoint "Get Tax Profile" GET "/api/tax/profile?client_id=$CLIENT_ID"
test_endpoint "Get State Allocations" GET "/api/tax/state-allocations?client_id=$CLIENT_ID&tax_year=2024"
test_endpoint "Calculate AMT" POST "/api/tax/calculate-amt" "{\"client_id\":\"$CLIENT_ID\",\"tax_year\":2024}"
test_endpoint "Get Tax Recommendations" GET "/api/tax/recommendations?client_id=$CLIENT_ID"

echo ""
echo "9. SCHEDULER TESTS"
echo "-----------------------------------"
test_endpoint "Calculate Settlement" POST "/api/scheduler/settlement-date" '{"trade_date":"2025-01-15","security_type":"EQUITY","custodian":"SCHWAB"}'
test_endpoint "Get Upcoming Deadlines" GET "/api/scheduler/deadlines?days=30"
test_endpoint "Calculate RMD" POST "/api/scheduler/calculate-rmd" "{\"client_id\":\"$CLIENT_ID\",\"tax_year\":2025}"

echo ""
echo "10. NBA (NEXT BEST ACTION) TESTS"
echo "-----------------------------------"
test_endpoint "Get Recommendations" GET "/api/nba/recommendations?status=PENDING"
test_endpoint "Get Actions" GET "/api/nba/actions"
test_endpoint "Execute Recommendation" POST "/api/nba/recommendations/rec-123/execute" '{}'
test_endpoint "Get Statistics" GET "/api/nba/statistics"

echo ""
echo "==================================="
echo "TEST SUMMARY"
echo "==================================="
echo -e "${GREEN}Passed: $passed${NC}"
echo -e "${RED}Failed: $failed${NC}"
echo "Total: $((passed + failed))"
echo ""

if [ $failed -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    exit 0
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    exit 1
fi
