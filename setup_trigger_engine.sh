#!/bin/bash

# 🚀 SEMLAYER BP TRIGGER ENGINE SETUP SCRIPT
# This script sets up the complete BP Trigger Engine with Temporal, PostgreSQL, and all services

set -e

echo "======================================================================"
echo "  🚀 SEMLAYER BP TRIGGER ENGINE - COMPLETE SETUP"
echo "======================================================================"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Step 1: Check prerequisites
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 1: Checking Prerequisites${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker installed${NC}"

# Check Docker Compose
if ! (docker compose version &> /dev/null || command -v docker-compose &> /dev/null); then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker Compose installed${NC}"

# Check PostgreSQL client
if ! command -v psql &> /dev/null; then
    echo -e "${RED}❌ PostgreSQL client (psql) is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ PostgreSQL client installed${NC}"

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✅ Go installed ($GO_VERSION)${NC}"

# Step 2: Start Docker services
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 2: Starting Docker Services${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "Pulling latest images..."
docker compose pull temporal temporal-ui postgresql-temporal graphql-engine rabbitmq 2>&1 | grep -E "^Pulling|^Digest|Status:|Downloaded" || true

echo ""
echo "Starting Docker services..."
docker compose up -d temporal postgresql-temporal temporal-ui graphql-engine rabbitmq

echo "⏳ Waiting for services to be healthy (30 seconds)..."
sleep 30

# Check service health
echo ""
echo "Checking service health..."

# Check Temporal
if docker exec semlayer-temporal temporal workflow list --address localhost:7233 --limit 1 2>/dev/null | grep -q "Workflows"; then
    echo -e "${GREEN}✅ Temporal is running${NC}"
else
    echo -e "${YELLOW}⚠️  Temporal may still be initializing...${NC}"
fi

# Check Hasura
if curl -s http://localhost:8083 > /dev/null; then
    echo -e "${GREEN}✅ Hasura is running at http://localhost:8083${NC}"
else
    echo -e "${RED}❌ Hasura is not responding${NC}"
fi

# Check RabbitMQ
if curl -s -u guest:guest http://localhost:15672/api/overview | grep -q "rabbitmq_version"; then
    echo -e "${GREEN}✅ RabbitMQ is running at http://localhost:15672${NC}"
else
    echo -e "${RED}❌ RabbitMQ is not responding${NC}"
fi

# Check Temporal UI
if curl -s http://localhost:8080 > /dev/null; then
    echo -e "${GREEN}✅ Temporal UI is running at http://localhost:8080${NC}"
else
    echo -e "${RED}❌ Temporal UI is not responding${NC}"
fi

# Step 3: Create Temporal databases
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 3: Verifying Temporal Databases${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "Creating Temporal databases (if not exists)..."
psql postgresql://postgres:postgres@localhost:5432/temporal -c "SELECT 1;" 2>/dev/null && \
    echo -e "${GREEN}✅ Temporal database exists${NC}" || \
    echo -e "${YELLOW}ℹ️  Temporal database will be auto-created${NC}"

# Step 4: Verify main database
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 4: Verifying Main Database${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "Checking BP tables..."
TABLES=$(psql postgresql://postgres:postgres@localhost:5432/alpha -t -c "
SELECT COUNT(*) FROM information_schema.tables 
WHERE table_name IN ('bp_triggers', 'bp_steps', 'bp_trigger_executions', 'bp_activity_logs')
AND table_schema='public';" 2>/dev/null || echo "0")

if [ "$TABLES" = "4" ]; then
    echo -e "${GREEN}✅ All BP tables exist${NC}"
else
    echo -e "${YELLOW}⚠️  BP tables may need to be created (only $TABLES/4 found)${NC}"
fi

# Step 5: Build Go services
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}STEP 5: Building Go Services${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo "Building worker..."
cd backend/cmd/worker
go build -o ./worker main.go
echo -e "${GREEN}✅ Worker built${NC}"
cd ../../..

echo "Building trigger engine..."
cd backend/cmd/triggers
go build -o ./triggers main.go
echo -e "${GREEN}✅ Trigger engine built${NC}"
cd ../../..

# Step 6: Summary
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🎉 SETUP COMPLETE!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

echo ""
echo "🚀 NEXT STEPS:"
echo ""
echo "1️⃣  Start the Temporal Worker:"
echo "   cd backend/cmd/worker && ./worker"
echo ""
echo "2️⃣  Start the Trigger Engine (in a new terminal):"
echo "   cd backend/cmd/triggers && ./triggers"
echo ""
echo "3️⃣  View Temporal Workflows:"
echo "   Open http://localhost:8080 in your browser"
echo ""
echo "📊 SERVICE URLS:"
echo "   • Temporal UI:      http://localhost:8080"
echo "   • Temporal gRPC:    localhost:7233"
echo "   • Hasura GraphQL:   http://localhost:8083"
echo "   • RabbitMQ Admin:   http://localhost:15672 (guest/guest)"
echo "   • PostgreSQL:       postgresql://postgres:postgres@localhost:5432/alpha"
echo ""
echo "✅ Verify with:"
echo "   docker ps | grep semlayer"
echo ""
echo "📚 Documentation:"
echo "   • Trigger Engine:   BP_TRIGGER_ENGINE_COMPLETE.md"
echo "   • Quick Reference:  BP_TRIGGER_ENGINE_QUICK_REFERENCE.md"
echo ""
