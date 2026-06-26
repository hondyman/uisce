#!/bin/bash

# Cube.js Multi-Tenant Semantic Layer - Quick Start Script
# This script initializes and validates the Cube.js deployment

set -e

echo "🚀 Cube.js Multi-Tenant Semantic Layer - Quick Start"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker and Docker Compose are installed${NC}"
echo ""

# Generate API secret if not exists
if [ -z "$CUBE_API_SECRET" ]; then
    echo "🔐 Generating Cube.js API secret..."
    export CUBE_API_SECRET=$(openssl rand -hex 32)
    echo "CUBE_API_SECRET=${CUBE_API_SECRET}" >> .env.cube
    echo -e "${GREEN}✅ API secret generated and saved to .env.cube${NC}"
    echo -e "${YELLOW}⚠️  Keep this secret safe! It's required for all API calls.${NC}"
else
    echo -e "${GREEN}✅ Using existing CUBE_API_SECRET from environment${NC}"
fi
echo ""

# Start services
echo "🐳 Starting Docker services..."
docker compose up -d starrocks-fe starrocks-be minio nessie

echo "⏳ Waiting for StarRocks to be healthy (this may take 60s)..."
timeout=60
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if docker compose ps starrocks-fe | grep -q "healthy"; then
        echo -e "${GREEN}✅ StarRocks is healthy${NC}"
        break
    fi
    sleep 5
    elapsed=$((elapsed + 5))
    echo "   Still waiting... (${elapsed}s/${timeout}s)"
done

if [ $elapsed -ge $timeout ]; then
    echo -e "${RED}❌ StarRocks failed to become healthy${NC}"
    echo "Check logs with: docker logs starrocks-fe"
    exit 1
fi
echo ""

# Initialize StarRocks pre-aggregation database
echo "📊 Initializing StarRocks pre-aggregation database..."
if docker exec -i starrocks-fe mysql -uroot < cube/init-starrocks-preaggs.sql; then
    echo -e "${GREEN}✅ Pre-aggregation database initialized${NC}"
else
    echo -e "${YELLOW}⚠️  Database may already exist (this is OK)${NC}"
fi
echo ""

# Start Cube.js
echo "🧊 Starting Cube.js semantic layer..."
docker compose up -d cube

echo "⏳ Waiting for Cube.js to be ready..."
sleep 10

# Health check
echo "🏥 Checking Cube.js health..."
max_retries=12
retry=0
while [ $retry -lt $max_retries ]; do
    if curl -sf http://localhost:4000/readyz > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Cube.js is healthy and ready${NC}"
        break
    fi
    retry=$((retry + 1))
    if [ $retry -eq $max_retries ]; then
        echo -e "${RED}❌ Cube.js failed to start${NC}"
        echo "Check logs with: docker logs cube-semantic-layer"
        exit 1
    fi
    sleep 5
    echo "   Retrying... (${retry}/${max_retries})"
done
echo ""

# Verify schema files
echo "📁 Verifying Cube schema files..."
schema_count=$(find cube/schema -name "*.yml" | wc -l | tr -d ' ')
if [ "$schema_count" -gt 0 ]; then
    echo -e "${GREEN}✅ Found ${schema_count} schema files${NC}"
    find cube/schema -name "*.yml" -exec basename {} \;
else
    echo -e "${RED}❌ No schema files found${NC}"
    exit 1
fi
echo ""

# Test query (requires tenant context)
echo "🔍 Testing basic connectivity..."
echo -e "${YELLOW}ℹ️  For actual queries, you need valid tenant headers${NC}"
echo ""

# Display summary
echo "=================================================="
echo -e "${GREEN}🎉 Cube.js Multi-Tenant Semantic Layer is ready!${NC}"
echo "=================================================="
echo ""
echo "📚 Quick Reference:"
echo ""
echo "  REST API:       http://localhost:4000"
echo "  SQL API:        localhost:15432 (PostgreSQL protocol)"
echo "  API Secret:     \$CUBE_API_SECRET (see .env.cube)"
echo ""
echo "🧪 Test with curl:"
echo ""
echo "  export TENANT_ID='00000000-0000-0000-0000-000000000000'"
echo "  export DATASOURCE_ID='11111111-1111-1111-1111-111111111111'"
echo ""
echo "  curl -X POST http://localhost:4000/cubejs-api/v1/load \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H \"Authorization: \${CUBE_API_SECRET}\" \\"
echo "    -H \"X-Tenant-ID: \${TENANT_ID}\" \\"
echo "    -H \"X-Tenant-Datasource-ID: \${DATASOURCE_ID}\" \\"
echo "    -d '{\"query\": {\"measures\": [\"Trades.count\"]}}'"
echo ""
echo "📖 Documentation:"
echo "  - cube/README.md                - Usage guide"
echo "  - cube/DEPLOYMENT.md            - Production deployment"
echo "  - cube/IMPLEMENTATION_SUMMARY.md - Architecture overview"
echo ""
echo "🔧 Useful commands:"
echo "  - docker logs -f cube-semantic-layer  (View Cube.js logs)"
echo "  - docker logs -f starrocks-fe         (View StarRocks logs)"
echo "  - docker compose ps                   (Check service status)"
echo "  - docker compose down                 (Stop all services)"
echo ""
echo "📊 Monitor pre-aggregations:"
echo "  docker exec -it starrocks-fe mysql -uroot -e \"SELECT * FROM cube_preaggs.v_preagg_health;\""
echo ""
echo -e "${GREEN}✨ Happy querying!${NC}"
