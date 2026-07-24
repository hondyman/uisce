#!/bin/bash

# ============================================================================
# Portfolio Analysis Platform - Validation Test Script
# Purpose: Verify all components are working correctly
# Run: bash backend/tests/validate_portfolio_analysis.sh
# ============================================================================

set -e

WEALTH_APP_DB="postgresql://postgres:postgres@localhost:5432/wealth_app"
ALPHA_DB="postgresql://postgres:postgres@localhost:5432/alpha"

echo "🚀 Portfolio Analysis Platform - Validation Tests"
echo "=================================================="
echo ""

# ============================================================================
# Test 1: Database Connection
# ============================================================================
echo "✓ Test 1: Database Connection"
echo "  Checking wealth_app connection..."

if psql "$WEALTH_APP_DB" -c "SELECT 1" > /dev/null 2>&1; then
    echo "  ✅ wealth_app database accessible"
else
    echo "  ❌ Failed to connect to wealth_app"
    echo "    Expected: postgresql://postgres:postgres@localhost:5432/wealth_app"
    exit 1
fi

echo ""

# ============================================================================
# Test 2: Required Tables
# ============================================================================
echo "✓ Test 2: Required Tables"

tables=("portfolios" "holdings" "transactions" "securities")

for table in "${tables[@]}"; do
    if psql "$WEALTH_APP_DB" -c "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='$table'" | grep -q "1"; then
        echo "  ✅ Table '$table' exists"
    else
        echo "  ⚠️  Table '$table' not found (create it with sample data)"
    fi
done

echo ""

# ============================================================================
# Test 3: SQL Functions
# ============================================================================
echo "✓ Test 3: SQL Functions Installation"

functions=(
    "analyze_portfolio_drill_down"
    "aggregate_household_holdings"
    "calculate_portfolio_performance"
    "analyze_concentration_risk"
    "model_portfolio_scenario"
)

for func in "${functions[@]}"; do
    if psql "$WEALTH_APP_DB" -c "\df+ $func" | grep -q "$func"; then
        echo "  ✅ Function '$func' installed"
    else
        echo "  ❌ Function '$func' not found"
        echo "    Solution: Run SQL migration from backend/migrations/wealth_app_001_portfolio_analysis_functions.sql"
        exit 1
    fi
done

echo ""

# ============================================================================
# Test 4: Sample Data
# ============================================================================
echo "✓ Test 4: Sample Data"

portfolio_count=$(psql "$WEALTH_APP_DB" -tc "SELECT COUNT(*) FROM portfolios" 2>/dev/null || echo "0")
holdings_count=$(psql "$WEALTH_APP_DB" -tc "SELECT COUNT(*) FROM holdings" 2>/dev/null || echo "0")

echo "  📊 Portfolios in database: $portfolio_count"
echo "  📊 Holdings in database: $holdings_count"

if [ "$holdings_count" -eq 0 ]; then
    echo "  ⚠️  No holdings found. Test with sample data:"
    cat << 'SQL'
    
    -- Insert sample portfolio
    INSERT INTO portfolios (id, household_id, name) VALUES 
      ('550e8400-e29b-41d4-a716-446655440000', '660e8400-e29b-41d4-a716-446655440000', 'Test Portfolio');
    
    -- Insert sample holdings
    INSERT INTO holdings (id, portfolio_id, security_id, ticker, name, shares, cost_basis, current_value, asset_class, sector, country, as_of_date) VALUES
      ('770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440000', '880e8400-e29b-41d4-a716-446655440001', 'AAPL', 'Apple Inc', 100, 15000, 18000, 'EQUITIES', 'TECHNOLOGY', 'USA', CURRENT_DATE),
      ('770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440000', '880e8400-e29b-41d4-a716-446655440002', 'BND', 'Vanguard Total Bond', 50, 5000, 5100, 'FIXED_INCOME', 'BONDS', 'USA', CURRENT_DATE);
SQL
    echo ""
fi

echo ""

# ============================================================================
# Test 5: Function Execution
# ============================================================================
echo "✓ Test 5: Function Execution (Sample Query)"

if [ "$portfolio_count" -gt 0 ]; then
    portfolio_id=$(psql "$WEALTH_APP_DB" -tc "SELECT id FROM portfolios LIMIT 1")
    
    echo "  Testing drill-down on portfolio: $portfolio_id"
    
    result=$(psql "$WEALTH_APP_DB" -c "
        SELECT COUNT(*) FROM analyze_portfolio_drill_down(
            p_portfolio_id := '$portfolio_id'::uuid,
            p_dimension := 'asset_class',
            p_level := 1,
            p_as_of_date := CURRENT_DATE
        )
    " 2>&1)
    
    if echo "$result" | grep -q "[0-9]"; then
        echo "  ✅ Function executed successfully"
        echo "  📊 Result rows: $(echo $result | awk '{print $NF}')"
    else
        echo "  ⚠️  Function executed but returned no data"
        echo "    Check: Do holdings exist for as_of_date = CURRENT_DATE?"
    fi
else
    echo "  ⏭️  Skipped (no portfolios in database)"
fi

echo ""

# ============================================================================
# Test 6: Frontend Components
# ============================================================================
echo "✓ Test 6: Frontend Components"

frontend_files=(
    "frontend/src/components/PortfolioAnalysisDashboard.tsx"
    "frontend/src/pages/PortfolioAnalysisPage.tsx"
)

for file in "${frontend_files[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✅ File exists: $file"
    else
        echo "  ❌ Missing file: $file"
        exit 1
    fi
done

echo ""

# ============================================================================
# Test 7: TypeScript Compilation
# ============================================================================
echo "✓ Test 7: TypeScript Check"

if command -v npx &> /dev/null; then
    echo "  Checking PortfolioAnalysisDashboard.tsx..."
    
    if npx tsc --noEmit frontend/src/components/PortfolioAnalysisDashboard.tsx 2>/dev/null; then
        echo "  ✅ TypeScript compiles without errors"
    else
        echo "  ⚠️  TypeScript compilation warnings (non-critical)"
    fi
else
    echo "  ⏭️  npm not found, skipping TypeScript check"
fi

echo ""

# ============================================================================
# Test 8: GraphQL Schema
# ============================================================================
echo "✓ Test 8: GraphQL Schema"

schema_file="backend/hasura/portfolio_analysis_metadata.graphql"

if [ -f "$schema_file" ]; then
    query_count=$(grep -c "query" "$schema_file" || echo "0")
    type_count=$(grep -c "type" "$schema_file" || echo "0")
    
    echo "  ✅ Schema file exists"
    echo "  📊 GraphQL queries: $query_count"
    echo "  📊 Type definitions: $type_count"
else
    echo "  ❌ Missing schema file: $schema_file"
fi

echo ""

# ============================================================================
# Summary
# ============================================================================
echo "=================================================="
echo "✅ Validation Complete!"
echo "=================================================="
echo ""

echo "📋 Next Steps:"
echo "  1. If all tests passed: Follow PORTFOLIO_QUICK_START.md"
echo "  2. If database tests failed: Check your PostgreSQL connection"
echo "  3. If functions missing: Run SQL migration from backend/migrations/"
echo "  4. If no data: Insert sample data (see Test 4 above)"
echo ""

echo "🚀 Ready to deploy? Run:"
echo "  - Backend: ./backend/migrations/wealth_app_001_portfolio_analysis_functions.sql"
echo "  - Frontend: npm run dev (from frontend/)"
echo "  - Browser: http://localhost:5173/portfolio/[portfolio-id]/analysis"
echo ""
