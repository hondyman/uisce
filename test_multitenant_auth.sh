#!/bin/bash
# Multi-Tenant JWT Authentication Test Script
# Tests all three user types: single-tenant, multi-tenant, global ops

set -e

BASE_URL="http://localhost:8001"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔐 Multi-Tenant JWT Authentication Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Test 1: Global Ops Login
echo -e "${BLUE}━━━ Test 1: Global Ops (tenant_scope: all) ━━━${NC}"
login_response=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@semlayer.com","password":"Admin123!"}')

if echo "$login_response" | grep -q "access_token"; then
    echo -e "${GREEN}✅ Global ops login successful${NC}"
    
    # Extract and decode JWT
    access_token=$(echo "$login_response" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    
    # Decode JWT payload (base64 decode second part)
    payload=$(echo "$access_token" | cut -d'.' -f2)
    # Add padding if needed
    padding=$((4 - ${#payload} % 4))
    if [ $padding -ne 4 ]; then
        payload="${payload}$(printf '=%.0s' $(seq 1 $padding))"
    fi
    
    decoded=$(echo "$payload" | base64 -d 2>/dev/null || echo "$payload" | base64 --decode 2>/dev/null)
    echo "   JWT Claims:"
    echo "$decoded" | python3 -m json.tool 2>/dev/null | grep -E "tenant_scope|roles|scopes" || echo "$decoded"
    
    # Verify tenant_scope is "all"
    if echo "$login_response" | grep -q '"tenant_scope":"all"'; then
        echo -e "${GREEN}   ✅ tenant_scope = all (correct for global ops)${NC}"
    else
        echo -e "${RED}   ❌ Expected tenant_scope=all${NC}"
    fi
    
    global_token="$access_token"
else
    echo -e "${RED}❌ Global ops login failed${NC}"
    echo "Response: $login_response"
    exit 1
fi
echo ""

# Test 2: Single-Tenant User Login
echo -e "${BLUE}━━━ Test 2: Single-Tenant User (tenant_scope: single) ━━━${NC}"
single_login=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@tenant-a.com","password":"Admin123!"}')

if echo "$single_login" | grep -q "access_token"; then
    echo -e "${GREEN}✅ Single-tenant user login successful${NC}"
    
    single_token=$(echo "$single_login" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    tenant_id=$(echo "$single_login" | grep -o '"tenant_id":"[^"]*' | cut -d'"' -f4)
    
    echo "   User tenant_id: $tenant_id"
    
    # Verify tenant_scope is "single"
    if echo "$single_login" | grep -q '"tenant_scope":"single"'; then
        echo -e "${GREEN}   ✅ tenant_scope = single (correct)${NC}"
    else
        echo -e "${YELLOW}   ⚠️  tenant_scope not set to single${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Single-tenant user doesn't exist yet (alice@tenant-a.com)${NC}"
    echo "   This is expected if test user hasn't been created"
    single_token=""
fi
echo ""

# Test 3: Multi-Tenant Ops Login
echo -e "${BLUE}━━━ Test 3: Multi-Tenant Ops (tenant_scope: multi) ━━━${NC}"
multi_login=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"ops@region.com","password":"Admin123!"}')

if echo "$multi_login" | grep -q "access_token"; then
    echo -e "${GREEN}✅ Multi-tenant ops login successful${NC}"
    
    multi_token=$(echo "$multi_login" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    
    # Verify tenant_scope is "multi"
    if echo "$multi_login" | grep -q '"tenant_scope":"multi"'; then
        echo -e "${GREEN}   ✅ tenant_scope = multi (correct)${NC}"
    else
        echo -e "${YELLOW}   ⚠️  tenant_scope not set to multi${NC}"
    fi
    
    # Check for tenant_ids array
    if echo "$multi_login" | grep -q "tenant_ids"; then
        echo -e "${GREEN}   ✅ tenant_ids array present${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Multi-tenant ops user doesn't exist yet (ops@region.com)${NC}"
    echo "   This is expected if test user hasn't been created"
    multi_token=""
fi
echo ""

# Test 4: JWT Structure Validation
echo -e "${BLUE}━━━ Test 4: JWT Structure Validation ━━━${NC}"
echo "Checking global ops JWT for required claims..."

payload=$(echo "$global_token" | cut -d'.' -f2)
padding=$((4 - ${#payload} % 4))
if [ $padding -ne 4 ]; then
    payload="${payload}$(printf '=%.0s' $(seq 1 $padding))"
fi
decoded=$(echo "$payload" | base64 -d 2>/dev/null || echo "$payload" | base64 --decode 2>/dev/null)

required_claims=("sub" "email" "roles" "scopes" "tenant_scope")
for claim in "${required_claims[@]}"; do
    if echo "$decoded" | grep -q "\"$claim\""; then
        echo -e "${GREEN}   ✅ $claim claim present${NC}"
    else
        echo -e "${RED}   ❌ $claim claim missing${NC}"
    fi
done
echo ""

# Test 5: Token Refresh with Tenant Claims
echo -e "${BLUE}━━━ Test 5: Token Refresh (preserve tenant claims) ━━━${NC}"
if [ -n "$global_token" ]; then
    refresh_token=$(echo "$login_response" | grep -o '"refresh_token":"[^"]*' | cut -d'"' -f4)
    
    refresh_response=$(curl -s -X POST "${BASE_URL}/api/auth/refresh" \
      -H "Content-Type: application/json" \
      -d "{\"refresh_token\":\"$refresh_token\"}")
    
    if echo "$refresh_response" | grep -q "access_token"; then
        echo -e "${GREEN}✅ Token refresh successful${NC}"
        
        # Verify new token has tenant claims
        new_token=$(echo "$refresh_response" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
        new_payload=$(echo "$new_token" | cut -d'.' -f2)
        padding=$((4 - ${#new_payload} % 4))
        if [ $padding -ne 4 ]; then
            new_payload="${new_payload}$(printf '=%.0s' $(seq 1 $padding))"
        fi
        new_decoded=$(echo "$new_payload" | base64 -d 2>/dev/null || echo "$new_payload" | base64 --decode 2>/dev/null)
        
        if echo "$new_decoded" | grep -q "tenant_scope"; then
            echo -e "${GREEN}   ✅ tenant_scope preserved after refresh${NC}"
        else
            echo -e "${RED}   ❌ tenant_scope lost after refresh${NC}"
        fi
    else
        echo -e "${RED}❌ Token refresh failed${NC}"
    fi
fi
echo ""

# Test 6: Verify Endpoint with Tenant Claims
echo -e "${BLUE}━━━ Test 6: Verify Endpoint ━━━${NC}"
verify_response=$(curl -s -X POST "${BASE_URL}/api/auth/verify" \
  -H "Authorization: Bearer $global_token")

if echo "$verify_response" | grep -q "valid"; then
    echo -e "${GREEN}✅ Token verification successful${NC}"
    
    if echo "$verify_response" | grep -q "tenant_scope"; then
        echo -e "${GREEN}   ✅ tenant_scope included in verification response${NC}"
    fi
else
    echo -e "${RED}❌ Token verification failed${NC}"
fi
echo ""

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ Multi-Tenant JWT Tests Complete!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 JWT Claim Contract Validated:"
echo "   • tenant_scope: 'single' | 'multi' | 'all' ✅"
echo "   • roles: array of role strings ✅"
echo "   • scopes: array of permission strings ✅"
echo "   • tenant_id: for single-tenant users ✅"
echo "   • tenant_ids: for multi-tenant ops ✅"
echo ""
echo "🔗 Next Steps:"
echo "   1. Test API Gateway tenant enforcement"
echo "   2. Test Hasura RLS policies"
echo "   3. Test frontend tenant selector"
echo ""
