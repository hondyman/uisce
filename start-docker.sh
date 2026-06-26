#!/bin/bash

# SemLayer Docker-Only Startup Script
# This script starts all SemLayer services in Docker Compose
# Prerequisites: PostgreSQL running on localhost:5432

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║       SemLayer - Docker Compose Startup Script            ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if PostgreSQL is accessible
echo -e "${YELLOW}[1/5]${NC} Checking PostgreSQL connection..."
if ! psql -h localhost -U postgres -d alpha -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${RED}✗ PostgreSQL not accessible at localhost:5432${NC}"
    echo -e "${YELLOW}Make sure PostgreSQL is running locally with:${NC}"
    echo -e "  ${BLUE}psql postgres://postgres:postgres@localhost:5432/alpha${NC}"
    exit 1
fi
echo -e "${GREEN}✓ PostgreSQL is accessible${NC}"
echo ""

# Create .env file with defaults if it doesn't exist
echo -e "${YELLOW}[2/5]${NC} Setting up environment configuration..."
if [ ! -f .env ]; then
    cat > .env << 'EOF'
# SemLayer Environment Configuration

# API Gateway
API_GATEWAY_HOST_PORT=8001

# Backend
BACKEND_HOST_PORT=8080

# Fabric Builder
FABRIC_HOST_PORT=8081

# Hasura
HASURA_ADMIN_SECRET=newadminsecretkey

# JWT
JWT_SECRET=your-jwt-secret-key

# XAI (Optional)
XAI_API_KEY=

# Revocation Store (Redis)
REVOCATION_REDIS_ADDR=

# Development flags
DEV_ALLOW_UNAUTH_FABRIC=true
DEV_ALLOW_UNAUTH_VIEWS=true
DEV_ALLOW_UNAUTH_MODELS=true
DEV_ALLOW_UNAUTH_CATALOG=true
DEV_ALLOW_UNAUTH_BUSINESS_TERM=true

# IP Whitelist
IP_WHITELIST_ENFORCE=false
EOF
    echo -e "${GREEN}✓ Created .env file${NC}"
else
    echo -e "${GREEN}✓ Using existing .env file${NC}"
fi
echo ""

# Stop any running containers
echo -e "${YELLOW}[3/5]${NC} Cleaning up existing containers..."
docker compose down --remove-orphans > /dev/null 2>&1 || true
echo -e "${GREEN}✓ Cleaned up${NC}"
echo ""

# Build images
echo -e "${YELLOW}[4/5]${NC} Building Docker images..."
docker compose build --progress=plain 2>&1 | grep -E "^(Building|Successfully|Step)" || true
echo -e "${GREEN}✓ Images built${NC}"
echo ""

# Start services
echo -e "${YELLOW}[5/5]${NC} Starting services..."
docker compose up -d

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║               ✓ All services started successfully!         ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Wait a bit for services to start
sleep 5

# Show service status
echo -e "${BLUE}Service Status:${NC}"
docker compose ps

echo ""
echo -e "${BLUE}Access Points:${NC}"
echo -e "  API Gateway:        ${GREEN}http://localhost:8001${NC}"
echo -e "  Hasura Console:     ${GREEN}http://localhost:8080${NC}"
echo -e "  Backend API:        ${GREEN}http://localhost:8080${NC}"
echo -e "  Fabric Builder:     ${GREEN}http://localhost:8081${NC}"
echo -e "  Temporal UI:        ${GREEN}http://localhost:8088${NC}"
echo -e "  RabbitMQ Console:   ${GREEN}http://localhost:15672${NC} (guest:guest)"
echo ""

echo -e "${BLUE}Health Checks:${NC}"
echo "  API Gateway:"
sleep 2
if curl -s http://localhost:8001/health > /dev/null; then
    echo -e "    ${GREEN}✓ Healthy${NC}"
else
    echo -e "    ${RED}✗ Not responding${NC}"
fi

echo "  Backend:"
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "    ${GREEN}✓ Healthy${NC}"
else
    echo -e "    ${RED}✗ Not responding${NC}"
fi

echo ""
echo -e "${BLUE}Useful Commands:${NC}"
echo -e "  View logs:          ${YELLOW}docker compose logs -f api-gateway${NC}"
echo -e "  Stop services:      ${YELLOW}docker compose down${NC}"
echo -e "  Restart service:    ${YELLOW}docker compose restart api-gateway${NC}"
echo -e "  View all services:  ${YELLOW}docker compose ps${NC}"
echo ""

echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Start the frontend: cd frontend && npm start"
echo "  2. Access the UI at: http://localhost:5173"
echo "  3. Set tenant context via the tenant picker in the UI"
echo ""
