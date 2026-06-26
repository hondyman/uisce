#!/bin/bash

# Custom Components API Testing Script
# Tests all 8 endpoints with tenant scope enforcement

set -e

# Configuration
API_BASE="http://localhost:8080/api"
TENANT_ID="00000000-0000-0000-0000-000000000001"
DATASOURCE_ID="00000000-0000-0000-0000-000000000001"
COMPONENT_ID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to print test results
test_result() {
    local test_name=$1
    local status=$2
    local message=$3
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name - $message"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Test 1: Create Custom Component
echo -e "\n${YELLOW}Test 1: Create Custom Component${NC}"
CREATE_RESPONSE=$(curl -s -X POST "$API_BASE/custom-components?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "name": "Test Sales Chart",
    "type": "chart",
    "config": {
      "chartType": "bar",
      "dataSource": "api://sales-data"
    },
    "events": [
      {
        "eventName": "onClick",
        "action": "filter",
        "targetComponentId": "list-component"
      }
    ],
    "filters": [],
    "description": "Test chart component"
  }')

COMPONENT_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

if [ ! -z "$COMPONENT_ID" ]; then
    test_result "Create component" "PASS" ""
    echo "  Component ID: $COMPONENT_ID"
else
    test_result "Create component" "FAIL" "No component ID returned: $CREATE_RESPONSE"
fi

# Test 2: List Custom Components
echo -e "\n${YELLOW}Test 2: List Custom Components${NC}"
LIST_RESPONSE=$(curl -s -X GET "$API_BASE/custom-components?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID")

if echo "$LIST_RESPONSE" | grep -q "Test Sales Chart"; then
    test_result "List components" "PASS" ""
else
    test_result "List components" "FAIL" "Component not found in list"
fi

# Test 3: Get Custom Component
echo -e "\n${YELLOW}Test 3: Get Custom Component${NC}"
GET_RESPONSE=$(curl -s -X GET "$API_BASE/custom-components/$COMPONENT_ID?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID")

if echo "$GET_RESPONSE" | grep -q "Test Sales Chart"; then
    test_result "Get component" "PASS" ""
else
    test_result "Get component" "FAIL" "Component not retrieved correctly"
fi

# Test 4: Update Custom Component
echo -e "\n${YELLOW}Test 4: Update Custom Component${NC}"
UPDATE_RESPONSE=$(curl -s -X PUT "$API_BASE/custom-components/$COMPONENT_ID?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "name": "Updated Sales Chart",
    "type": "chart",
    "config": {
      "chartType": "line",
      "dataSource": "api://sales-data-updated"
    },
    "events": [],
    "filters": [],
    "description": "Updated test chart"
  }')

if echo "$UPDATE_RESPONSE" | grep -q "Updated Sales Chart"; then
    test_result "Update component" "PASS" ""
else
    test_result "Update component" "FAIL" "Component not updated correctly"
fi

# Test 5: Export Components
echo -e "\n${YELLOW}Test 5: Export Components (JSON)${NC}"
EXPORT_RESPONSE=$(curl -s -X GET "$API_BASE/custom-components/export?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID&format=json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID")

if echo "$EXPORT_RESPONSE" | grep -q "Updated Sales Chart"; then
    test_result "Export as JSON" "PASS" ""
    echo "$EXPORT_RESPONSE" > /tmp/components_export.json
else
    test_result "Export as JSON" "FAIL" "Export failed"
fi

# Test 6: Import Components
echo -e "\n${YELLOW}Test 6: Import Components${NC}"
if [ -f /tmp/components_export.json ]; then
    IMPORT_RESPONSE=$(curl -s -X POST "$API_BASE/custom-components/import?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
      -H "X-Tenant-ID: $TENANT_ID" \
      -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
      -F "file=@/tmp/components_export.json")
    
    if echo "$IMPORT_RESPONSE" | grep -q "imported"; then
        test_result "Import components" "PASS" ""
    else
        test_result "Import components" "FAIL" "Import failed: $IMPORT_RESPONSE"
    fi
else
    test_result "Import components" "SKIP" "Export file not found"
fi

# Test 7: Test API Endpoint
echo -e "\n${YELLOW}Test 7: Test Component API${NC}"
TEST_API_RESPONSE=$(curl -s -X POST "$API_BASE/custom-components/test-api?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "url": "https://api.github.com/repos/eganpj/semlayer",
    "method": "GET",
    "headers": {}
  }')

if echo "$TEST_API_RESPONSE" | grep -q "200"; then
    test_result "Test API endpoint" "PASS" ""
else
    test_result "Test API endpoint" "FAIL" "API test failed"
fi

# Test 8: Delete Custom Component
echo -e "\n${YELLOW}Test 8: Delete Custom Component${NC}"
DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE/custom-components/$COMPONENT_ID?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID")

HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "204" ]; then
    test_result "Delete component" "PASS" ""
else
    test_result "Delete component" "FAIL" "Expected 204, got $HTTP_CODE"
fi

# Test 9: Tenant Scope Isolation
echo -e "\n${YELLOW}Test 9: Tenant Scope Isolation${NC}"
WRONG_TENANT="00000000-0000-0000-0000-000000000099"
ISOLATION_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE/custom-components?tenant_id=$WRONG_TENANT&datasource_id=$DATASOURCE_ID" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID")

HTTP_CODE=$(echo "$ISOLATION_RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "403" ]; then
    test_result "Tenant scope isolation" "PASS" ""
else
    test_result "Tenant scope isolation" "FAIL" "Expected 403 Forbidden, got $HTTP_CODE"
fi

# Summary
echo -e "\n${YELLOW}========== Test Summary ==========${NC}"
echo "Tests Run:    $TESTS_RUN"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo -e "${YELLOW}==================================${NC}\n"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
