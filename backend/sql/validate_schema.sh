#!/bin/bash
# ============================================================================
# Phase 3.21: Schema Validation & Health Check Script
# ============================================================================
# Verifies that the feature engineering schema is correctly initialized.
# Checks table counts, indexes, views, and data integrity.
#
# Usage:
#   ./validate_schema.sh -h localhost -p 5432 -U postgres -d semlayer
# ============================================================================

set -e
set -o pipefail

# Configuration defaults
DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_NAME="semlayer"
PASSWORD=""

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
while getopts "h:p:U:d:P:" opt; do
    case $opt in
        h) DB_HOST="$OPTARG" ;;
        p) DB_PORT="$OPTARG" ;;
        U) DB_USER="$OPTARG" ;;
        d) DB_NAME="$OPTARG" ;;
        P) PASSWORD="$OPTARG" ;;
        *) echo "Usage: $0 [-h host] [-p port] [-U user] [-d database] [-P password]"; exit 1 ;;
    esac
done

echo -e "${BLUE}Phase 3.21 Feature Engineering Schema Validation${NC}"
echo "================================================================"
echo "Database: ${DB_NAME}@${DB_HOST}:${DB_PORT}"
echo ""

# Helper function to run SQL
run_sql() {
    local query="$1"
    if [ -z "$PASSWORD" ]; then
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null
    else
        PGPASSWORD="$PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null
    fi
}

# Test database connectivity
echo -e "${YELLOW}1. Testing Database Connectivity${NC}"
if run_sql "SELECT 1" > /dev/null; then
    echo -e "   ${GREEN}✓${NC} Connected to PostgreSQL"
else
    echo -e "   ${RED}✗${NC} Failed to connect to PostgreSQL"
    exit 1
fi

# Check tables exist
echo ""
echo -e "${YELLOW}2. Checking Core Tables${NC}"

EXPECTED_TABLES=(
    "feature_catalog"
    "feature_watermarks"
    "feature_drift_metrics"
    "feature_quality_checks"
    "feature_importance"
    "feature_change_log"
    "feature_test_cases"
    "feature_lineage"
    "feature_computations"
    "schema_migrations"
)

TABLE_COUNT=0
for table in "${EXPECTED_TABLES[@]}"; do
    if run_sql "SELECT to_regclass('public.$table')" | grep -q "public.$table"; then
        echo -e "   ${GREEN}✓${NC} $table"
        ((TABLE_COUNT++))
    else
        echo -e "   ${RED}✗${NC} $table (missing)"
    fi
done

echo "   Tables: ${TABLE_COUNT}/${#EXPECTED_TABLES[@]}"

# Check views
echo ""
echo -e "${YELLOW}3. Checking Views${NC}"

EXPECTED_VIEWS=(
    "feature_catalog_active"
    "quality_check_failures"
    "failing_tests"
    "pending_approvals"
    "feature_lineage_ancestors"
    "active_drifts"
    "top_features_by_model"
)

VIEW_COUNT=0
for view in "${EXPECTED_VIEWS[@]}"; do
    if run_sql "SELECT to_regclass('public.$view')" | grep -q "public.$view"; then
        echo -e "   ${GREEN}✓${NC} $view"
        ((VIEW_COUNT++))
    else
        echo -e "   ${RED}✗${NC} $view (missing)"
    fi
done

echo "   Views: ${VIEW_COUNT}/${#EXPECTED_VIEWS[@]}"

# Check indexes
echo ""
echo -e "${YELLOW}4. Checking Production Indexes${NC}"

INDEX_COUNT=$(run_sql "SELECT COUNT(*) FROM pg_indexes WHERE schemaname='public' AND indexname LIKE 'idx_%';" | tr -d ' ')
echo "   Total indexes: ${INDEX_COUNT}"
if [ "$INDEX_COUNT" -ge 30 ]; then
    echo -e "   ${GREEN}✓${NC} Comprehensive indexing in place"
else
    echo -e "   ${YELLOW}!${NC} Only $INDEX_COUNT indexes (expected ~35+)"
fi

# Check functions
echo ""
echo -e "${YELLOW}5. Checking Helper Functions${NC}"

EXPECTED_FUNCTIONS=(
    "get_feature_health"
    "get_feature_ancestors"
    "update_feature_catalog_timestamp"
    "update_test_cases_timestamp"
)

FUNC_COUNT=0
for func in "${EXPECTED_FUNCTIONS[@]}"; do
    if run_sql "SELECT 1 FROM information_schema.routines WHERE routine_name='$func' AND routine_schema='public';" | grep -q "1"; then
        echo -e "   ${GREEN}✓${NC} $func"
        ((FUNC_COUNT++))
    else
        echo -e "   ${RED}✗${NC} $func (missing)"
    fi
done

echo "   Functions: ${FUNC_COUNT}/${#EXPECTED_FUNCTIONS[@]}"

# Check sample data (if loaded)
echo ""
echo -e "${YELLOW}6. Checking Sample Data${NC}"

FEATURE_COUNT=$(run_sql "SELECT COUNT(*) FROM feature_catalog;" | tr -d ' ')
DRIFT_COUNT=$(run_sql "SELECT COUNT(*) FROM feature_drift_metrics;" | tr -d ' ')
IMPORTANCE_COUNT=$(run_sql "SELECT COUNT(*) FROM feature_importance;" | tr -d ' ')
TEST_COUNT=$(run_sql "SELECT COUNT(*) FROM feature_test_cases;" | tr -d ' ')

echo "   Features: ${FEATURE_COUNT}"
echo "   Drift metrics: ${DRIFT_COUNT}"
echo "   Importance records: ${IMPORTANCE_COUNT}"
echo "   Test cases: ${TEST_COUNT}"

if [ "$FEATURE_COUNT" -gt 0 ]; then
    echo -e "   ${GREEN}✓${NC} Sample data loaded"
else
    echo -e "   ${YELLOW}!${NC} No sample data (run: psql -d semlayer -f sample_data.sql)"
fi

# Check constraints
echo ""
echo -e "${YELLOW}7. Checking Constraints${NC}"

CONSTRAINT_COUNT=$(run_sql "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema='public' AND constraint_type IN ('FOREIGN KEY', 'PRIMARY KEY', 'UNIQUE', 'CHECK');" | tr -d ' ')
echo "   Total constraints: ${CONSTRAINT_COUNT}"
if [ "$CONSTRAINT_COUNT" -ge 20 ]; then
    echo -e "   ${GREEN}✓${NC} Comprehensive constraint enforcement"
else
    echo -e "   ${YELLOW}!${NC} Limited constraints (expected ~25+)"
fi

# Check permissions
echo ""
echo -e "${YELLOW}8. Checking Permissions${NC}"

# Check if PUBLIC can SELECT
if run_sql "SELECT 1 FROM feature_catalog LIMIT 1" > /dev/null 2>&1; then
    echo -e "   ${GREEN}✓${NC} Public READ access granted"
else
    echo -e "   ${YELLOW}!${NC} Public READ access may be restricted"
fi

# Detailed feature health check (if sample data exists)
if [ "$FEATURE_COUNT" -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}9. Feature Health Summary${NC}"
    
    run_sql "
    SELECT 
        fc.feature_id,
        fw.materialization_lag_seconds as lag_sec,
        (SELECT COUNT(*) FROM feature_drift_metrics 
         WHERE feature_id = fc.feature_id AND is_drifted) as active_drifts,
        (SELECT COUNT(*) FROM feature_test_cases 
         WHERE feature_id = fc.feature_id AND last_run_passed = FALSE AND enabled) as failing_tests
    FROM feature_catalog fc
    LEFT JOIN feature_watermarks fw ON fc.feature_id = fw.feature_id
    ORDER BY fc.feature_id
    LIMIT 10;
    " || true
fi

# Summary
echo ""
echo -e "${BLUE}Summary${NC}"
echo "================================================================"
if [ "$TABLE_COUNT" -eq ${#EXPECTED_TABLES[@]} ] && [ "$VIEW_COUNT" -eq ${#EXPECTED_VIEWS[@]} ] && [ "$FUNC_COUNT" -eq ${#EXPECTED_FUNCTIONS[@]} ]; then
    echo -e "${GREEN}✓ Schema validation PASSED${NC}"
    echo "  All core tables, views, and functions are in place."
    echo "  Ready for Phase 3.21 operations."
    exit 0
else
    echo -e "${YELLOW}! Schema validation INCOMPLETE${NC}"
    echo "  Some components may be missing."
    echo "  Run './init_schema.sh' to initialize."
    exit 1
fi
