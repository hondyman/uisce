#!/bin/bash

# Docker Compose Startup Script for Report Builder Backend
# This script sets up and runs all services for the AI Trade Reconciliation system
# with Phase 2/3 improvements (transactions, caching, audit logging, metrics)

set -e

PROJECT_NAME="atr"
COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    print_success "Docker found: $(docker --version)"
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed"
        exit 1
    fi
    print_success "Docker Compose found: $(docker-compose --version)"
}

# Setup environment file
setup_env_file() {
    print_header "Setting Up Environment"
    
    if [ ! -f "$ENV_FILE" ]; then
        print_warning "No .env file found, creating default"
        cat > "$ENV_FILE" << 'EOF'
# AI Trade Reconciliation Backend Environment Configuration
# Phase 2/3: Core Improvements + Advanced Features

# Database
DB_HOST=atr-db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=alpha
DATABASE_URL=postgres://postgres:postgres@atr-db:5432/alpha?sslmode=disable

# Temporal
TEMPORAL_HOST=temporal
TEMPORAL_PORT=7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=reconciliation

# API
PORT=8080
GIN_MODE=debug
LOG_LEVEL=info

# Phase 3 Features
CACHE_TTL=300s
CACHE_ENABLED=true
AUDIT_ENABLED=true
AUDIT_QUEUE_SIZE=1000
METRICS_ENABLED=true

# API Keys (optional)
XAI_API_KEY=
EOF
        print_success "Created .env file"
    else
        print_success ".env file already exists"
    fi
}

# Build images
build_images() {
    print_header "Building Docker Images"
    
    docker-compose -p "$PROJECT_NAME" build --no-cache
    print_success "Images built successfully"
}

# Start services
start_services() {
    print_header "Starting Services"
    
    print_info "Pulling latest images..."
    docker-compose -p "$PROJECT_NAME" pull
    
    print_info "Starting database service..."
    docker-compose -p "$PROJECT_NAME" up -d atr-db
    
    print_info "Waiting for database to be healthy..."
    sleep 10
    
    print_info "Starting Temporal service..."
    docker-compose -p "$PROJECT_NAME" up -d temporal
    docker-compose -p "$PROJECT_NAME" up -d temporal-ui
    
    print_info "Starting backend service..."
    docker-compose -p "$PROJECT_NAME" up -d atr-backend
    
    print_info "Starting frontend service..."
    docker-compose -p "$PROJECT_NAME" up -d atr-frontend
    
    print_success "Services started"
}

# Check service health
check_health() {
    print_header "Checking Service Health"
    
    print_info "Waiting for services to be healthy..."
    sleep 15
    
    # Check database
    if docker-compose -p "$PROJECT_NAME" exec -T atr-db pg_isready -U postgres > /dev/null 2>&1; then
        print_success "PostgreSQL database is healthy"
    else
        print_error "PostgreSQL database is not responding"
        return 1
    fi
    
    # Check backend
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        print_success "Backend API is healthy"
    else
        print_warning "Backend API not yet responding (may still be starting)"
    fi
    
    # Check frontend
    if curl -s http://localhost:3000 > /dev/null 2>&1; then
        print_success "Frontend is running"
    else
        print_warning "Frontend not yet responding (may still be starting)"
    fi
}

# Display service URLs
show_urls() {
    print_header "Service URLs"
    
    echo ""
    echo -e "${GREEN}API & Services:${NC}"
    echo "  Backend API:          http://localhost:8080"
    echo "  Frontend:             http://localhost:3000"
    echo "  Temporal UI:          http://localhost:8081"
    echo "  Database:             postgres://postgres:postgres@localhost:5432/alpha"
    echo ""
    echo -e "${GREEN}Monitoring (Optional - run with: docker-compose --profile monitoring up):${NC}"
    echo "  Prometheus:           http://localhost:9091"
    echo "  Grafana:              http://localhost:3001"
    echo ""
}

# Show logs
show_logs() {
    print_header "Service Logs"
    
    echo ""
    echo "To view logs for all services:"
    echo "  docker-compose -p $PROJECT_NAME logs -f"
    echo ""
    echo "To view logs for specific service:"
    echo "  docker-compose -p $PROJECT_NAME logs -f atr-backend"
    echo "  docker-compose -p $PROJECT_NAME logs -f atr-database"
    echo ""
}

# Show stop instructions
show_stop_instructions() {
    echo ""
    echo -e "${YELLOW}To stop services:${NC}"
    echo "  docker-compose -p $PROJECT_NAME down"
    echo ""
    echo -e "${YELLOW}To stop and remove volumes (careful - removes data!):${NC}"
    echo "  docker-compose -p $PROJECT_NAME down -v"
    echo ""
}

# Main execution
main() {
    echo ""
    print_header "AI Trade Reconciliation Backend - Docker Compose Setup"
    echo ""
    print_info "Phase 2: Core Improvements (6/6 tasks complete)"
    print_info "Phase 3: Advanced Features (5/5 features complete)"
    echo ""
    
    # Run setup steps
    check_prerequisites
    echo ""
    
    setup_env_file
    echo ""
    
    read -p "Build Docker images? (y/n, default: y) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        build_images
        echo ""
    fi
    
    start_services
    echo ""
    
    check_health
    echo ""
    
    show_urls
    show_logs
    show_stop_instructions
    
    print_header "Setup Complete!"
    print_success "Your backend is running with Phase 2/3 features:"
    print_success "✓ Core Improvements: Error handling, validation, type mapping"
    print_success "✓ Advanced Features: Transactions, caching, batch ops, audit, metrics"
    echo ""
}

# Run main function
main "$@"
