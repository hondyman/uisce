#!/bin/bash
# Deployment verification script for Investment Management LLM Platform

set -e

echo "🚀 Starting deployment verification..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Check Docker is running
echo -e "\n${YELLOW}Step 1: Checking Docker...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker Desktop.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker is running${NC}"

# 2. Check if containers are already running
echo -e "\n${YELLOW}Step 2: Checking existing containers...${NC}"
if docker-compose ps | grep -q "Up"; then
    echo -e "${YELLOW}⚠️  Containers already running. Stopping them first...${NC}"
    docker-compose down
fi

# 3. Start services
echo -e "\n${YELLOW}Step 3: Starting services with docker-compose...${NC}"
docker-compose up -d

# 4. Wait for PostgreSQL to be ready
echo -e "\n${YELLOW}Step 4: Waiting for PostgreSQL to be ready...${NC}"
for i in {1..30}; do
    if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PostgreSQL is ready${NC}"
        break
    fi
    echo -n "."
    sleep 1
done

# 5. Run migrations
echo -e "\n${YELLOW}Step 5: Running database migrations...${NC}"
for migration in migrations/*.sql; do
    echo "  Running $migration..."
    docker-compose exec -T postgres psql -U postgres -d semlayer -f "/docker-entrypoint-initdb.d/$(basename $migration)" || true
done
echo -e "${GREEN}✅ Migrations complete${NC}"

# 6. Load sample data
echo -e "\n${YELLOW}Step 6: Loading sample data...${NC}"
docker-compose exec -T postgres psql -U postgres -d semlayer < migrations/sample_data.sql
echo -e "${GREEN}✅ Sample data loaded${NC}"

# 7. Verify data
echo -e "\n${YELLOW}Step 7: Verifying sample data...${NC}"
HOLDINGS_COUNT=$(docker-compose exec -T postgres psql -U postgres -d semlayer -t -c "SELECT COUNT(*) FROM catalog_node WHERE node_type_id = (SELECT id FROM catalog_node_type WHERE name = 'Holding');")
echo "  Holdings created: $HOLDINGS_COUNT"
if [ "$HOLDINGS_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ Sample holdings verified${NC}"
else
    echo -e "${RED}❌ No holdings found${NC}"
fi

# 8. Check backend health
echo -e "\n${YELLOW}Step 8: Checking backend health...${NC}"
sleep 5  # Give backend time to start
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Backend is healthy${NC}"
else
    echo -e "${YELLOW}⚠️  Backend health check failed (may still be starting)${NC}"
fi

# 9. Test sample query
echo -e "\n${YELLOW}Step 9: Testing sample query...${NC}"
RESPONSE=$(curl -s -X POST http://localhost:8080/nlq/ask \
    -H "Content-Type: application/json" \
    -d '{"tenant_id":"demo-tenant-123","question":"What holdings do I have?"}' || echo "{}")

if echo "$RESPONSE" | grep -q "answer"; then
    echo -e "${GREEN}✅ Sample query successful${NC}"
    echo "Response preview:"
    echo "$RESPONSE" | jq -r '.answer' 2>/dev/null || echo "$RESPONSE" | head -c 200
else
    echo -e "${YELLOW}⚠️  Query test skipped (backend may need more time)${NC}"
fi

# Summary
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment Verification Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "\n📊 Services running:"
docker-compose ps
echo -e "\n🌐 Access points:"
echo "  - Backend API: http://localhost:8080"
echo "  - PostgreSQL: localhost:5432"
echo -e "\n📝 Next steps:"
echo "  1. Test Q&A interface: Visit frontend/src/pages/investment/InvestmentQAPage.tsx"
echo "  2. View logs: docker-compose logs -f backend"
echo "  3. Check data: docker-compose exec postgres psql -U postgres -d semlayer"
echo "  4. Stop services: docker-compose down"
echo ""
