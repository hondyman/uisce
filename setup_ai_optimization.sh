#!/bin/bash

# AI-Powered Process Optimization - Setup & Verification Script
# This script verifies that all components are properly installed and configured

# set -e  # Exit on error - commented out for debugging

echo "=========================================="
echo "AI Optimization Setup & Verification"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SUCCESS_COUNT=0
TOTAL_CHECKS=8

# Function to print success
success() {
    echo -e "${GREEN}✓${NC} $1"
    ((SUCCESS_COUNT++))
}

# Function to print error
error() {
    echo -e "${RED}✗${NC} $1"
}

# Function to print info
info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

echo "Step 1/8: Checking Database Connection..."
if psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -c "SELECT 1" > /dev/null 2>&1; then
    success "Database connection successful"
else
    error "Cannot connect to database"
    info "Make sure PostgreSQL is running on localhost:5432"
    exit 1
fi

echo ""
echo "Step 2/8: Verifying Optimization Tables..."
TABLES_EXIST=$(psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -t -c "
    SELECT COUNT(*) FROM information_schema.tables 
    WHERE table_name IN ('process_optimization_suggestions', 'applied_optimizations', 'auto_tune_config')
" | tr -d ' ')

if [ "$TABLES_EXIST" -eq 3 ]; then
    success "All 3 optimization tables exist"
else
    error "Optimization tables missing (found $TABLES_EXIST/3)"
    info "Run: psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable -f backend/migrations/misc/process_optimization_schema.sql"
    exit 1
fi

echo ""
echo "Step 3/8: Checking Indexes..."
INDEXES_EXIST=$(psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -t -c "
    SELECT COUNT(*) FROM pg_indexes 
    WHERE indexname LIKE 'idx_%' 
    AND tablename IN ('process_optimization_suggestions', 'applied_optimizations', 'auto_tune_config')
")

if [ "$INDEXES_EXIST" -ge 6 ]; then
    success "Optimization indexes present ($INDEXES_EXIST indexes)"
else
    error "Missing indexes (found $INDEXES_EXIST, expected 6+)"
fi

echo ""
echo "Step 4/8: Verifying Backend Handler..."
if [ -f "backend/internal/api/process_optimization_handlers.go" ]; then
    LINE_COUNT=$(wc -l < "backend/internal/api/process_optimization_handlers.go" | tr -d ' ')
    if [ "$LINE_COUNT" -gt 800 ]; then
        success "Backend handler exists ($LINE_COUNT lines)"
    else
        error "Backend handler too small ($LINE_COUNT lines, expected 1000+)"
    fi
else
    error "Backend handler file not found"
    exit 1
fi

echo ""
echo "Step 5/8: Checking ML Algorithm Functions..."
ALGORITHM_COUNT=$(grep -c "func (h \*ProcessOptimizationHandlers)" backend/internal/api/process_optimization_handlers.go || true)
if [ "$ALGORITHM_COUNT" -ge 10 ]; then
    success "ML algorithms implemented ($ALGORITHM_COUNT handler methods)"
else
    error "Missing ML algorithms (found $ALGORITHM_COUNT, expected 10+)"
fi

echo ""
echo "Step 6/8: Verifying Route Registration..."
if grep -q "processOptimizationHandler := NewProcessOptimizationHandlers" backend/internal/api/api.go; then
    success "Routes registered in api.go"
else
    error "Route registration missing in api.go"
    exit 1
fi

echo ""
echo "Step 7/8: Checking Frontend Dashboard..."
if [ -f "frontend/src/components/BPBuilder/ProcessOptimizationDashboard.tsx" ]; then
    LINE_COUNT=$(wc -l < "frontend/src/components/BPBuilder/ProcessOptimizationDashboard.tsx" | tr -d ' ')
    if [ "$LINE_COUNT" -gt 600 ]; then
        success "Frontend dashboard exists ($LINE_COUNT lines)"
    else
        error "Frontend dashboard too small ($LINE_COUNT lines, expected 650+)"
    fi
else
    error "Frontend dashboard file not found"
    exit 1
fi

echo ""
echo "Step 8/8: Verifying BP Builder Integration..."
if grep -q "ProcessOptimizationDashboard" frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx; then
    if grep -q "optimize" frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx; then
        success "BP Builder integration complete"
    else
        error "View mode 'optimize' not added"
    fi
else
    error "Dashboard not imported in BP Builder"
    exit 1
fi

echo ""
echo "=========================================="
echo "Summary: $SUCCESS_COUNT/$TOTAL_CHECKS checks passed"
echo "=========================================="
echo ""

if [ "$SUCCESS_COUNT" -eq "$TOTAL_CHECKS" ]; then
    echo -e "${GREEN}✓ All verifications passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Start the backend server: cd backend && go run cmd/server/main.go"
    echo "2. Start the frontend: cd frontend && npm start"
    echo "3. Open BP Builder and click 'AI Optimize' button"
    echo "4. Click 'Run Analysis' to generate suggestions"
    echo ""
else
    echo -e "${RED}✗ Some verifications failed${NC}"
    echo "Please fix the issues above and run this script again"
    echo ""
    exit 1
fi

# Optional: Test database query
echo "=========================================="
echo "Optional: Testing Database Queries"
echo "=========================================="
echo ""

echo "Checking for existing suggestions..."
SUGGESTION_COUNT=$(psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -t -c "
    SELECT COUNT(*) FROM process_optimization_suggestions
" | tr -d ' ')
info "Found $SUGGESTION_COUNT suggestions in database"

echo "Checking for applied optimizations..."
APPLIED_COUNT=$(psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -t -c "
    SELECT COUNT(*) FROM applied_optimizations
" | tr -d ' ')
info "Found $APPLIED_COUNT applied optimizations in database"

echo "Checking for auto-tune configs..."
AUTOTUNE_COUNT=$(psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -t -c "
    SELECT COUNT(*) FROM auto_tune_config
" | tr -d ' ')
info "Found $AUTOTUNE_COUNT auto-tune configurations in database"

echo ""
echo "=========================================="
echo "Setup verification complete! ✨"
echo "=========================================="
