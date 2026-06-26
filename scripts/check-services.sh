#!/bin/bash

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Semlayer Development Environment Status${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"

# Function to check service
check_service() {
    local service=$1
    local port=$2
    local name=$3
    
    if nc -z localhost $port 2>/dev/null; then
        echo -e "${GREEN}✓${NC} ${name} (${service}) - http://localhost:${port}"
        return 0
    else
        echo -e "${RED}✗${NC} ${name} (${service}) - Not responding on port ${port}"
        return 1
    fi
}

echo -e "${YELLOW}Infrastructure Services:${NC}"
check_service "hasura" "8888" "Hasura GraphQL"
check_service "rabbitmq" "5672" "RabbitMQ AMQP"
check_service "rabbitmq-mgmt" "15672" "RabbitMQ Management"
check_service "temporal" "7233" "Temporal Workflow"
check_service "temporal-ui" "8088" "Temporal UI"

echo -e "\n${YELLOW}Backend Services:${NC}"
check_service "backend" "8080" "Backend API"
check_service "api-gateway" "8001" "API Gateway"
check_service "fabric-builder" "8080" "Fabric Builder"

echo -e "\n${YELLOW}Frontend:${NC}"
check_service "frontend" "5173" "Frontend Dev Server"

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}Web Interfaces:${NC}"
echo -e "  • Hasura Console       : ${GREEN}http://localhost:8888${NC}"
echo -e "  • RabbitMQ Management  : ${GREEN}http://localhost:15672${NC} (guest/guest)"
echo -e "  • Temporal UI          : ${GREEN}http://localhost:8088${NC}"
echo -e "  • Frontend App         : ${GREEN}http://localhost:5173${NC}"
echo -e "  • Backend API          : ${GREEN}http://localhost:8080${NC}"

echo -e "\n${BLUE}API Endpoints:${NC}"
echo -e "  • GraphQL              : ${GREEN}http://localhost:8888/v1/graphql${NC}"
echo -e "  • Business Entities    : ${GREEN}http://localhost:8080/api/business-entities${NC}"
echo -e "  • Relationships        : ${GREEN}http://localhost:8080/api/relationships${NC}"

echo -e "\n${BLUE}Environment Variables:${NC}"
echo -e "  • Tenant ID            : ${YELLOW}910638ba-a459-4a3f-bb2d-78391b0595f6${NC}"
echo -e "  • Datasource ID        : ${YELLOW}982aef38-418f-46dc-acd0-35fe8f3b97b0${NC}"
echo -e "  • VITE_API_BASE_URL    : ${YELLOW}http://localhost:8080${NC}"
echo -e "  • VITE_GRAPHQL_ENDPOINT: ${YELLOW}http://localhost:8080/v1/graphql${NC}"

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}Quick Commands:${NC}"
echo -e "  • View logs            : ${YELLOW}docker compose -f docker-compose.dev.simple.yml logs -f [service]${NC}"
echo -e "  • Stop services        : ${YELLOW}docker compose -f docker-compose.dev.simple.yml down${NC}"
echo -e "  • Restart services     : ${YELLOW}docker compose -f docker-compose.dev.simple.yml restart${NC}"
echo -e "  • View containers      : ${YELLOW}docker compose -f docker-compose.dev.simple.yml ps${NC}"

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"
