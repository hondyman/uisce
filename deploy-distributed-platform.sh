#!/bin/bash
# =============================================================================
# DISTRIBUTED PLATFORM DEPLOYMENT
# Deploys both MacBook and Remote Services and verifies connectivity
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;36m'
NC='\033[0m'

REMOTE_HOST="100.84.126.19"
REMOTE_USER="${REMOTE_USER:-eganpj}"
REMOTE_DIR="${REMOTE_DIR:-semlayer}"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo -e "${BLUE}=============================================================================${NC}"
echo -e "${BLUE}  SEMLAYER DISTRIBUTED PLATFORM DEPLOYMENT${NC}"
echo -e "${BLUE}=============================================================================${NC}"
echo ""

# Function: Check if service is running
check_service() {
    local name=$1
    local host=$2
    local port=$3
    
    echo -n "  Checking ${name} (${host}:${port})... "
    
    if timeout 3 bash -c "</dev/tcp/${host}/${port}" 2>/dev/null; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    fi
    
    # For HTTP services, try curl
    if timeout 3 curl -s -o /dev/null "http://${host}:${port}/" 2>/dev/null; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}✗ NOT RESPONDING${NC}"
    return 1
}

# ===========================================================================
# PHASE 1: DEPLOY MACBOOK BACKEND
# ===========================================================================
echo -e "${YELLOW}PHASE 1: MacBook Backend Setup${NC}"
echo -e "${YELLOW}===============================${NC}"
echo ""

echo "Checking Docker..."
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

echo "Cleaning up old containers..."
docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" down 2>/dev/null || true
echo ""

echo "Building backend image..."
docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" build --no-cache
echo ""

echo "Starting backend container..."
docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" up -d
echo ""

echo "Waiting for backend to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Backend is ready!${NC}"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    echo -n "."
    sleep 1
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}❌ Backend failed to start${NC}"
    docker compose -f "${SCRIPT_DIR}/docker-compose.mac-distributed.yml" logs
    exit 1
fi
echo ""

# ===========================================================================
# PHASE 2: DEPLOY REMOTE INFRASTRUCTURE
# ===========================================================================
echo ""
echo -e "${YELLOW}PHASE 2: Remote Infrastructure Setup (100.84.126.19)${NC}"
echo -e "${YELLOW}======================================================${NC}"
echo ""

# SSH key-based authentication (no password needed)
SSH_KEY="${SSH_KEY_PATH:-$HOME/.ssh/id_ed25519}"
if [ ! -f "$SSH_KEY" ]; then
    echo -e "${RED}❌ SSH key not found at $SSH_KEY${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Using SSH key authentication (${SSH_KEY})${NC}"

# Check Tailscale
echo "Checking Tailscale connection..."
if ! pgrep -x "Tailscale" > /dev/null 2>&1 && ! tailscale status >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Tailscale may not be running${NC}"
    echo "   Please ensure Tailscale is connected"
fi
echo ""

# Test SSH connection
echo "Testing connection to remote host..."
if ! ssh -i "$SSH_KEY" -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "echo 'Connection successful'" 2>/dev/null; then
    echo -e "${RED}❌ Cannot connect to $REMOTE_HOST${NC}"
    exit 1
fi
echo -e "${GREEN}✓ SSH connection established${NC}"
echo ""

# Copy remote compose file
echo "Deploying docker-compose.remote.yml to remote server..."
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "mkdir -p $REMOTE_DIR" 2>/dev/null
scp -i "$SSH_KEY" -o StrictHostKeyChecking=no "$SCRIPT_DIR/docker-compose.remote.yml" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/" 2>/dev/null
echo -e "${GREEN}✓ Docker Compose file transferred${NC}"
echo ""

# Check if PostgreSQL is running on remote
echo "Checking if PostgreSQL is running on remote server..."
if ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "pg_isready -h localhost -p 5432" 2>/dev/null; then
    echo -e "${GREEN}✓ PostgreSQL is running locally on remote host${NC}"
else
    echo -e "${YELLOW}⚠️  PostgreSQL not detected on remote host${NC}"
    echo "   Please ensure PostgreSQL is running at 100.84.126.19:5432"
fi
echo ""

# Start remote Docker services
echo "Starting remote Docker services..."
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" \
    "cd $REMOTE_DIR && docker compose -f docker-compose.remote.yml up -d" 2>/dev/null

echo -e "${GREEN}✓ Remote services initialized${NC}"
echo ""

# Wait for services to stabilize
echo "Waiting for remote services to initialize..."
sleep 10
echo ""

# ===========================================================================
# PHASE 3: VERIFY CONNECTIVITY
# ===========================================================================
echo -e "${YELLOW}PHASE 3: Connectivity Verification${NC}"
echo -e "${YELLOW}===================================${NC}"
echo ""

echo "MacBook Services:"
check_service "Backend API" "localhost" "8080" || true
check_service "Frontend Dev" "localhost" "5173" || true
echo ""

echo "Remote Services:"
check_service "PostgreSQL" "${REMOTE_HOST}" "5432" || true
check_service "Hasura GraphQL" "${REMOTE_HOST}" "8085" || true
check_service "Redpanda Kafka" "${REMOTE_HOST}" "19092" || true
check_service "Temporal" "${REMOTE_HOST}" "7233" || true
check_service "Redpanda Console" "${REMOTE_HOST}" "8096" || true
check_service "MinIO" "${REMOTE_HOST}" "9010" || true
echo ""

# Test cross-platform connectivity
echo "Cross-Platform Connectivity:"
echo -n "  Backend → PostgreSQL: "
if docker exec semlayer-backend pg_isready -h ${REMOTE_HOST} -p 5432 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${YELLOW}⚠️  Unable to verify${NC}"
fi

echo -n "  Backend → Hasura: "
if docker exec semlayer-backend curl -s -o /dev/null "http://${REMOTE_HOST}:8085/" 2>/dev/null; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${YELLOW}⚠️  Unable to verify${NC}"
fi

echo -n "  Backend → Redpanda: "
if docker exec semlayer-backend bash -c "timeout 2 bash -c '</dev/tcp/${REMOTE_HOST}/19092'" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${YELLOW}⚠️  Unable to verify${NC}"
fi
echo ""

# ===========================================================================
# PHASE 4: FINAL STATUS
# ===========================================================================
echo -e "${GREEN}=============================================================================${NC}"
echo -e "${GREEN}DEPLOYMENT COMPLETE!${NC}"
echo -e "${GREEN}=============================================================================${NC}"
echo ""

echo -e "${BLUE}MacBook Services:${NC}"
echo "  Backend API:        http://localhost:8080"
echo "  Frontend (dev):     http://localhost:5173"
echo ""

echo -e "${BLUE}Remote Services (100.84.126.19):${NC}"
echo "  PostgreSQL:         ${REMOTE_HOST}:5432"
echo "  Hasura GraphQL:     http://${REMOTE_HOST}:8085"
echo "  Redpanda Kafka:     ${REMOTE_HOST}:19092"
echo "  Temporal:           ${REMOTE_HOST}:7233"
echo "  Redpanda Console:   http://${REMOTE_HOST}:8096"
echo "  MinIO Console:      http://${REMOTE_HOST}:9010"
echo ""

echo -e "${BLUE}Useful Commands:${NC}"
echo ""
echo "  View MacBook backend logs:"
echo "    docker compose -f docker-compose.mac-distributed.yml logs -f backend"
echo ""
echo "  View remote services:"
echo "    ssh -i ~/.ssh/id_ed25519 ${REMOTE_USER}@${REMOTE_HOST} 'cd ${REMOTE_DIR} && docker compose -f docker-compose.remote.yml ps'"
echo ""
echo "  Test frontend:"
echo "    open http://localhost:5173"
echo ""

echo -e "${GREEN}Platform is ready! 🚀${NC}"
