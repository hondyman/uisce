#!/bin/bash

# ============================================================================
# Local Environment Fix Script
# ============================================================================
#
# Fixes common local development issues:
# - Missing .env file
# - Unhealthy services
# - Database connection issues
#
# Usage: bash scripts/fix_local_env.sh
# ============================================================================

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${BOLD}=== $1 ===${NC}"
}

cd /Users/eganpj/GitHub/semlayer

# Fix 1: Create .env file if missing
fix_env_file() {
    log_section "Fixing .env Configuration"
    
    if [ ! -f ".env" ]; then
        log_warn "No .env file found"
        log_info "Checking available templates..."
        
        if [ -f ".env.split" ]; then
            log_info "Copying .env.split to .env"
            cp .env.split .env
            log_info "✓ .env file created"
        elif [ -f ".env.example" ]; then
            log_info "Copying .env.example to .env"
            cp .env.example .env
            log_info "⚠️ .env file created (edit with your credentials)"
        else
            log_error "No env template found"
            return 1
        fi
    else
        log_info "✓ .env file already exists"
    fi
}

# Fix 2: Restart unhealthy services
fix_unhealthy_services() {
    log_section "Restarting Unhealthy Services"
    
    local services=("auth-service" "event-router" "notifications" "search-service" "bp-backend")
    
    for service in "${services[@]}"; do
        log_info "Restarting $service..."
        docker compose restart "$service" || {
            log_warn "Failed to restart $service"
        }
    done
    
    log_info "✓ Restart commands sent"
    log_info "Waiting 10 seconds for services to start..."
    sleep 10
}

# Fix 3: Check service health
check_health() {
    log_section "Checking Service Health"
    
    local healthy=0
    local unhealthy=0
    
    docker compose ps --format "table {{.Service}}\t{{.Status}}" | while read service status; do
        if echo "$status" | grep -q "healthy"; then
            log_info "✓ $service: healthy"
            healthy=$((healthy + 1))
        elif echo "$status" | grep -q "Up"; then
            log_info "⚠️ $service: up"
        else
            log_warn "✗ $service: $status"
            unhealthy=$((unhealthy + 1))
        fi
    done
}

# Fix 4: Verify network
verify_network() {
    log_section "Verifying Docker Network"
    
    if docker network inspect semlayer-net > /dev/null 2>&1; then
        log_info "✓ semlayer-net network exists"
        
        local containers=$(docker network inspect semlayer-net --format='{{len .Containers}}')
        log_info "  $containers containers connected"
    else
        log_error "semlayer-net network not found"
        log_info "Recreating network..."
        docker network create semlayer-net || log_warn "Network may already exist"
    fi
}

# Fix 5: Run E2E tests
run_tests() {
    log_section "Running E2E Tests"
    
    if [ -f "scripts/e2e_test.sh" ]; then
        bash scripts/e2e_test.sh local
    else
        log_warn "E2E test script not found"
    fi
}

# Main execution
main() {
    echo -e "${BOLD}Semlayer Local Environment Fix${NC}"
    echo ""
    
    fix_env_file
    verify_network
    fix_unhealthy_services
    check_health
    
    log_section "Completed"
    log_info "Local environment fix complete"
    log_info "Recommended next steps:"
    echo "  1. Check service logs: docker compose logs -f [service-name]"
    echo "  2. Run full tests: bash scripts/e2e_test.sh all"
    echo "  3. Verify remote server is online: ssh 100.84.126.19"
}

main "$@"
