# Usice MDM Implementation Guide

## End-to-End Setup & Deployment

This document provides complete instructions for deploying the **Usice Semantic Master Data Management System** in your environment.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Database Setup](#database-setup)
3. [Building Services](#building-services)
4. [Docker Deployment](#docker-deployment)
5. [Verification & Testing](#verification--testing)
6. [Operations](#operations)

---

## Prerequisites

### System Requirements
- Docker & Docker Compose 3.8+
- Go 1.21+ (for building services)
- Python 3.11+ (for local testing)
- PostgreSQL 15+ (external, running on 100.84.126.19)
- 4GB free RAM minimum (8GB recommended)

### Network Prerequisites
- Postgres accessible on 100.84.126.19:5432
- Firewall allows Docker container → 100.84.126.19:5432 (TCP)
- Docker network subnet: 172.28.0.0/16

### Credentials & Secrets
- Postgres password for `usice_app` user
- (Optional) API keys for commercial sources (TradingHours, EODHD, etc.)

---

## Database Setup

### Step 1: Connect to Postgres (on 100.84.126.19)

```bash
# From your Macbook, connect to the external Postgres
psql -h 100.84.126.19 -U postgres
```

### Step 2: Create Users (if not already present)

```sql
-- Create application user (if not exists)
CREATE USER IF NOT EXISTS usice_app WITH PASSWORD 'your-secure-password-here';

-- Create ops user (if not exists)
CREATE USER IF NOT EXISTS usice_ops WITH PASSWORD 'your-secure-ops-password';

-- Grant permissions on alpha database
GRANT CONNECT ON DATABASE alpha TO usice_app;
GRANT CONNECT ON DATABASE alpha TO usice_ops;
```

Note: The `edm` schema will be created automatically by the DDL script below.

### Step 3: Apply Schema

```bash
# Run the DDL script to create the edm schema and tables within the alpha database
psql -h 100.84.126.19 -U postgres -d alpha -f schema/001_mdm_init.sql
```

### Step 4: Verify Installation

```bash
# Connect as the application user to the alpha database
psql -h 100.84.126.19 -U usice_app -d alpha

# List tables in edm schema
\dt edm.mdm_*

# Verify source registry is seeded
SELECT source_name, is_active, priority_score FROM edm.mdm_source_registry ORDER BY priority_score;
```

Expected output:
```
  source_name  | is_active | priority_score
---------------+-----------+----------------
 NagerDate     | t         | 4
 OpenHolidays  | t         | 4
 Workalendar   | t         | 3
 HolidaysPyPI  | t         | 3
 TradingHours  | f         | 1
 EODHD         | f         | 2
 Xignite       | f         | 2
 Finnhub       | f         | 2
(8 rows)
```

---

## Building Services

### Step 1: Build Go Services

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Build API Gateway
go build -o bin/api-gateway ./cmd/api-gateway

# Build Semantic Engine
go build -o bin/semantic-engine ./cmd/semantic-engine
```

### Step 2: Build Python Services (Docker images)

Python services will be built by Docker Compose automatically. Verify Dockerfiles exist:

```bash
ls -la services/workalendar-adapter/Dockerfile
ls -la services/holidays-adapter/Dockerfile
```

### Step 3: Prepare Environment Variables

```bash
# Create .env file for Docker Compose
cat > .env.mdm << 'EOF'
DB_PASSWORD=your-secure-password-here
DB_HOST=100.84.126.19
DB_PORT=5432
DB_NAME=alpha
DB_USER=usice_app

# Redpanda settings
REDPANDA_BROKERS=redpanda:9092

# API Keys (optional for commercial sources)
TRADINGHOURS_API_KEY=your-key-here
EODHD_API_KEY=your-key-here
EOF
```

---

## Docker Deployment

### Step 1: Configure Docker Network

Docker Compose will create the network, but ensure your Postgres firewall allows it:

```bash
# On your desktop (100.84.126.19), verify Postgres firewall
# Allow Docker subnet to connect
sudo ufw allow from 172.28.0.0/16 to any port 5432 proto tcp

# Or if using different firewall:
sudo iptables -A INPUT -s 172.28.0.0/16 -p tcp --dport 5432 -j ACCEPT
```

### Step 2: Start All Services

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Start with Docker Compose
docker-compose -f docker-compose.mdm.yml up -d

# Watch logs
docker-compose -f docker-compose.mdm.yml logs -f
```

### Step 3: Verify Service Health

```bash
# Check all containers are running
docker-compose -f docker-compose.mdm.yml ps

# Expected output:
# NAME                   STATUS              PORTS
# redpanda-1            Up 30s             0.0.0.0:9092->9092/tcp
# schema-registry       Up 25s             0.0.0.0:8081->8081/tcp
# workalendar-adapter   Up 20s             0.0.0.0:8000->8000/tcp
# holidays-adapter      Up 20s             0.0.0.0:8001->8001/tcp
# semantic-engine       Up 15s             0.0.0.0:9000->9000/tcp
# api-gateway           Up 10s             0.0.0.0:8080->8080/tcp
# mdm-frontend          Up 5s              0.0.0.0:3000->80/tcp
```

### Step 4: Verify Individual Services

```bash
# Workalendar Health
curl http://localhost:8000/health

# Holidays Health
curl http://localhost:8001/health

# Semantic Engine Health
curl http://localhost:9000/health

# API Gateway Health
curl http://localhost:8080/health

# Redpanda Health
curl http://localhost:9644/metrics | head -5

# Frontend
open http://localhost:3000
```

---

## Verification & Testing

### Test 1: Verify Source Registry in Database

```bash
psql -h 100.84.126.19 -U usice_app -d alpha << 'SQL'
SELECT source_name, is_active, priority_score, confidence_base
FROM edm.mdm_source_registry
ORDER BY priority_score;
SQL
```

### Test 2: Trigger Manual Ingestion

```bash
# Create a test tenant
TENANT_ID="00000000-0000-0000-0000-000000000001"

# Trigger ingestion via API
curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "tenant_id": "'$TENANT_ID'",
    "regions": ["US"],
    "year": 2026
  }'
```

### Test 3: Query Golden Calendar

```bash
# Query via API
curl "http://localhost:8080/api/v1/calendar/golden?region=US&start_date=2026-01-01&end_date=2026-12-31" \
  -H "X-Tenant-ID: $TENANT_ID"

# Or directly verify in database
psql -h 100.84.126.19 -U usice_app -d alpha << 'SQL'
SELECT calendar_date, is_business_day, holiday_name, confidence_score
FROM edm.mdm_calendar_golden
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
AND region_code = 'US'
LIMIT 10;
SQL
```

### Test 4: Verify Redpanda Event Publishing

```bash
# Access Redpanda console
open http://localhost:8888

# Or list topics via CLI
docker-compose -f docker-compose.mdm.yml exec redpanda \
  rpk topic list
```

### Test 5: Check Logging

```bash
# Semantic Engine logs
docker-compose -f docker-compose.mdm.yml logs semantic-engine | head -50

# API Gateway logs
docker-compose -f docker-compose.mdm.yml logs api-gateway | head -50

# Check for errors
docker-compose -f docker-compose.mdm.yml logs | grep -i error
```

### Test 6: Run Integration Tests

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Run all MDM tests
go test ./internal/mdm -v

# Run specific test
go test ./internal/mdm -run TestIngestionOrchestrator_NagerDateSource -v

# Run benchmarks
go test ./internal/mdm -bench=. -benchmem
```

---

## Operations

### Activate a Commercial Source

1. **Via Ops Console (Frontend)**
   - Navigate to http://localhost:3000
   - Go to "MDM Calendar Management"
   - Find "TradingHours" in the source list
   - Click "Activate" button

2. **Via API**
   ```bash
   SOURCE_ID="<uuid-from-database>"
   curl -X PATCH "http://localhost:8080/api/v1/mdm/sources/$SOURCE_ID/activate" \
     -H "X-User-Role: global_ops"
   ```

3. **Verify Activation**
   ```bash
   psql -h 100.84.126.19 -U usice_app -d alpha << 'SQL'
   SELECT source_name, is_active FROM edm.mdm_source_registry WHERE source_name = 'TradingHours';
   SQL
   ```

### Monitor Ingestion Jobs

```bash
# Check recent jobs
psql -h 100.84.126.19 -U usice_app -d alpha << 'SQL'
SELECT id, job_type, status, records_ingested, error_message, started_at
FROM edm.mdm_ingestion_jobs
ORDER BY started_at DESC
LIMIT 10;
SQL
```

### Resolve Conflicts

1. **Identify conflicts**
   ```bash
   curl "http://localhost:8080/api/v1/mdm/conflicts?tenant_id=$TENANT_ID"
   ```

2. **Access Stewardship Queue**
   ```bash
   psql -h 100.84.126.19 -U usice_app -d alpha << 'SQL'
   SELECT id, issue_type, description, status
   FROM edm.mdm_stewardship_queue
   WHERE status = 'PENDING'
   ORDER BY priority DESC;
   SQL
   ```

### View Data Quality Metrics

```bash
psql -h 100.84.126.19 -U usice_app -d alpha << 'SQL'
-- Refresh materialized view
REFRESH MATERIALIZED VIEW CONCURRENTLY edm.mdm_calendar_coverage;

-- View coverage by region
SELECT tenant_id, region_code, total_days, business_days, high_confidence_pct
FROM edm.mdm_calendar_coverage
ORDER BY high_confidence_pct DESC;
SQL
```

### Enable Logging & Debugging

```bash
# Set log level to DEBUG
docker-compose -f docker-compose.mdm.yml exec semantic-engine \
  env LOG_LEVEL=debug

# View current logs
docker-compose -f docker-compose.mdm.yml logs -f --tail=100 semantic-engine
```

### Restart a Service

```bash
# Restart semantic engine
docker-compose -f docker-compose.mdm.yml restart semantic-engine

# Restart all services
docker-compose -f docker-compose.mdm.yml restart
```

### View Admin UIs

- **Redpanda Console:** http://localhost:8888
- **Postgres Adminer:** http://localhost:8889
- **Frontend Ops Console:** http://localhost:3000

---

## Troubleshooting

### "Cannot connect to Postgres"

```bash
# Verify Postgres is running on 100.84.126.19
ssh user@100.84.126.19 'sudo systemctl status postgresql'

# Check firewall
sudo ufw status
sudo ufw allow from 172.28.0.0/16 to any port 5432

# Test connectivity from Docker
docker-compose -f docker-compose.mdm.yml exec api-gateway \
  psql postgresql://usice_app@100.84.126.19:5432/usice_mdm
```

### "Redpanda not starting"

```bash
# Check Redpanda logs
docker-compose -f docker-compose.mdm.yml logs redpanda | tail -50

# Increase RAM allocation
docker-compose -f docker-compose.mdm.yml down
# Edit docker-compose.mdm.yml: increase memory from 1G to 2G
docker-compose -f docker-compose.mdm.yml up -d redpanda
```

### "Sources not ingesting data"

```bash
# Check semantic engine logs
docker-compose -f docker-compose.mdm.yml logs semantic-engine | grep -i error

# Verify Python services are healthy
curl http://localhost:8000/health
curl http://localhost:8001/health

# Check if cron schedule is triggering
docker-compose -f docker-compose.mdm.yml exec semantic-engine \
  grep "Ingestion cycle" /proc/*/environ  # Check environment
```

### "API returning 500 errors"

```bash
# Check API gateway logs
docker-compose -f docker-compose.mdm.yml logs api-gateway | tail -50

# Verify database credentials
grep "DB_PASSWORD" .env.mdm

# Test database connection directly
psql -h 100.84.126.19 -U usice_app -d usice_mdm -c "SELECT 1;"
```

---

## Next Steps

1. **Customize Survivorship Rules** - Edit DSL policies in Starlark and compile to WASM
2. **Activate Commercial Sources** - When ready, toggle TradingHours/EODHD via Ops Console
3. **Set Up Monitoring** - Configure Prometheus/Grafana for metrics
4. **Extend Semantic Model** - Add new Business Objects for other master data (Security, Price, Portfolio)
5. **Production Hardening** - Add authentication, encryption, backups

---

## Support & Documentation

- **Architecture:** Usice Architecture (Sections 2-7)
- **API Docs:** Generated from OpenAPI spec at `/openapi.yaml`
- **Database Schema:** `/schema/001_mdm_init.sql`
- **Tests:** `/internal/mdm/orchestrator_test.go`

