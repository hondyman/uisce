#!/bin/bash
set -e

echo "🎯 Testing Audit & Snapshot Plane"
echo "=================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check if infrastructure is running
echo -e "${BLUE}1. Checking infrastructure...${NC}"

if ! docker ps | grep -q audit-redpanda; then
    echo -e "${RED}❌ Redpanda not running. Start with: cd audit-infrastructure && ./start.sh${NC}"
    exit 1
fi

if ! docker ps | grep -q audit-trino; then
    echo -e "${RED}❌ Trino not running. Start with: cd audit-infrastructure && ./start.sh${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Infrastructure is running${NC}"
echo ""

# Test Kafka connectivity
echo -e "${BLUE}2. Testing Kafka connectivity...${NC}"
if docker exec audit-redpanda rpk cluster info > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Kafka is accessible${NC}"
else
    echo -e "${RED}❌ Cannot connect to Kafka${NC}"
    exit 1
fi
echo ""

# List topics
echo -e "${BLUE}3. Listing Kafka topics...${NC}"
docker exec audit-redpanda rpk topic list | grep audit || echo "No audit topics found yet"
echo ""

# Test Trino connectivity
echo -e "${BLUE}4. Testing Trino connectivity...${NC}"
if docker exec audit-trino trino --execute "SELECT 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Trino is accessible${NC}"
else
    echo -e "${RED}❌ Cannot connect to Trino${NC}"
    exit 1
fi
echo ""

# Check Iceberg tables
echo -e "${BLUE}5. Checking Iceberg tables...${NC}"
TABLES=$(docker exec audit-trino trino --execute "SHOW TABLES IN iceberg.audit" 2>/dev/null || echo "")
if [ -z "$TABLES" ]; then
    echo -e "${YELLOW}⚠️  No tables found. Run DDL scripts:${NC}"
    echo "   docker exec -i audit-trino trino < ../internal/audit/iceberg_schema.sql"
else
    echo -e "${GREEN}✓ Tables exist${NC}"
    echo "$TABLES" | head -5
fi
echo ""

# Publish test events
echo -e "${BLUE}6. Publishing test audit events...${NC}"
cd ..
if go run scripts/test-audit-event.go; then
    echo -e "${GREEN}✓ Test events published${NC}"
else
    echo -e "${YELLOW}⚠️  Test events may have failed (check if backend builds)${NC}"
fi
cd audit-infrastructure
echo ""

# Wait for events to be ingested
echo -e "${BLUE}7. Waiting for events to be ingested (10 seconds)...${NC}"
sleep 10
echo ""

# Query data
echo -e "${BLUE}8. Querying audit data from Trino...${NC}"
echo "Job Runs:"
docker exec audit-trino trino --execute "SELECT run_id, job_id, status, start_ts FROM iceberg.audit.scheduler_job_runs LIMIT 5" 2>/dev/null || echo "No data yet (may take a moment for sink to process)"
echo ""

echo "Compliance Violations:"
docker exec audit-trino trino --execute "SELECT violation_id, tenant_id, violation_type, severity FROM iceberg.audit.compliance_violations LIMIT 5" 2>/dev/null || echo "No data yet"
echo ""

# Test API (if running)
echo -e "${BLUE}9. Testing Audit API (if backend is running)...${NC}"
RESPONSE=$(curl -s -H "X-Tenant-ID: tenant-test-001" http://localhost:8080/api/audit/job-runs 2>/dev/null || echo "")
if [ ! -z "$RESPONSE" ]; then
    echo -e "${GREEN}✓ API is responding${NC}"
    echo "$RESPONSE" | head -20
else
    echo -e "${YELLOW}⚠️  API not responding (make sure backend is running with audit routes)${NC}"
fi
echo ""

# Summary
echo "=================================="
echo -e "${GREEN}✅ Test Complete!${NC}"
echo ""
echo "📊 View data:"
echo "  • Kafka Console: http://localhost:8080"
echo "  • MinIO Console: http://localhost:9001"
echo "  • Query Trino:"
echo "    docker exec -it audit-trino trino"
echo "    USE iceberg.audit;"
echo "    SELECT * FROM scheduler_job_runs;"
echo ""
echo "🔍 Next steps:"
echo "  1. Wire up publishers in your scheduler code"
echo "  2. Add audit routes to your API server"
echo "  3. Deploy the Audit Explorer UI"
echo ""
