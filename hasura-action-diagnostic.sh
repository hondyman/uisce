#!/bin/bash

# Hasura Action Integration Diagnostic Script
# This script verifies all components of the search_business_terms integration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000000}"
DATASOURCE_ID="${DATASOURCE_ID:-11111111-1111-1111-1111-111111111111}"
BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8001}"
HASURA_URL="${HASURA_URL:-http://localhost:8080}"

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Helper functions
print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"
}

print_test() {
    echo -e "${YELLOW}→${NC} $1"
}

print_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

print_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED++))
}

print_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

print_summary() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Passed:${NC}  $PASSED"
    echo -e "${RED}Failed:${NC}  $FAILED"
    echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"
}

# ============================================================================
# Test 1: Docker Services Running
# ============================================================================
print_header "Test 1: Docker Services Status"

print_test "Checking if docker-compose services are running..."

if ! command -v docker &> /dev/null; then
    print_fail "Docker is not installed"
else
    print_pass "Docker is installed"
    
    # Check each service
    for service in backend api-gateway hasura; do
        if docker ps --format '{{.Names}}' | grep -q "$service"; then
            print_pass "$service is running"
        else
            print_fail "$service is not running"
            echo "  Run: docker-compose up -d $service"
        fi
    done
fi

# ============================================================================
# Test 2: Connectivity Tests
# ============================================================================
print_header "Test 2: Service Connectivity"

# Test Backend
print_test "Testing backend connectivity (${BACKEND_URL})..."
if curl -s -o /dev/null -w "%{http_code}" "${BACKEND_URL}/api/health" | grep -q "200"; then
    print_pass "Backend is reachable"
else
    print_fail "Backend is not reachable"
fi

# Test API Gateway
print_test "Testing API Gateway connectivity (${GATEWAY_URL})..."
if curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/health" | grep -q "200"; then
    print_pass "API Gateway is reachable"
else
    print_fail "API Gateway is not reachable"
fi

# Test Hasura
print_test "Testing Hasura connectivity (${HASURA_URL})..."
if curl -s -o /dev/null -w "%{http_code}" "${HASURA_URL}/healthz" | grep -q "200"; then
    print_pass "Hasura is reachable"
else
    print_fail "Hasura is not reachable"
fi

# ============================================================================
# Test 3: Backend Endpoint Verification
# ============================================================================
print_header "Test 3: Backend Endpoint"

print_test "Testing POST ${BACKEND_URL}/business-terms/search..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BACKEND_URL}/business-terms/search" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{"search_term": "test", "limit": 5}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [[ "$HTTP_CODE" == "200" ]]; then
    print_pass "Backend /business-terms/search endpoint is working (HTTP $HTTP_CODE)"
    if echo "$BODY" | grep -q "terms"; then
        print_pass "Backend response contains 'terms' field"
    else
        print_warn "Backend response doesn't contain expected 'terms' field"
    fi
elif [[ "$HTTP_CODE" == "400" ]]; then
    print_fail "Backend returned 400 (Bad Request)"
    if echo "$BODY" | grep -q "headers are required"; then
        print_fail "Tenant headers are not being set correctly"
    fi
elif [[ "$HTTP_CODE" == "404" ]]; then
    print_fail "Backend endpoint not found (HTTP 404)"
    echo "  Expected: POST /business-terms/search"
else
    print_fail "Backend returned unexpected HTTP code: $HTTP_CODE"
fi

# ============================================================================
# Test 4: API Gateway Route Verification
# ============================================================================
print_header "Test 4: API Gateway Route"

print_test "Testing POST ${GATEWAY_URL}/api/search/business-terms..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${GATEWAY_URL}/api/search/business-terms" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{"search_term": "test", "limit": 5}')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [[ "$HTTP_CODE" == "200" ]]; then
    print_pass "API Gateway route is registered and working (HTTP $HTTP_CODE)"
    if echo "$BODY" | grep -q "terms"; then
        print_pass "API Gateway response contains 'terms' field"
    else
        print_warn "API Gateway response doesn't contain expected 'terms' field"
    fi
elif [[ "$HTTP_CODE" == "404" ]]; then
    print_fail "API Gateway route not found (HTTP 404)"
    echo "  Expected route: POST /api/search/business-terms"
    echo "  Verify: /Users/eganpj/GitHub/semlayer/api-gateway/main.go line 944"
else
    print_fail "API Gateway returned unexpected HTTP code: $HTTP_CODE"
    echo "  Response: $BODY"
fi

# ============================================================================
# Test 5: Hasura Action Metadata
# ============================================================================
print_header "Test 5: Hasura Action Metadata"

print_test "Checking if search_business_terms action is registered in Hasura..."

# Try to get metadata (may require auth)
METADATA=$(curl -s -X POST "${HASURA_URL}/v1/metadata" \
    -H "Content-Type: application/json" \
    -H "X-Hasura-Admin-Secret: ${HASURA_ADMIN_SECRET:-newadminsecretkey}" \
    -d '{"type": "export_metadata"}' 2>/dev/null || echo '{}')

if echo "$METADATA" | grep -q "search_business_terms"; then
    print_pass "search_business_terms action is registered in Hasura"
    
    # Check if it's a query (not mutation)
    if echo "$METADATA" | grep -A5 "search_business_terms" | grep -q '"type".*"query"'; then
        print_pass "Action is correctly defined as type: query"
    elif echo "$METADATA" | grep -A5 "search_business_terms" | grep -q '"type".*"mutation"'; then
        print_fail "Action is incorrectly defined as type: mutation (should be query)"
    else
        print_warn "Could not determine action type from metadata"
    fi
else
    print_warn "Could not verify action in metadata (may require authentication)"
fi

# ============================================================================
# Test 6: End-to-End Hasura GraphQL Query
# ============================================================================
print_header "Test 6: End-to-End GraphQL Query"

print_test "Testing GraphQL query through Hasura..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${HASURA_URL}/v1/graphql" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: ${TENANT_ID}" \
  -H "X-Tenant-Datasource-ID: ${DATASOURCE_ID}" \
  -d '{
    "query": "query { search_business_terms(search_term: \"test\", limit: 5) { id term_name } }"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [[ "$HTTP_CODE" == "200" ]]; then
    print_pass "Hasura GraphQL endpoint responded (HTTP $HTTP_CODE)"
    
    if echo "$BODY" | grep -q '"data"'; then
        print_pass "Response contains data field"
        
        if echo "$BODY" | grep -q '"search_business_terms"'; then
            print_pass "Response contains search_business_terms result"
        else
            if echo "$BODY" | grep -q '"errors"'; then
                print_fail "GraphQL query returned an error"
                echo "  Error: $(echo "$BODY" | grep -o '"message":"[^"]*"')"
            else
                print_warn "Response doesn't contain search_business_terms result"
            fi
        fi
    else
        print_fail "Response doesn't contain data field"
        echo "  Response: $BODY"
    fi
else
    print_fail "Hasura GraphQL endpoint returned HTTP $HTTP_CODE"
fi

# ============================================================================
# Test 7: Configuration Verification
# ============================================================================
print_header "Test 7: Configuration Verification"

print_test "Checking Hasura actions.yaml..."
if grep -q "search_business_terms" "/Users/eganpj/GitHub/semlayer/hasura/metadata/actions.yaml"; then
    print_pass "search_business_terms found in actions.yaml"
    
    if grep -A10 "name: search_business_terms" "/Users/eganpj/GitHub/semlayer/hasura/metadata/actions.yaml" | grep -q "type: query"; then
        print_pass "Action type is set to query (correct)"
    else
        print_warn "Action type verification inconclusive"
    fi
else
    print_fail "search_business_terms not found in actions.yaml"
fi

print_test "Checking API Gateway routes..."
if grep -q "/search/business-terms" "/Users/eganpj/GitHub/semlayer/api-gateway/main.go"; then
    print_pass "/search/business-terms route found in api-gateway"
else
    print_fail "/search/business-terms route not found in api-gateway"
fi

print_test "Checking backend endpoint..."
if grep -q '"/business-terms/search"' "/Users/eganpj/GitHub/semlayer/backend/internal/api/api.go"; then
    print_pass "/business-terms/search endpoint found in backend"
else
    print_fail "/business-terms/search endpoint not found in backend"
fi

# ============================================================================
# Summary
# ============================================================================
print_summary

if [[ $FAILED -eq 0 ]]; then
    echo -e "${GREEN}🎉 All critical tests passed! Integration appears to be working correctly.${NC}\n"
    exit 0
elif [[ $FAILED -lt 3 ]]; then
    echo -e "${YELLOW}⚠️  Some tests failed. Check the issues above and consult the troubleshooting guide.${NC}\n"
    exit 1
else
    echo -e "${RED}❌ Multiple critical tests failed. Integration needs attention.${NC}\n"
    exit 2
fi
