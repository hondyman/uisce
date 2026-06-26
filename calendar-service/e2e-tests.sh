#!/bin/bash

# Phase 4 E2E Test Script - Simplified Version
set -e

BASE_URL="http://localhost:8080/api/v1"
JWT_SECRET="test-secret-key-abcdef1234567890"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_count=0
pass_count=0
fail_count=0

log_test() {
    local name="$1"
    local status="$2"
    local details="$3"
    
    test_count=$((test_count + 1))
    
    if [ "$status" = "PASS" ]; then
        pass_count=$((pass_count + 1))
        echo -e "${GREEN}✓ PASS${NC} [$test_count] $name"
    else
        fail_count=$((fail_count + 1))
        echo -e "${RED}✗ FAIL${NC} [$test_count] $name: $details"
    fi
}

# Generate JWT using openssl and base64
generate_jwt() {
    local tenant_id="$1"
    local user_id="$2"
    
    local header='{"alg":"HS256","typ":"JWT"}'
    local payload="{\"user_id\":\"$user_id\",\"tenant_id\":\"$tenant_id\",\"roles\":[\"admin\"]}"
    
    local b64_header=$(echo -n "$header" | base64 | tr '+/' '-_' | tr -d '=\n')
    local b64_payload=$(echo -n "$payload" | base64 | tr '+/' '-_' | tr -d '=\n')
    
    local message="$b64_header.$b64_payload"
    local signature=$(echo -n "$message" | openssl dgst -sha256 -mac HMAC -macopt key:$JWT_SECRET -binary | base64 | tr '+/' '-_' | tr -d '=\n')
    
    echo "$message.$signature"
}


echo "======================================"
echo "Phase 4: E2E Testing"
echo "======================================"
echo ""

TOKEN_TENANT_A=$(generate_jwt "tenant-a-id" "user-a")
TOKEN_TENANT_B=$(generate_jwt "tenant-b-id" "user-b")

echo "Tokens generated successfully"
echo ""

# SECTION 1: Authentication Tests
echo -e "${YELLOW}=== SECTION 1: Authentication Tests ===${NC}"

# Test 1.1: Missing JWT token
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/calendars")
[ "$status" = "401" ] && log_test "Missing JWT token returns 401" "PASS" || log_test "Missing JWT token returns 401" "FAIL" "Got $status"

# Test 1.2: Invalid JWT token
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/calendars" -H "Authorization: Bearer invalid")
[ "$status" = "401" ] && log_test "Invalid JWT token returns 401" "PASS" || log_test "Invalid JWT token returns 401" "FAIL" "Got $status"

echo ""

# SECTION 2: Calendar Endpoints
echo -e "${YELLOW}=== SECTION 2: Calendar Endpoints ===${NC}"

# Test 2.1: Create calendar
status=$(curl -s -o /tmp/cal_response.json -w "%{http_code}" -X POST "$BASE_URL/calendars" \
    -H "Authorization: Bearer $TOKEN_TENANT_A" \
    -H "Content-Type: application/json" \
    -d '{"name":"Q1 Calendar","timezone":"UTC"}')

if [ "$status" = "201" ] || [ "$status" = "200" ]; then
    log_test "Create calendar for Tenant A" "PASS"
    CALENDAR_ID_A=$(grep -o '"id":"[^"]*"' /tmp/cal_response.json 2>/dev/null | head -1 | cut -d'"' -f4)
else
    log_test "Create calendar for Tenant A" "FAIL" "Status: $status"
    CALENDAR_ID_A=""
fi

# Test 2.2: List calendars
status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/calendars" \
    -H "Authorization: Bearer $TOKEN_TENANT_A")
[ "$status" = "200" ] && log_test "List calendars" "PASS" || log_test "List calendars" "FAIL" "Got $status"

if [ -n "$CALENDAR_ID_A" ]; then
    # Test 2.3: Get calendar
    status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/calendars/$CALENDAR_ID_A" \
        -H "Authorization: Bearer $TOKEN_TENANT_A")
    [ "$status" = "200" ] && log_test "Get calendar" "PASS" || log_test "Get calendar" "FAIL" "Got $status"
    
    # Test 2.4: Update calendar
    status=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/calendars/$CALENDAR_ID_A" \
        -H "Authorization: Bearer $TOKEN_TENANT_A" \
        -H "Content-Type: application/json" \
        -d '{"name":"Updated Q1 Calendar","timezone":"UTC"}')
    [ "$status" = "200" ] && log_test "Update calendar" "PASS" || log_test "Update calendar" "FAIL" "Got $status"
fi

echo ""

# SECTION 3: Cross-Tenant Security
echo -e "${YELLOW}=== SECTION 3: Cross-Tenant Security ===${NC}"

if [ -n "$CALENDAR_ID_A" ]; then
    # Create calendar for Tenant B
    status=$(curl -s -o /tmp/cal_b.json -w "%{http_code}" -X POST "$BASE_URL/calendars" \
        -H "Authorization: Bearer $TOKEN_TENANT_B" \
        -H "Content-Type: application/json" \
        -d '{"name":"B Calendar","timezone":"Europe/London"}')
    CALENDAR_ID_B=$(grep -o '"id":"[^"]*"' /tmp/cal_b.json 2>/dev/null | head -1 | cut -d'"' -f4)
    
    # Test: Tenant B cannot access Tenant A's calendar
    status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/calendars/$CALENDAR_ID_A" \
        -H "Authorization: Bearer $TOKEN_TENANT_B")
    
    if [ "$status" = "403" ] || [ "$status" = "404" ]; then
        log_test "Cross-tenant access blocked (GET)" "PASS" "Status: $status"
    else
        log_test "Cross-tenant access blocked (GET)" "FAIL" "Got $status"
    fi
    
    # Test: Tenant B cannot update Tenant A's calendar
    status=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/calendars/$CALENDAR_ID_A" \
        -H "Authorization: Bearer $TOKEN_TENANT_B" \
        -H "Content-Type: application/json" \
        -d '{"name":"Hacked","timezone":"UTC"}')
    
    if [ "$status" = "403" ] || [ "$status" = "404" ]; then
        log_test "Cross-tenant access blocked (PUT)" "PASS" "Status: $status"
    else
        log_test "Cross-tenant access blocked (PUT)" "FAIL" "Got $status"
    fi
    
    # Test: Tenant B cannot delete Tenant A's calendar
    status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/calendars/$CALENDAR_ID_A" \
        -H "Authorization: Bearer $TOKEN_TENANT_B")
    
    if [ "$status" = "403" ] || [ "$status" = "404" ]; then
        log_test "Cross-tenant access blocked (DELETE)" "PASS" "Status: $status"
    else
        log_test "Cross-tenant access blocked (DELETE)" "FAIL" "Got $status"
    fi
fi

echo ""

# SECTION 4: Availability Endpoints
echo -e "${YELLOW}=== SECTION 4: Availability Endpoints ===${NC}"

if [ -n "$CALENDAR_ID_A" ]; then
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/availability" \
        -H "Authorization: Bearer $TOKEN_TENANT_A" \
        -H "Content-Type: application/json" \
        -d "{\"calendar_id\":\"$CALENDAR_ID_A\",\"start_time\":\"2024-02-20T09:00:00Z\",\"duration_secs\":3600}")
    [ "$status" = "200" ] || [ "$status" = "201" ] && log_test "Check availability" "PASS" || log_test "Check availability" "FAIL" "Got $status"
fi

echo ""

# SECTION 5: Blackout Endpoints
echo -e "${YELLOW}=== SECTION 5: Blackout Endpoints ===${NC}"

if [ -n "$CALENDAR_ID_A" ]; then
    # Create blackout
    status=$(curl -s -o /tmp/blackout.json -w "%{http_code}" -X POST "$BASE_URL/blackouts" \
        -H "Authorization: Bearer $TOKEN_TENANT_A" \
        -H "Content-Type: application/json" \
        -d "{\"calendar_id\":\"$CALENDAR_ID_A\",\"name\":\"Maintenance\",\"start_time\":\"2024-02-20T22:00:00Z\",\"end_time\":\"2024-02-21T06:00:00Z\"}")
    
    if [ "$status" = "200" ] || [ "$status" = "201" ]; then
        log_test "Create blackout" "PASS"
        BLACKOUT_ID=$(grep -o '"id":"[^"]*"' /tmp/blackout.json 2>/dev/null | head -1 | cut -d'"' -f4)
    else
        log_test "Create blackout" "FAIL" "Got $status"
        BLACKOUT_ID=""
    fi
    
    # Get blackout occurrences
    if [ -n "$BLACKOUT_ID" ]; then
        status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/blackouts/$BLACKOUT_ID/occurrences?start=2024-02-20T00:00:00Z&end=2024-02-28T23:59:59Z" \
            -H "Authorization: Bearer $TOKEN_TENANT_A")
        [ "$status" = "200" ] && log_test "Get blackout occurrences" "PASS" || log_test "Get blackout occurrences" "FAIL" "Got $status"
    fi
fi

echo ""

# SECTION 6: Tenant Management
echo -e "${YELLOW}=== SECTION 6: Tenant Management ===${NC}"

status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/tenants/tenant-a-id" \
    -H "Authorization: Bearer $TOKEN_TENANT_A")
[ "$status" = "200" ] && log_test "Get tenant" "PASS" || log_test "Get tenant" "FAIL" "Got $status"

echo ""

# SECTION 7: Delete Calendar (cleanup)
echo -e "${YELLOW}=== SECTION 7: Cleanup ===${NC}"

if [ -n "$CALENDAR_ID_A" ]; then
    status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/calendars/$CALENDAR_ID_A" \
        -H "Authorization: Bearer $TOKEN_TENANT_A")
    [ "$status" = "200" ] || [ "$status" = "204" ] && log_test "Delete calendar" "PASS" || log_test "Delete calendar" "FAIL" "Got $status"
fi

echo ""

# Summary
echo -e "${YELLOW}=====================================${NC}"
echo "Test Summary:"
echo "Total:  $test_count"
echo -e "Passed: ${GREEN}$pass_count${NC}"
if [ $fail_count -gt 0 ]; then
    echo -e "Failed: ${RED}$fail_count${NC}"
else
    echo -e "Failed: ${GREEN}$fail_count${NC}"
fi
echo -e "${YELLOW}=====================================${NC}"

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}✓ All E2E tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
