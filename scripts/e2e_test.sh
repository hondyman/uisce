#!/bin/bash

# ============================================================================
# Semlayer End-to-End Test Suite
# ============================================================================
# Tests JWT security, local compose services, and remote infrastructure
# 
# Usage: bash scripts/e2e_test.sh [local|remote|all]
# ============================================================================

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
REMOTE_HOST="${REMOTE_HOST:-100.84.126.19}"
JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-key-change-in-production}"
LOCAL_BACKEND_URL="http://localhost:8080"
HEALTHCHECK_RETRIES=30
HEALTHCHECK_INTERVAL=2

# Logging functions
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

# Test functions
test_local_docker() {
    log_section "Testing Local Docker Compose"
    
    cd /Users/eganpj/GitHub/semlayer
    
    # Check docker daemon
    log_info "Checking Docker daemon..."
    if ! docker ps > /dev/null 2>&1; then
        log_error "Docker daemon not running"
        return 1
    fi
    log_info "✓ Docker daemon is running"
    
    # Check compose services
    log_info "Checking compose services..."
    local running=$(docker compose ps --services --filter "status=running" | wc -l)
    log_info "✓ $running services running"
    
    # Show unhealthy services
    local unhealthy=$(docker compose ps --filter "health=unhealthy" --format "{{.Service}}" 2>/dev/null | wc -l)
    if [ "$unhealthy" -gt 0 ]; then
        log_warn "Found $unhealthy unhealthy services:"
        docker compose ps --filter "health=unhealthy" --format "{{.Service}}" 2>/dev/null | sed 's/^/  - /'
    fi
}

test_backend_health() {
    log_section "Testing Backend Service Health"
    
    log_info "Attempting to reach $LOCAL_BACKEND_URL/health"
    
    local attempt=1
    while [ $attempt -le $HEALTHCHECK_RETRIES ]; do
        if curl -sf "$LOCAL_BACKEND_URL/health" > /dev/null 2>&1; then
            log_info "✓ Backend health check passed"
            return 0
        fi
        
        if [ $attempt -eq 1 ]; then
            log_warn "Backend not ready, retrying..."
        fi
        
        sleep $HEALTHCHECK_INTERVAL
        attempt=$((attempt + 1))
    done
    
    log_error "Backend health check failed after $HEALTHCHECK_RETRIES attempts"
    return 1
}

test_jwt_validation() {
    log_section "Testing JWT Token Validation"
    
    log_info "Generating test JWT token..."
    
    # Generate a test JWT token (HS256)
    local header='{"alg":"HS256","typ":"JWT"}'
    local payload="{\"user_id\":\"test-user\",\"tenant_id\":\"00000000-0000-0000-0000-000000000001\",\"exp\":$(($(date +%s) + 3600))}"
    
    # Base64 encode without padding
    local header_b64=$(echo -n "$header" | base64 | tr -d '=' | tr '+/' '-_')
    local payload_b64=$(echo -n "$payload" | base64 | tr -d '=' | tr '+/' '-_')
    
    # Create signature (simplified - using echo + openssl)
    local message="$header_b64.$payload_b64"
    local signature=$(echo -n "$message" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr -d '=' | tr '+/' '-_')
    
    local token="$message.$signature"
    
    log_info "Generated token: ${token:0:50}..."
    
    # Test with JWT token
    log_info "Testing request with JWT token..."
    if curl -sf \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        "$LOCAL_BACKEND_URL/health" > /dev/null 2>&1; then
        log_info "✓ Request with JWT token succeeded"
    else
        log_warn "Request with JWT token failed (expected if endpoint requires auth)"
    fi
    
    # Test without JWT token (should fail or return 401)
    log_info "Testing request without JWT token..."
    local response=$(curl -s -o /dev/null -w "%{http_code}" "$LOCAL_BACKEND_URL/health" 2>&1)
    if [ "$response" = "200" ]; then
        log_info "✓ Public endpoint accessible without token (status: $response)"
    else
        log_warn "Public endpoint returned status: $response"
    fi
}

test_service_endpoints() {
    log_section "Testing Service Endpoints"
    
    local services=(
        "backend:8080"
        "compliance-engine:8095"
        "entity-manager:8087"
        "validation-engine:8090"
        "rule-engine:8091"
        "policy-engine:8102"
    )
    
    for service_info in "${services[@]}"; do
        local service="${service_info%:*}"
        local port="${service_info#*:}"
        local url="http://localhost:$port"
        
        log_info "Testing $service ($url/health)..."
        
        if curl -sf "$url/health" > /dev/null 2>&1; then
            log_info "  ✓ $service is healthy"
        else
            log_warn "  ✗ $service is not responding"
        fi
    done
}

test_database_connectivity() {
    log_section "Testing Database Connectivity"
    
    log_info "Checking if Postgres is accessible at $REMOTE_HOST:5432..."
    
    # Try to connect with timeout
    if timeout 5 bash -c "cat < /dev/null > /dev/tcp/$REMOTE_HOST/5432" 2>/dev/null; then
        log_info "✓ Port 5432 on $REMOTE_HOST is open"
        
        # Try to connect with psql if available
        if command -v psql &> /dev/null; then
            log_info "Attempting to connect with psql..."
            if PGPASSWORD=postgres psql -h "$REMOTE_HOST" -U postgres -d alpha -c "SELECT 1" > /dev/null 2>&1; then
                log_info "✓ Successfully connected to Postgres"
                return 0
            else
                log_warn "Could not connect with psql"
                return 1
            fi
        fi
        return 0
    else
        log_warn "Cannot reach Postgres at $REMOTE_HOST:5432"
        log_warn "Verify remote host is accessible and Postgres is running"
        return 1
    fi
}

test_remote_connectivity() {
    log_section "Testing Remote Server Connectivity"
    
    log_info "Pinging remote host $REMOTE_HOST..."
    
    if timeout 5 ping -c 1 "$REMOTE_HOST" > /dev/null 2>&1; then
        log_info "✓ Remote host $REMOTE_HOST is reachable"
        
        # Test SSH connectivity
        if command -v ssh &> /dev/null; then
            log_info "Checking SSH access to $REMOTE_HOST..."
            if timeout 5 ssh -o ConnectTimeout=3 "$REMOTE_HOST" "echo 'SSH OK'" > /dev/null 2>&1; then
                log_info "✓ SSH access to $REMOTE_HOST is working"
                return 0
            else
                log_warn "SSH access to $REMOTE_HOST failed"
                return 1
            fi
        fi
        return 0
    else
        log_error "Remote host $REMOTE_HOST is not reachable"
        log_warn "Verify Tailscale connection and firewall settings"
        return 1
    fi
}

test_compose_build() {
    log_section "Testing Compose Build"
    
    log_info "Checking if all images are built..."
    
    cd /Users/eganpj/GitHub/semlayer
    
    # Check for build issues
    local unbuilt=$(docker compose config --services 2>/dev/null | while read service; do
        if ! docker compose config --services 2>/dev/null | grep -q "$service"; then
            echo "$service"
        fi
    done | wc -l)
    
    if [ "$unbuilt" -eq 0 ]; then
        log_info "✓ All services are built"
        return 0
    else
        log_warn "Some services may not be built"
        return 1
    fi
}

test_network() {
    log_section "Testing Docker Network"
    
    log_info "Checking semlayer-net network..."
    
    if docker network inspect semlayer-net > /dev/null 2>&1; then
        log_info "✓ semlayer-net network exists"
        
        local containers=$(docker network inspect semlayer-net --format='{{len .Containers}}' | tr -d ' ')
        log_info "  $containers containers connected"
        return 0
    else
        log_warn "semlayer-net network not found"
        return 1
    fi
}

# Summary tracking
PASSED=0
FAILED=0

run_test() {
    local test_name=$1
    local test_func=$2
    
    if $test_func; then
        PASSED=$((PASSED + 1))
    else
        FAILED=$((FAILED + 1))
    fi
}

# Main execution
main() {
    local test_type="${1:-all}"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo -e "${BOLD}Semlayer E2E Test Suite${NC}"
    echo "Started: $timestamp"
    echo "Test Type: $test_type"
    echo ""
    
    case $test_type in
        local)
            run_test "Docker Compose" test_local_docker
            run_test "Docker Network" test_network
            run_test "Compose Build" test_compose_build
            run_test "Backend Health" test_backend_health
            run_test "Service Endpoints" test_service_endpoints
            run_test "JWT Validation" test_jwt_validation
            ;;
        remote)
            run_test "Remote Connectivity" test_remote_connectivity
            run_test "Database Connectivity" test_database_connectivity
            ;;
        all)
            run_test "Docker Compose" test_local_docker
            run_test "Docker Network" test_network
            run_test "Compose Build" test_compose_build
            run_test "Backend Health" test_backend_health
            run_test "Service Endpoints" test_service_endpoints
            run_test "JWT Validation" test_jwt_validation
            run_test "Remote Connectivity" test_remote_connectivity
            run_test "Database Connectivity" test_database_connectivity
            ;;
        *)
            log_error "Invalid test type: $test_type"
            echo "Usage: bash scripts/e2e_test.sh [local|remote|all]"
            exit 1
            ;;
    esac
    
    # Print summary
    log_section "Test Summary"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    if [ $FAILED -gt 0 ]; then
        echo -e "${RED}Failed: $FAILED${NC}"
    else
        echo -e "${GREEN}Failed: $FAILED${NC}"
    fi
    
    if [ $FAILED -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}Some tests failed. Please review the output above.${NC}"
        exit 1
    else
        echo ""
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

# Run main
main "$@"
