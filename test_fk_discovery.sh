#!/bin/bash

# FK Discovery Test Script
# Tests the relationship discovery endpoint end-to-end

set -e

TENANT_ID="00000000-0000-0000-0000-000000000000"
DATASOURCE_ID="982aef38-418f-46dc-acd0-35fe8f3b97b0"
API_BASE="http://localhost:8080/api"

echo "=== Foreign Key Discovery Endpoint Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Test Parameters:"
echo "  Tenant ID: $TENANT_ID"
echo "  Datasource ID: $DATASOURCE_ID"
echo "  API Base: $API_BASE"
echo ""

# Test 1: orders -> customers
echo "${YELLOW}Test 1: Discovery for 'orders' table${NC}"
RESPONSE=$(curl -s -X GET "${API_BASE}/relationships/objects?entity=orders&tenant_id=${TENANT_ID}&datasource_id=${DATASOURCE_ID}")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q "customers"; then
    echo -e "${GREEN}✅ PASS: Found customers relationship${NC}"
else
    echo -e "${RED}❌ FAIL: customers not found${NC}"
fi
echo ""

# Test 2: customers <- orders  
echo "${YELLOW}Test 2: Discovery for 'customers' table${NC}"
RESPONSE=$(curl -s -X GET "${API_BASE}/relationships/objects?entity=customers&tenant_id=${TENANT_ID}&datasource_id=${DATASOURCE_ID}")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q "orders"; then
    echo -e "${GREEN}✅ PASS: Found orders relationship${NC}"
else
    echo -e "${RED}❌ FAIL: orders not found${NC}"
fi
echo ""

# Test 3: products relationships
echo "${YELLOW}Test 3: Discovery for 'products' table${NC}"
RESPONSE=$(curl -s -X GET "${API_BASE}/relationships/objects?entity=products&tenant_id=${TENANT_ID}&datasource_id=${DATASOURCE_ID}")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -qE "categories|suppliers"; then
    echo -e "${GREEN}✅ PASS: Found product relationships${NC}"
else
    echo -e "${RED}❌ FAIL: categories/suppliers not found${NC}"
fi
echo ""

# Test 4: Missing parameters
echo "${YELLOW}Test 4: Missing tenant_id parameter (should fail gracefully)${NC}"
RESPONSE=$(curl -s -X GET "${API_BASE}/relationships/objects?entity=orders&datasource_id=${DATASOURCE_ID}")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q "Missing required parameters"; then
    echo -e "${GREEN}✅ PASS: Correctly rejected missing tenant_id${NC}"
else
    echo -e "${RED}❌ FAIL: Should have rejected missing tenant_id${NC}"
fi
echo ""

echo "${YELLOW}=== All Tests Complete ===${NC}"
