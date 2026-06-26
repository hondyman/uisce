#!/bin/bash
# =============================================================================
# TEST DISTRIBUTED PLATFORM CONNECTIVITY
# =============================================================================
# This script tests all connections needed for the distributed setup
#
# Usage: ./test-distributed-connectivity.sh
# =============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m'

REMOTE_HOST="${1:-100.84.126.19}"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo -e "${BLUE}=============================================================================${NC}"
echo -e "${BLUE}  DISTRIBUTED PLATFORM CONNECTIVITY TEST${NC}"
echo -e "${BLUE}=============================================================================${NC}"
echo ""
echo "Testing connectivity to remote host: ${REMOTE_HOST}"
echo ""

# Function to test port connectivity
test_port() {
    local service=$1
    local host=$2
    local port=$3
    
    echo -n "  Testing ${service} (${host}:${port})... "
    
    if timeout 3 bash -c "cat < /dev/null > /dev/tcp/${host}/${port}" 2>/dev/null; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        return 1
    fi
}

# Test Remote Services
echo -e "${YELLOW}=== REMOTE SERVICES (${REMOTE_HOST}) ===${NC}"
echo ""

PASS=0
FAIL=0

test_port "PostgreSQL" "${REMOTE_HOST}" 5432 && ((PASS++)) || ((FAIL++))
test_port "Hasura GraphQL" "${REMOTE_HOST}" 8085 && ((PASS++)) || ((FAIL++))
test_port "Redpanda Kafka (external)" "${REMOTE_HOST}" 19092 && ((PASS++)) || ((FAIL++))
test_port "Redpanda Admin" "${REMOTE_HOST}" 9644 && ((PASS++)) || ((FAIL++))
test_port "Schema Registry" "${REMOTE_HOST}" 8081 && ((PASS++)) || ((FAIL++))
test_port "Redpanda Console" "${REMOTE_HOST}" 8096 && ((PASS++)) || ((FAIL++))
test_port "Debezium" "${REMOTE_HOST}" 8083 && ((PASS++)) || ((FAIL++))
test_port "Temporal" "${REMOTE_HOST}" 7233 && ((PASS++)) || ((FAIL++))
test_port "Temporal UI" "${REMOTE_HOST}" 8088 && ((PASS++)) || ((FAIL++))
test_port "Trino" "${REMOTE_HOST}" 8094 && ((PASS++)) || ((FAIL++))
test_port "MinIO API" "${REMOTE_HOST}" 9010 && ((PASS++)) || ((FAIL++))
test_port "MinIO Console" "${REMOTE_HOST}" 9011 && ((PASS++)) || ((FAIL++))

echo ""

# Test Local Services
echo -e "${YELLOW}=== LOCAL SERVICES (localhost) ===${NC}"
echo ""

test_port "Backend API" "localhost" 8080 && ((PASS++)) || ((FAIL++))
test_port "Frontend Dev Server" "localhost" 5173 && ((PASS++)) || ((FAIL++))
test_port "Frontend Alt Port" "localhost" 5174 && ((PASS++)) || ((FAIL++))

echo ""

# Check Docker
echo -e "${YELLOW}=== DOCKER CONFIGURATION ===${NC}"
echo ""

echo -n "  Checking Docker daemon... "
if docker info > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ Not running${NC}"
    ((FAIL++))
fi

echo -n "  Checking Docker Compose... "
if docker compose version > /dev/null 2>&1; then
    VERSION=$(docker compose version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
    echo -e "${GREEN}✓ Version ${VERSION}${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ Not installed${NC}"
    ((FAIL++))
fi

echo ""

# Check network connectivity utilities
echo -e "${YELLOW}=== NETWORK UTILITIES ===${NC}"
echo ""

echo -n "  Checking netcat (nc)... "
if command -v nc &> /dev/null; then
    echo -e "${GREEN}✓ Available${NC}"
    ((PASS++))
else
    echo -e "${YELLOW}⚠ Not available (fallback to bash)${NC}"
fi

echo -n "  Checking curl... "
if command -v curl &> /dev/null; then
    echo -e "${GREEN}✓ Available${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ Not available${NC}"
    ((FAIL++))
fi

echo ""

# Detailed Checks
echo -e "${YELLOW}=== DETAILED SERVICE CHECKS ===${NC}"
echo ""

# Test PostgreSQL connection
echo -n "  PostgreSQL connection: "
if command -v psql &> /dev/null; then
    if PGPASSWORD=postgres psql -h ${REMOTE_HOST} -U postgres -d alpha -c "SELECT 1;" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Connected${NC}"
        ((PASS++))
    else
        echo -e "${RED}✗ Connection failed${NC}"
        echo "    Try: PGPASSWORD=postgres psql -h ${REMOTE_HOST} -U postgres -d alpha"
        ((FAIL++))
    fi
else
    echo -e "${YELLOW}⚠ psql not installed (skip)${NC}"
fi

# Test Backend Health
echo -n "  Backend health endpoint: "
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
    ((PASS++))
else
    echo -e "${YELLOW}⚠ Backend not responding${NC}"
fi

# Test Hasura
echo -n "  Hasura GraphQL: "
if curl -sf -H "X-Hasura-Admin-Secret: myadminsecret" http://${REMOTE_HOST}:8085/v1/version > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Responding${NC}"
    ((PASS++))
else
    echo -e "${YELLOW}⚠ Hasura not responding${NC}"
fi

# Test Redpanda
echo -n "  Redpanda cluster: "
if curl -sf http://${REMOTE_HOST}:8082/brokers | head -c 10 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
    ((PASS++))
else
    echo -e "${YELLOW}⚠ Redpanda not responding${NC}"
fi

echo ""

# DNS and Network Info
echo -e "${YELLOW}=== NETWORK INFORMATION ===${NC}"
echo ""

echo "  Remote Host: ${REMOTE_HOST}"
echo "  Hostname:"
HOSTNAME=$(hostname 2>/dev/null || echo "N/A")
echo "    $HOSTNAME"

echo "  Local IP:"
LOCAL_IP=$(ifconfig | grep "inet " | grep -v "127.0.0.1" | head -1 | awk '{print $2}' || echo "N/A")
echo "    $LOCAL_IP"

echo "  DNS Resolution:"
DNS_RESULT=$(dig +short ${REMOTE_HOST} 2>/dev/null || echo "N/A")
if [ "$DNS_RESULT" = "$REMOTE_HOST" ] || [ "$DNS_RESULT" = "N/A" ]; then
    echo "    Raw IP (no DNS)"
else
    echo "    $DNS_RESULT"
fi

echo ""

# Summary
echo -e "${BLUE}=============================================================================${NC}"
echo -e "${BLUE}  SUMMARY${NC}"
echo -e "${BLUE}=============================================================================${NC}"
echo ""
echo -e "  Passed: ${GREEN}${PASS}${NC}"
echo -e "  Failed: ${RED}${FAIL}${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed! Your platform is ready to start.${NC}"
    echo ""
    echo -e "Run: ${YELLOW}./start-distributed-platform.sh${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed. Check configuration above.${NC}"
    echo ""
    echo "Troubleshooting tips:"
    echo "  1. Verify remote services are running:"
    echo "     ssh user@${REMOTE_HOST}"
    echo "     docker compose -f docker-compose.remote.yml ps"
    echo ""
    echo "  2. Check network connectivity:"
    echo "     ping -c 3 ${REMOTE_HOST}"
    echo ""
    echo "  3. Verify firewall rules allow access to remote ports"
    echo ""
    echo "  4. Check if using VPN or proxy that might block connections"
    echo ""
    exit 1
fi
