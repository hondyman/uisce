#!/bin/bash

# Rebalancing System - Docker Startup Script
# Automates the complete deployment process

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
LOG_FILE="$SCRIPT_DIR/docker-startup.log"

# Functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Step 1: Check Prerequisites
print_header "Step 1/7: Checking Prerequisites"

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi
print_success "Docker installed: $(docker --version)"

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed"
    exit 1
fi
print_success "Docker Compose installed: $(docker-compose --version)"

# Step 2: Setup Environment
print_header "Step 2/7: Setting up Environment"

if [ ! -f "$ENV_FILE" ]; then
    print_warning ".env file not found, creating from .env.example"
    if [ -f "$SCRIPT_DIR/.env.example" ]; then
        cp "$SCRIPT_DIR/.env.example" "$ENV_FILE"
        print_success "Created .env file"
        print_info "Review and update .env with your API keys and secrets"
    else
        print_error ".env.example not found"
        exit 1
    fi
else
    print_success ".env file exists"
fi

# Step 3: Clean Up Previous Containers (Optional)
print_header "Step 3/7: Preparing Docker Environment"

if [ "$1" == "--clean" ]; then
    print_warning "Removing existing containers and images..."
    docker-compose down -v --rmi local 2>/dev/null || true
    print_success "Clean up complete"
else
    print_info "Skipping clean (use --clean flag to remove existing containers)"
fi

# Step 4: Build Images
print_header "Step 4/7: Building Docker Images"

print_info "Building frontend image..."
docker-compose build rebalance-frontend 2>&1 | tee -a "$LOG_FILE"
print_success "Frontend image built"

print_info "Building API image..."
docker-compose build rebalance-api 2>&1 | tee -a "$LOG_FILE"
print_success "API image built"

print_info "Building worker image..."
docker-compose build rebalance-worker 2>&1 | tee -a "$LOG_FILE"
print_success "Worker image built"

# Step 5: Start Services
print_header "Step 5/7: Starting Docker Services"

print_info "Starting services (this may take 2-3 minutes)..."
docker-compose up -d 2>&1 | tee -a "$LOG_FILE"
print_success "Docker services started"

# Step 6: Wait for Services
print_header "Step 6/7: Waiting for Services to be Healthy"

TIMEOUT=180
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $TIMEOUT ]; do
    ALL_HEALTHY=true
    
    echo -n "."
    
    # Check PostgreSQL
    if ! docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
        ALL_HEALTHY=false
    fi
    
    # Check Temporal
    if ! docker-compose exec -T temporal curl -s http://localhost:7233/ > /dev/null 2>&1; then
        ALL_HEALTHY=false
    fi
    
    # Check Hasura
    if ! docker-compose exec -T hasura curl -s http://localhost:8080/v1/metadata > /dev/null 2>&1; then
        ALL_HEALTHY=false
    fi
    
    if [ "$ALL_HEALTHY" = true ]; then
        echo ""
        print_success "All services are healthy"
        break
    fi
    
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    print_warning "Services took longer than expected to start"
    print_info "Checking service status..."
fi

# Step 7: Verify Deployment
print_header "Step 7/7: Verifying Deployment"

echo ""
echo "Service Status:"
docker-compose ps

echo ""
echo "Health Checks:"

SERVICES=("postgres" "temporal" "redpanda" "hasura" "redis" "rebalance-api" "rebalance-frontend")

for service in "${SERVICES[@]}"; do
    if docker-compose ps "$service" | grep -q "healthy\|Up"; then
        print_success "$service"
    else
        print_warning "$service (still starting)"
    fi
done

# Final Summary
print_header "🎉 Deployment Complete!"

echo ""
echo "Access your services at:"
echo ""
echo -e "  ${GREEN}React Dashboard:${NC}        http://localhost:3000"
echo -e "  ${GREEN}Hasura Console:${NC}         http://localhost:8080"
echo -e "  ${GREEN}Temporal UI:${NC}            http://localhost:8081"
echo -e "  ${GREEN}Redpanda (Pandaproxy):${NC}  http://localhost:8082"
echo -e "  ${GREEN}API Server:${NC}             http://localhost:8090"
echo -e "  ${GREEN}GraphQL Endpoint:${NC}       http://localhost:8080/v1/graphql"
echo ""

echo "Credentials:"
echo ""
echo -e "  ${GREEN}Hasura Admin Secret:${NC}    $(grep HASURA_SECRET "$ENV_FILE" | cut -d= -f2)"
echo -e "  ${GREEN}Redpanda (Kafka):${NC}        brokers at localhost:9092 (Pandaproxy: http://localhost:8082)"
echo -e "  ${GREEN}PostgreSQL:${NC}              postgres / postgres"
echo ""

echo "Useful Commands:"
echo ""
echo -e "  ${BLUE}View logs:${NC}"
echo "    docker-compose logs -f"
echo ""
echo -e "  ${BLUE}Access PostgreSQL:${NC}"
echo "    docker-compose exec postgres psql -U postgres -d portfolio"
echo ""
echo -e "  ${BLUE}Stop services:${NC}"
echo "    docker-compose down"
echo ""
echo -e "  ${BLUE}View service status:${NC}"
echo "    docker-compose ps"
echo ""

echo "Documentation:"
echo ""
echo -e "  ${GREEN}Full deployment guide:${NC}   DOCKER_DEPLOYMENT.md"
echo -e "  ${GREEN}Rebalancing guide:${NC}       REBALANCING_GUIDE.md"
echo -e "  ${GREEN}Quick reference:${NC}         REBALANCING_INDEX.md"
echo ""

print_success "System is ready to use!"
print_info "Next: Open http://localhost:3000 to access the dashboard"

# Tail logs option
if [ "$1" == "--logs" ] || [ "$2" == "--logs" ]; then
    print_header "Tailing Logs (Press Ctrl+C to exit)"
    docker-compose logs -f
fi
