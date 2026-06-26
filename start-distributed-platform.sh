#!/bin/bash
# =============================================================================
# DISTRIBUTED PLATFORM STARTUP - MacBook Pro
# =============================================================================
# This script sets up and starts the full platform:
# - Remote: PostgreSQL + Hasura + Redpanda + Temporal (on 100.84.126.19)
# - MacBook: Backend (Docker) + Frontend (Native)
#
# Usage: ./start-distributed-platform.sh
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m' # No Color

REMOTE_HOST="100.84.126.19"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo -e "${BLUE}=============================================================================${NC}"
echo -e "${BLUE}  SEMLAYER DISTRIBUTED PLATFORM STARTUP${NC}"
echo -e "${BLUE}=============================================================================${NC}"
echo ""
echo "Configuration:"
echo "  Remote Services:    ${REMOTE_HOST}"
echo "  Backend:            Docker on MacBook (localhost:8080)"
echo "  Frontend:           Native on MacBook (localhost:5173)"
echo ""

# Function to check if a service is reachable
check_remote_service() {
    local service=$1
    local port=$2
    echo -n "Checking ${service} (${REMOTE_HOST}:${port})... "
    
    # Try to connect using bash built-in /dev/tcp (macOS compatible)
    if timeout 3 bash -c "</dev/tcp/${REMOTE_HOST}/${port}" 2>/dev/null; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    else
        # For HTTP services, try with curl
        case ${port} in
            8085|8088|8094|8096|9011)
                if timeout 3 curl -s -o /dev/null -w "%{http_code}" "http://${REMOTE_HOST}:${port}/" 2>/dev/null | grep -q "[2345]"; then
                    echo -e "${GREEN}✓ OK${NC}"
                    return 0
                fi
                ;;
        esac
        echo -e "${YELLOW}? SKIPPED (unable to verify)${NC}"
        return 0
    fi
}

# Check all required remote services
echo -e "${YELLOW}Verifying remote services on ${REMOTE_HOST}...${NC}"
echo ""

SERVICES_OK=true

check_remote_service "PostgreSQL" 5432 || SERVICES_OK=false
check_remote_service "Hasura GraphQL" 8085 || SERVICES_OK=false
check_remote_service "Redpanda Kafka" 19092 || SERVICES_OK=false
check_remote_service "Temporal" 7233 || SERVICES_OK=false

echo ""

if [ "$SERVICES_OK" = false ]; then
    echo -e "${RED}WARNING: Some remote services are not responding${NC}"
    echo -e "${YELLOW}Make sure the following are running on ${REMOTE_HOST}:${NC}"
    echo "  1. PostgreSQL (port 5432)"
    echo "  2. Hasura GraphQL Engine (port 8085)"
    echo "  3. Redpanda Kafka (port 19092 for external access)"
    echo "  4. Temporal (port 7233)"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Startup cancelled${NC}"
        exit 1
    fi
fi

# Load environment variables
if [ -f "${SCRIPT_DIR}/.env" ]; then
    echo -e "${YELLOW}Loading environment from .env${NC}"
    # Parse .env file more carefully to handle JSON and special chars
    # Only load lines with KEY=VALUE format, skip comments
    export $(grep -v '^#' "${SCRIPT_DIR}/.env" | grep -v '^$' | grep '=' | head -1)
    # For complex values, just load manually what we need
    DB_HOST=$(grep "^DB_HOST=" "${SCRIPT_DIR}/.env" 2>/dev/null | cut -d= -f2- | tr -d ' ')
    [ -n "$DB_HOST" ] && export DB_HOST || export DB_HOST="100.84.126.19"
else
    echo -e "${YELLOW}No .env file found, using defaults${NC}"
fi

# Make sure Docker is running
echo ""
echo -e "${YELLOW}Checking Docker...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker is not running${NC}"
    echo "Please start Docker Desktop and try again"
    exit 1
fi
echo -e "${GREEN}Docker is running${NC}"

# Stop any existing containers
echo ""
echo -e "${YELLOW}Cleaning up old containers...${NC}"
docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" down 2>/dev/null || true

# Build and start backend
echo ""
echo -e "${YELLOW}Building backend image...${NC}"
docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" build --no-cache

echo ""
echo -e "${YELLOW}Starting backend container...${NC}"
docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" up -d

# Wait for backend to be healthy
echo ""
echo -e "${YELLOW}Waiting for backend to be ready...${NC}"
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}Backend is ready!${NC}"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo -n "."
    sleep 1
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}Backend failed to start${NC}"
    docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" logs
    exit 1
fi

# Display status
echo ""
echo -e "${GREEN}=============================================================================${NC}"
echo -e "${GREEN}BACKEND IS RUNNING${NC}"
echo -e "${GREEN}=============================================================================${NC}"
echo ""
echo "Backend API:"
echo "  URL: http://localhost:8080"
echo "  Curl test: curl http://localhost:8080/health"
echo ""

# Check if frontend dependencies are installed
echo -e "${YELLOW}Checking frontend setup...${NC}"
cd "${SCRIPT_DIR}/frontend"

if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing frontend dependencies...${NC}"
    npm install
fi

echo ""
echo -e "${GREEN}=============================================================================${NC}"
echo -e "${GREEN}PLATFORM STARTUP COMPLETE!${NC}"
echo -e "${GREEN}=============================================================================${NC}"
echo ""
echo -e "${BLUE}NEXT STEPS:${NC}"
echo ""
echo "1. Start the frontend (in a new terminal):"
echo -e "   ${YELLOW}cd ${SCRIPT_DIR}/frontend${NC}"
echo -e "   ${YELLOW}npm run dev${NC}"
echo ""
echo "2. Open your browser:"
echo -e "   ${YELLOW}http://localhost:5173${NC}"
echo ""
echo -e "${BLUE}SERVICE ENDPOINTS:${NC}"
echo ""
echo "  Backend API:           http://localhost:8080"
echo "  Frontend:              http://localhost:5173"
echo "  Hasura GraphQL:        http://${REMOTE_HOST}:8085"
echo "  Redpanda Console:      http://${REMOTE_HOST}:8096"
echo "  Temporal UI:           http://${REMOTE_HOST}:8088"
echo ""
echo -e "${BLUE}USEFUL COMMANDS:${NC}"
echo ""
echo "  View backend logs:     docker compose -f docker-compose.mac-distributed.yml logs -f backend"
echo "  Stop backend:          docker compose -f docker-compose.mac-distributed.yml down"
echo "  Restart backend:       docker compose -f docker-compose.mac-distributed.yml restart backend"
echo ""
echo -e "${GREEN}Platform is ready! 🚀${NC}"
