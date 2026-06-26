#!/bin/bash
# NBA (Next Best Action) System - Phase 1 Test Suite
# Tests database, signal detection, and API endpoints

set -e

echo "========================================"
echo "NBA System - Phase 1 Test Suite"
echo "========================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-semlayer}
DB_USER=${DB_USER:-postgres}
API_URL=${API_URL:-http://localhost:8080}

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((TESTS_PASSED++))
    ((TESTS_RUN++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    echo -e "  ${RED}Error: $2${NC}"
    ((TESTS_FAILED++))
    ((TESTS_RUN++))
}

info() {
    echo -e "${YELLOW}ℹ INFO${NC}: $1"
}

# =============================================================================
# TEST 1: Database Schema Verification
# =============================================================================

echo "TEST 1: Database Schema Verification"
echo "--------------------------------------"

# Test 1.1: Check nba_signals table exists
if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1 FROM nba_signals LIMIT 1;" &> /dev/null; then
    pass "Table nba_signals exists"
else
    fail "Table nba_signals exists" "Table not found"
fi

# Test 1.2: Check nba_action_catalog table exists
if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1 FROM nba_action_catalog LIMIT 1;" &> /dev/null; then
    pass "Table nba_action_catalog exists"
else
    fail "Table nba_action_catalog exists" "Table not found"
fi

# Test 1.3: Check nba_recommendations table exists
if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1 FROM nba_recommendations LIMIT 1;" &> /dev/null; then
    pass "Table nba_recommendations exists"
else
    fail "Table nba_recommendations exists" "Table not found"
fi

# Test 1.4: Check nba_action_outcomes table exists
if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1 FROM nba_action_outcomes LIMIT 1;" &> /dev/null; then
    pass "Table nba_action_outcomes exists"
else
    fail "Table nba_action_outcomes exists" "Table not found"
fi

# Test 1.5: Verify action catalog has seed data
ACTION_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM nba_action_catalog;")
if [ "$ACTION_COUNT" -ge 50 ]; then
    pass "Action catalog has $ACTION_COUNT actions (expected ≥50)"
else
    fail "Action catalog has >= 50 actions" "Found only $ACTION_COUNT actions"
fi

# Test 1.6: Verify indexes exist
INDEX_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM pg_indexes WHERE tablename IN ('nba_signals', 'nba_recommendations', 'nba_action_outcomes');")
if [ "$INDEX_COUNT" -ge 8 ]; then
    pass "NBA indexes created ($INDEX_COUNT indexes found)"
else
    fail "NBA indexes created" "Found only $INDEX_COUNT indexes"
fi

echo ""

# =============================================================================
# TEST 2: Signal Detection Functionality
# =============================================================================

echo "TEST 2: Signal Detection Functionality"
echo "---------------------------------------"

# Test 2.1: Insert test portfolio data
info "Creating test client portfolio..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF > /dev/null
-- Create test tenant
INSERT INTO tenants (tenant_id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'Test Tenant NBA')
ON CONFLICT DO NOTHING;

-- Create test client
INSERT INTO clients (client_id, tenant_id, first_name, last_name, email)
VALUES ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Test', 'Client', 'test@example.com')
ON CONFLICT DO NOTHING;

-- Create test portfolio with unrealized loss
INSERT INTO portfolio_positions (position_id, tenant_id, client_id, symbol, quantity, cost_basis, current_price, asset_class)
VALUES 
    ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'AAPL', 100, 180.00, 150.00, 'EQUITY'),
    ('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'MSFT', 50, 350.00, 320.00, 'EQUITY')
ON CONFLICT DO NOTHING;
EOF

if [ $? -eq 0 ]; then
    pass "Test portfolio data created"
else
    fail "Test portfolio data created" "SQL insert failed"
fi

# Test 2.2: Manually detect signals using SQL
SIGNAL_TEST=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
    SELECT COUNT(*)
    FROM clients c
    JOIN portfolio_positions p ON c.client_id = p.client_id
    WHERE c.tenant_id = '00000000-0000-0000-0000-000000000001'
    AND p.current_price < p.cost_basis
    GROUP BY c.client_id
    HAVING SUM(p.quantity * (p.current_price - p.cost_basis)) < -10000;
")

if [ ! -z "$SIGNAL_TEST" ] && [ "$SIGNAL_TEST" -gt 0 ]; then
    pass "Unrealized loss signal detection logic working"
else
    fail "Unrealized loss signal detection logic" "No signals detected for test data"
fi

# Test 2.3: Test signal insertion
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF > /dev/null
INSERT INTO nba_signals (
    tenant_id, client_id, signal_type, signal_category, signal_strength,
    signal_data, processed
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    'TEST_SIGNAL',
    'PORTFOLIO',
    0.75,
    '{"test": true}'::jsonb,
    false
);
EOF

if [ $? -eq 0 ]; then
    pass "Signal insertion successful"
else
    fail "Signal insertion" "Insert failed"
fi

echo ""

# =============================================================================
# TEST 3: API Endpoint Testing
# =============================================================================

echo "TEST 3: API Endpoint Testing"
echo "-----------------------------"

# Test 3.1: Get action catalog
CATALOG_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/api/nba/actions")
HTTP_CODE=$(echo "$CATALOG_RESPONSE" | tail -n 1)
BODY=$(echo "$CATALOG_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    ACTION_COUNT_API=$(echo "$BODY" | jq -r '.total // 0')
    if [ "$ACTION_COUNT_API" -ge 50 ]; then
        pass "GET /api/nba/actions returns $ACTION_COUNT_API actions"
    else
        fail "GET /api/nba/actions returns >= 50 actions" "Only $ACTION_COUNT_API actions returned"
    fi
else
    fail "GET /api/nba/actions endpoint" "HTTP $HTTP_CODE (expected 200)"
fi

# Test 3.2: Get NBA stats
STATS_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/api/nba/stats?advisor_id=00000000-0000-0000-0000-000000000001")
HTTP_CODE=$(echo "$STATS_RESPONSE" | tail -n 1)

if [ "$HTTP_CODE" = "200" ]; then
    pass "GET /api/nba/stats endpoint working"
else
    fail "GET /api/nba/stats endpoint" "HTTP $HTTP_CODE (expected 200)"
fi

# Test 3.3: Get recommendations (should be empty for now)
REC_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_URL/api/nba/recommendations?advisor_id=00000000-0000-0000-0000-000000000001")
HTTP_CODE=$(echo "$REC_RESPONSE" | tail -n 1)

if [ "$HTTP_CODE" = "200" ]; then
    pass "GET /api/nba/recommendations endpoint working"
else
    fail "GET /api/nba/recommendations endpoint" "HTTP $HTTP_CODE (expected 200)"
fi

echo ""

# =============================================================================
# TEST 4: Helper Functions
# =============================================================================

echo "TEST 4: Helper Functions"
echo "------------------------"

# Test 4.1: calculate_nba_overall_score function
SCORE_TEST=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT calculate_nba_overall_score(0.8, 5000, 0.75);")
if [ ! -z "$SCORE_TEST" ]; then
    pass "calculate_nba_overall_score function exists and works"
else
    fail "calculate_nba_overall_score function" "Function not found or failed"
fi

# Test 4.2: expire_old_nba_recommendations function
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT expire_old_nba_recommendations();" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    pass "expire_old_nba_recommendations function works"
else
    fail "expire_old_nba_recommendations function" "Execution failed"
fi

echo ""

# =============================================================================
# TEST 5: Data Integrity & Constraints
# =============================================================================

echo "TEST 5: Data Integrity & Constraints"
echo "-------------------------------------"

# Test 5.1: Signal strength constraint (must be 0-1)
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF > /dev/null 2>&1
INSERT INTO nba_signals (tenant_id, client_id, signal_type, signal_category, signal_strength, signal_data)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'TEST', 'PORTFOLIO', 1.5, '{}');
EOF

if [ $? -ne 0 ]; then
    pass "Signal strength constraint enforced (rejects > 1.0)"
else
    fail "Signal strength constraint" "Constraint not enforced"
fi

# Test 5.2: Confidence score constraint
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF > /dev/null 2>&1
INSERT INTO nba_recommendations (tenant_id, client_id, advisor_id, action_id, confidence_score, reasoning)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 
        '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 1.5, 'test');
EOF

if [ $? -ne 0 ]; then
    pass "Recommendation confidence constraint enforced"
else
    fail "Recommendation confidence constraint" "Constraint not enforced"
fi

# Test 5.3: Advisor rating constraint (1-5)
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF > /dev/null 2>&1
INSERT INTO nba_action_outcomes (recommendation_id, client_id, advisor_id, action_id, advisor_rating)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 6);
EOF

if [ $? -ne 0 ]; then
    pass "Advisor rating constraint enforced (1-5 only)"
else
    fail "Advisor rating constraint" "Constraint not enforced"
fi

echo ""

# =============================================================================
# TEST 6: RLS Policies  
# =============================================================================

echo "TEST 6: Row Level Security"
echo "--------------------------"

# Test 6.1: Verify RLS is enabled
RLS_CHECK=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "
    SELECT COUNT(*) FROM pg_tables 
    WHERE schemaname = 'public' 
    AND tablename IN ('nba_signals', 'nba_recommendations', 'nba_action_outcomes')
    AND rowsecurity = true;
")

if [ "$RLS_CHECK" = "3" ]; then
    pass "RLS enabled on all NBA tables"
else
    fail "RLS enabled on all NBA tables" "RLS not enabled on all tables"
fi

echo ""

# =============================================================================
# Cleanup
# =============================================================================

echo "Cleaning up test data..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOF > /dev/null
DELETE FROM nba_signals WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM portfolio_positions WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM clients WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM tenants WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
EOF

echo ""

# =============================================================================
# Summary
# =============================================================================

echo "========================================"
echo "Test Summary"
echo "========================================"
echo "Tests run:    $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED!${NC}"
    echo "NBA Phase 1 is ready for production."
    exit 0
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    echo "Please review the failures above."
    exit 1
fi
