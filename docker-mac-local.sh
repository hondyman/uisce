#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Title
echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║        SemLayer - Local Docker Compose Deployment             ║"
echo "║         macOS with External Dependencies Support              ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker Desktop and try again.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker is running${NC}"
echo ""

# Parse arguments
ACTION="${1:-up}"
DETACH="${2:--d}"

if [ "$ACTION" == "down" ]; then
    echo -e "${YELLOW}Stopping all services...${NC}"
    docker compose -f docker-compose.mac-local.yml down
    echo -e "${GREEN}✅ Services stopped${NC}"
    exit 0
elif [ "$ACTION" == "logs" ]; then
    SERVICE="${2:-all}"
    if [ "$SERVICE" == "all" ]; then
        docker compose -f docker-compose.mac-local.yml logs -f
    else
        docker compose -f docker-compose.mac-local.yml logs -f "$SERVICE"
    fi
    exit 0
elif [ "$ACTION" == "restart" ]; then
    echo -e "${YELLOW}Restarting services...${NC}"
    docker compose -f docker-compose.mac-local.yml restart
    echo -e "${GREEN}✅ Services restarted${NC}"
    exit 0
fi

# Build and start services
echo -e "${YELLOW}Building Docker images...${NC}"
docker compose -f docker-compose.mac-local.yml build

echo ""
echo -e "${YELLOW}Starting services (in background)...${NC}"
docker compose -f docker-compose.mac-local.yml up $DETACH

if [ "$DETACH" == "-d" ]; then
    sleep 3
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}✅ All services started successfully!${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${YELLOW}📍 Service Endpoints:${NC}"
    echo -e "  Frontend:        ${GREEN}http://localhost:5173${NC}"
    echo -e "  Alternative:     ${GREEN}http://localhost:5174${NC}"
    echo -e "  Backend API:     ${GREEN}http://localhost:8080${NC}"
    echo -e "  Auth Service:    ${GREEN}http://localhost:8001${NC}"
    echo -e "  GraphQL (ext):   ${GREEN}http://100.84.126.19:8080/v1/graphql${NC}"
    echo ""
    echo -e "${YELLOW}🔐 Login Credentials:${NC}"
    echo -e "  Email:           ${GREEN}test@example.com${NC}"
    echo -e "  Password:        ${GREEN}password123${NC}"
    echo -e "  Role:            ${GREEN}global_ops${NC}"
    echo ""
    echo -e "${YELLOW}📊 Database:${NC}"
    echo -e "  Host:            ${GREEN}100.84.126.19:5432${NC}"
    echo -e "  Database:        ${GREEN}alpha${NC}"
    echo -e "  User:            ${GREEN}postgres${NC}"
    echo ""
    echo -e "${YELLOW}📎 Useful Commands:${NC}"
    echo -e "  View logs:       ${GREEN}./docker-mac-local.sh logs${NC}"
    echo -e "  View backend:    ${GREEN}./docker-mac-local.sh logs backend${NC}"
    echo -e "  View frontend:   ${GREEN}./docker-mac-local.sh logs frontend${NC}"
    echo -e "  View auth:       ${GREEN}./docker-mac-local.sh logs auth-service${NC}"
    echo -e "  Restart:         ${GREEN}./docker-mac-local.sh restart${NC}"
    echo -e "  Stop all:        ${GREEN}./docker-mac-local.sh down${NC}"
    echo ""
    echo -e "${YELLOW}🚀 Next Steps:${NC}"
    echo -e "  1. Open browser to ${GREEN}http://localhost:5173${NC}"
    echo -e "  2. Login with credentials above"
    echo -e "  3. Navigate to Glossary to manage semantic terms"
    echo -e "  4. Choose uisce tenant + northwinds datasource"
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════════${NC}"
fi
