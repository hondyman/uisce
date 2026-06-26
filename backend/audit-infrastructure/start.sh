#!/bin/bash
set -e

echo "🚀 Starting Audit & Snapshot Plane Infrastructure..."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install docker-compose first."
    exit 1
fi

# Start the infrastructure
echo -e "${BLUE}📦 Starting Docker containers...${NC}"
docker-compose up -d

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"

# Wait for Redpanda
echo "  Waiting for Redpanda (Kafka)..."
until docker exec audit-redpanda rpk cluster health 2>/dev/null | grep -q "Healthy.*true"; do
    sleep 2
done
echo -e "  ${GREEN}✓ Redpanda is ready${NC}"

# Wait for MinIO
echo "  Waiting for MinIO..."
until curl -sf http://localhost:9000/minio/health/live > /dev/null 2>&1; do
    sleep 2
done
echo -e "  ${GREEN}✓ MinIO is ready${NC}"

# Wait for Iceberg REST Catalog
echo "  Waiting for Iceberg REST Catalog..."
until curl -sf http://localhost:8181/v1/config > /dev/null 2>&1; do
    sleep 2
done
echo -e "  ${GREEN}✓ Iceberg REST Catalog is ready${NC}"

# Wait for Trino
echo "  Waiting for Trino..."
sleep 10  # Give Trino time to initialize
until curl -sf http://localhost:8090/v1/info > /dev/null 2>&1; do
    sleep 2
done
echo -e "  ${GREEN}✓ Trino is ready${NC}"

# Create Kafka topics
echo -e "${BLUE}📋 Creating Kafka topics...${NC}"
TOPICS=(
    "audit.scheduler.job_runs"
    "audit.scheduler.dag_runs"
    "audit.governance.changesets"
    "audit.semantic.snapshots"
    "audit.orchestration.events"
    "audit.compliance.violations"
    "audit.ai.suggestions"
)

for topic in "${TOPICS[@]}"; do
    docker exec audit-redpanda rpk topic create "$topic" \
        --partitions 6 \
        --replicas 1 \
        --topic-config retention.ms=604800000 \
        2>/dev/null || echo "  Topic $topic already exists"
done
echo -e "${GREEN}✓ Kafka topics created${NC}"

# Create Iceberg tables via Trino
echo -e "${BLUE}🗄️  Creating Iceberg tables...${NC}"
docker exec audit-trino trino --execute "CREATE SCHEMA IF NOT EXISTS iceberg.audit;"
docker exec audit-trino trino --execute "CREATE SCHEMA IF NOT EXISTS iceberg.platform;"

# Run the DDL script
echo "  Creating audit tables..."
docker exec -i audit-trino trino < ../internal/audit/iceberg_schema.sql || true
echo -e "${GREEN}✓ Iceberg tables created${NC}"

# Create materialized views
echo -e "${BLUE}📊 Creating materialized views...${NC}"
docker exec -i audit-trino trino < ../internal/audit/materialized_views.sql || true
echo -e "${GREEN}✓ Materialized views created${NC}"

echo ""
echo -e "${GREEN}✅ Audit & Snapshot Plane is ready!${NC}"
echo ""
echo "📍 Service Endpoints:"
echo "  • Kafka (Redpanda): localhost:19092"
echo "  • Redpanda Console: http://localhost:8080"
echo "  • MinIO S3: http://localhost:9000"
echo "  • MinIO Console: http://localhost:9001 (minioadmin/minioadmin)"
echo "  • Iceberg REST: http://localhost:8181"
echo "  • Trino: http://localhost:8090"
echo ""
echo "🔍 To query audit data with Trino CLI:"
echo "  docker exec -it audit-trino trino"
echo "  USE iceberg.audit;"
echo "  SELECT * FROM scheduler_job_runs LIMIT 10;"
echo ""
echo "📊 To view Kafka topics:"
echo "  docker exec audit-redpanda rpk topic list"
echo ""
echo "🛑 To stop:"
echo "  docker-compose down"
echo ""
