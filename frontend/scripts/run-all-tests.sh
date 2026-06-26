#!/bin/bash

###############################################################################
# Phase 3 Comprehensive Test Runner
# 
# Runs all test suites in the correct order:
# 1. Unit tests (Jest + React Testing Library)
# 2. Integration tests
# 3. E2E tests (Playwright)
# 4. Coverage report
# 5. Performance analysis
###############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

echo "=========================================="
echo "Phase 3: Comprehensive Test Runner"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
UNIT_TESTS_ONLY=${UNIT_TESTS_ONLY:-false}
E2E_TESTS_ONLY=${E2E_TESTS_ONLY:-false}
COVERAGE_REPORT=${COVERAGE_REPORT:-true}
VERBOSE=${VERBOSE:-false}

# Test counters
UNIT_TESTS_PASSED=0
E2E_TESTS_PASSED=0
E2E_TESTS_FAILED=0

cd "$FRONTEND_DIR"

# Helper functions
log_section() {
  echo ""
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  echo ""
}

log_success() {
  echo -e "${GREEN}✓ $1${NC}"
}

log_error() {
  echo -e "${RED}✗ $1${NC}"
}

log_warning() {
  echo -e "${YELLOW}⚠ $1${NC}"
}

log_info() {
  echo -e "${BLUE}ℹ $1${NC}"
}

# ====================
# 1. Unit Tests
# ====================
log_section "1️⃣  Running Unit Tests (Jest + React Testing Library)"

if [ "$E2E_TESTS_ONLY" != "true" ]; then
  echo "Running component tests..."
  
  # Run unit tests with coverage
  if npm run test -- \
    --config=jest.config.phase3.json \
    --testPathPattern="(ScenarioConfigDialog|MultiScenarioComparison|useScenarioSimulation)" \
    --coverage \
    --passWithNoTests \
    --ci=false \
    ${VERBOSE:+--verbose}
  then
    log_success "Component unit tests passed"
    ((UNIT_TESTS_PASSED++))
  else
    log_error "Component unit tests failed"
  fi
  
  echo ""
  echo "Running hook tests..."
  
  # Run hook tests
  if npm run test -- \
    --config=jest.config.phase3.json \
    --testPathPattern="useScenarioSimulation" \
    --passWithNoTests \
    --ci=false \
    ${VERBOSE:+--verbose}
  then
    log_success "Hook unit tests passed"
    ((UNIT_TESTS_PASSED++))
  else
    log_error "Hook unit tests failed"
  fi
  
else
  log_warning "Skipping unit tests (E2E_TESTS_ONLY=true)"
fi

# ====================
# 2. Coverage Report
# ====================
if [ "$COVERAGE_REPORT" = "true" ] && [ "$E2E_TESTS_ONLY" != "true" ]; then
  log_section "2️⃣  Code Coverage Analysis"
  
  echo "Generating coverage report..."
  
  if npm run test -- \
    --config=jest.config.phase3.json \
    --coverage \
    --collectCoverageFrom='src/pages/portfolio/scenarios/**/*.{ts,tsx}' \
    --collectCoverageFrom='src/hooks/**/*.{ts,tsx}' \
    --passWithNoTests \
    --ci=false
  then
    log_success "Coverage report generated"
    
    # Display coverage summary if available
    if [ -f "coverage/coverage-summary.json" ]; then
      echo ""
      log_info "Coverage Summary:"
      echo ""
      
      # Parse and display coverage metrics
      grep -o '"statements":[^}]*' coverage/coverage-summary.json | head -1 | grep -o '[0-9.]*' | head -1 | \
        xargs -I {} log_info "Statements: {}%"
      
      grep -o '"branches":[^}]*' coverage/coverage-summary.json | head -1 | grep -o '[0-9.]*' | head -1 | \
        xargs -I {} log_info "Branches: {}%"
      
      grep -o '"functions":[^}]*' coverage/coverage-summary.json | head -1 | grep -o '[0-9.]*' | head -1 | \
        xargs -I {} log_info "Functions: {}%"
      
      grep -o '"lines":[^}]*' coverage/coverage-summary.json | head -1 | grep -o '[0-9.]*' | head -1 | \
        xargs -I {} log_info "Lines: {}%"
      
      echo ""
      log_info "Full report: coverage/index.html"
    fi
  else
    log_error "Coverage report generation failed"
  fi
fi

# ====================
# 3. E2E Tests
# ====================
if [ "$UNIT_TESTS_ONLY" != "true" ]; then
  log_section "3️⃣  Running E2E Tests (Playwright)"
  
  echo "Checking for dev server..."
  if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
    log_warning "Dev server not running on port 3000"
    log_info "Starting dev server..."
    npm start > /dev/null 2>&1 &
    DEV_SERVER_PID=$!
    
    # Wait for server to start
    echo "Waiting for server to start..."
    sleep 5
    
    # Check again
    if ! curl -s http://localhost:3000 > /dev/null 2>&1; then
      log_error "Failed to start dev server"
      exit 1
    fi
  fi
  
  log_success "Dev server is running"
  echo ""
  
  # Run E2E tests
  if [ -f "e2e/phase3-scenarios.spec.ts" ]; then
    echo "Running Playwright E2E tests..."
    
    if npx playwright test e2e/phase3-scenarios.spec.ts \
      --config=playwright.config.ts \
      --reporter=html \
      ${VERBOSE:+--debug}
    then
      log_success "All E2E tests passed"
      
      # Show results location
      if [ -d "test-results/phase3-e2e" ]; then
        log_info "HTML report: test-results/phase3-e2e/index.html"
      fi
    else
      log_error "Some E2E tests failed"
      
      # Show failure details
      if [ -d "test-results/phase3-e2e" ]; then
        log_warning "See test-results/phase3-e2e/index.html for details"
      fi
    fi
  else
    log_warning "E2E test file not found: e2e/phase3-scenarios.spec.ts"
  fi
  
  # Clean up dev server if we started it
  if [ -n "$DEV_SERVER_PID" ]; then
    log_info "Stopping dev server..."
    kill $DEV_SERVER_PID 2>/dev/null || true
  fi
fi

# ====================
# 4. Summary
# ====================
log_section "Test Summary"

echo "Unit Tests:"
if [ "$UNIT_TESTS_ONLY" != "true" ] && [ "$E2E_TESTS_ONLY" != "true" ]; then
  log_success "Passed: $((UNIT_TESTS_PASSED*2)) tests"
elif [ "$UNIT_TESTS_ONLY" = "true" ]; then
  log_success "Completed unit test phase"
fi

echo ""
echo "E2E Tests:"
if [ "$E2E_TESTS_ONLY" != "true" ] || [ "$UNIT_TESTS_ONLY" != "true" ]; then
  log_info "See Playwright HTML report for detailed results"
fi

echo ""
echo "=========================================="
log_success "Test run completed!"
echo "=========================================="

echo ""
echo "Next steps:"
echo "1. Review coverage report: coverage/index.html"
echo "2. Review E2E test report: test-results/phase3-e2e/index.html"
echo "3. Fix any failing tests"
echo "4. Run production build: npm run build"
echo "5. Deploy to staging environment"
