#!/bin/bash

# Service Health Check Script
# Tests all major services in the SemLayer stack

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}        SEMLAYER SERVICE HEALTH CHECK${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Function to test a service
test_service() {
    local name=$1
    local port=$2
    local endpoint=${3:-/health}
    local expected_code=${4:-200}
    
    echo -n "Testing $name on :$port$endpoint... "
    
    if response=$(curl -s -w "\n%{http_code}" http://localhost:$port$endpoint 2>/dev/null); then
        http_code=$(echo "$response" | tail -n1)
        if [ "$http_code" = "$expected_code" ] || [ "$http_code" = "200" ] || [ "$http_code" = "000" ]; then
            echo -e "${GREEN}✓ OK (HTTP $http_code)${NC}"
            return 0
        else
            echo -e "${YELLOW}⚠ HTTP $http_code${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ FAILED (no response)${NC}"
        return 1
    fi
}

# Function to test container health
test_container() {
    local name=$1
    local container=$2
    
    echo -n "Checking container $container... "
    
    if status=$(docker compose ps $container 2>/dev/null | grep "$container"); then
        if echo "$status" | grep -q "Up"; then
            echo -e "${GREEN}✓ Running${NC}"
            return 0
        elif echo "$status" | grep -q "Restarting"; then
            echo -e "${YELLOW}⚠ Restarting${NC}"
            return 1
        else
            echo -e "${RED}✗ Not running${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Not found${NC}"
        return 1
    fi
}

echo -e "${BLUE}1. CONTAINER STATUS${NC}"
echo "─────────────────────────────────────────────────────────────"
test_container "API Gateway" "api-gateway"
test_container "Backend" "backend"
test_container "RabbitMQ" "rabbitmq"
test_container "Temporal" "temporal"
test_container "Frontend" "frontend"
echo ""

echo -e "${BLUE}2. HTTP ENDPOINTS${NC}"
echo "─────────────────────────────────────────────────────────────"
test_service "API Gateway Health" "8001" "/health"
test_service "API Gateway Debug" "8001" "/api/_debug/headers"
test_service "Backend Health" "8080" "/health"
test_service "Fabric Builder" "8081" "/health"
echo ""

echo -e "${BLUE}3. MESSAGE QUEUE (RabbitMQ)${NC}"
echo "─────────────────────────────────────────────────────────────"
echo -n "RabbitMQ Management UI... "
if curl -s -u guest:guest http://localhost:15672/api/aliveness-test/%2F 2>/dev/null | grep -q "ok"; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo -n "RabbitMQ TCP (5672)... "
if nc -z localhost 5672 2>/dev/null; then
    echo -e "${GREEN}✓ Listening${NC}"
else
    echo -e "${RED}✗ Not accessible${NC}"
fi
echo ""

echo -e "${BLUE}4. WORKFLOW ENGINE (Temporal)${NC}"
echo "─────────────────────────────────────────────────────────────"
echo -n "Temporal gRPC (7233)... "
if nc -z localhost 7233 2>/dev/null; then
    echo -e "${GREEN}✓ Listening${NC}"
else
    echo -e "${RED}✗ Not accessible (may be restarting)${NC}"
fi

echo -n "Temporal UI (8088)... "
if curl -s http://localhost:8088/api/cluster-info 2>/dev/null | grep -q "name"; then
    echo -e "${GREEN}✓ Working${NC}"
else
    echo -e "${YELLOW}⚠ Not ready yet${NC}"
fi
echo ""

echo -e "${BLUE}5. DATABASE CONNECTION${NC}"
echo "─────────────────────────────────────────────────────────────"
echo -n "PostgreSQL (localhost:5432)... "
if psql postgres://postgres:postgres@localhost:5432/alpha -c "SELECT 1" &>/dev/null; then
    echo -e "${GREEN}✓ Connected${NC}"
else
    echo -e "${RED}✗ Not accessible${NC}"
fi
echo ""

echo -e "${BLUE}6. MONITORING${NC}"
echo "─────────────────────────────────────────────────────────────"
test_service "Prometheus" "9090" "/-/healthy"
test_service "Grafana" "3000" "/api/health"
echo ""

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}SERVICE CHECK COMPLETE${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Note:${NC}"
echo "  • Temporal may need time to initialize"
echo "  • If they're restarting, check their logs for errors"
echo "  • RabbitMQ is critical for message queuing"
echo ""
