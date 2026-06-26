# Phase 1: Free Calendar Sources Deployment Guide

## Objective
Get calendar data flowing from free sources (Nager.Date, OpenHolidays, Workalendar, Holidays PyPI) through the MDM system into the golden records table within **2-4 hours**.

---

## Prerequisites Check

Before starting, verify you have:

```bash
# 1. Postgres running on external desktop
psql -h 100.84.126.19 -U postgres -d alpha -c "SELECT * FROM edm.mdm_source_registry LIMIT 1;"
# Should return: (should have 4+ rows with sources)

# 2. Docker running
docker --version
# Should return: Docker version 20.10+

# 3. Python 3.11+
python3 --version
# Should return: Python 3.11+
```

If any fail, stop and fix before continuing.

---

## Step 1: Verify Database Configuration (5 min)

The schema should already exist from the consolidation work. Verify:

```bash
# Connect to alpha database
psql -h 100.84.126.19 -U postgres -d alpha

# Check schema exists
\dn edm
# Should show: edm | public

# Check tables exist
\dt edm.mdm_*
# Should list all 14 tables

# Check source registry is seeded
SELECT source_name, is_active, priority_score FROM edm.mdm_source_registry ORDER BY priority_score;

# Expected output:
# source_name | is_active | priority_score
# NagerDate | t | 4
# OpenHolidays | t | 4
# Workalendar | t | 3
# HolidaysPyPI | t | 3

# If not seeded, run:
INSERT INTO edm.mdm_source_registry (source_name, source_type, endpoint_url, is_active, priority_score, confidence_base) VALUES
('NagerDate', 'API', 'https://date.nager.at/api/v3', true, 4, 70),
('OpenHolidays', 'API', 'https://openholidaysapi.org', true, 4, 70),
('Workalendar', 'PYTHON_SERVICE', 'http://workalendar-service:8000', true, 3, 65),
('HolidaysPyPI', 'PYTHON_SERVICE', 'http://holidays-service:8001', true, 3, 65),
('TradingHours', 'API', 'https://api.tradinghours.com/v1', false, 1, 95),
('EODHD', 'API', 'https://eodhd.com/api', false, 2, 90),
('Xignite', 'API', 'https://api.xignite.com', false, 2, 90),
('Finnhub', 'API', 'https://finnhub.io/api', false, 2, 85);
```

---

## Step 2: Configure Environment (5 min)

Create `.env.mdm` in calendar-service directory:

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

cat > .env.mdm << 'EOF'
# Database Configuration
DB_HOST=100.84.126.19
DB_PORT=5432
DB_NAME=alpha
DB_USER=usice_app
DB_PASSWORD=change_me_in_production
DB_SCHEMA=edm
DB_SSL_MODE=disable

# MDM Service Configuration
MDM_ENABLED=true
MDM_SERVICE_URL=http://localhost:8080
MDM_CACHE_TTL=5m
MDM_TIMEOUT=10s
MDM_FAILURE_MODE=fallback

# Python Services
WORKALENDAR_SERVICE_URL=http://workalendar-service:8000
HOLIDAYS_SERVICE_URL=http://holidays-service:8001

# API Configuration
PORT=8080
LOG_LEVEL=info

# Ingestion Configuration
INGESTION_SCHEDULE="0 2 * * *"  # Daily at 2 AM UTC
INGESTION_REGIONS=US,GB,JP,DE,FR
INGESTION_YEAR=2026

# Docker Network
DOCKER_NETWORK=usice-network
EOF

echo "✅ .env.mdm created"
```

---

## Step 3: Build Docker Images (10 min)

```bash
# Build Python microservices
docker build -t semlayer/workalendar-adapter:latest ./services/workalendar-adapter
docker build -t semlayer/holidays-adapter:latest ./services/holidays-adapter

# Verify builds
docker images | grep semlayer

# Expected output:
# semlayer/workalendar-adapter   latest
# semlayer/holidays-adapter       latest
```

If build fails:
- Check that `Dockerfile` exists in each service directory
- Verify Docker daemon is running: `docker ps`

---

## Step 4: Start Docker Compose Stack (15 min)

```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Start all services
docker-compose -f docker-compose.mdm.yml up -d

# Watch container startup
watch -n 1 'docker-compose -f docker-compose.mdm.yml ps'

# Expected output (takes ~30 seconds):
# NAME                   STATUS              PORTS
# redpanda-1            Up 30s              0.0.0.0:9092->9092/tcp
# schema-registry       Up 25s              0.0.0.0:8081->8081/tcp
# workalendar-adapter   Up 20s              0.0.0.0:8000->8000/tcp
# holidays-service      Up 20s              0.0.0.0:8001->8001/tcp
# semantic-engine       Up 15s              (no ports - internal)
# api-gateway           Up 10s              0.0.0.0:8080->8080/tcp

# Exit watch mode (Ctrl+C)
```

### **Troubleshooting Docker Issues**

If containers fail to start:

```bash
# View logs for specific service
docker-compose -f docker-compose.mdm.yml logs workalendar-adapter
docker-compose -f docker-compose.mdm.yml logs semantic-engine

# If port conflict (e.g., 8000 already in use):
lsof -i :8000  # See what's using port
kill -9 <PID>  # Kill the process

# Reset completely
docker-compose -f docker-compose.mdm.yml down
docker volume prune -f
docker-compose -f docker-compose.mdm.yml up -d
```

---

## Step 5: Verify Services Are Healthy (10 min)

```bash
# Test each service

# 1. Workalendar Adapter
curl http://localhost:8000/health
# Expected: {"status": "healthy", "service": "workalendar-adapter"}

# 2. Holidays PyPI Adapter
curl http://localhost:8001/health
# Expected: {"status": "healthy", ...}

# 3. API Gateway (should fail until semantic-engine is ready)
curl http://localhost:8080/health
# Expected: {"status": "healthy"} or connection refused initially (OK)

# 4. Redpanda (message broker)
curl http://localhost:9644/admin/brokers
# Expected: JSON with broker info

# All services healthy? Continue to next step.
```

---

## Step 6: Trigger Manual Ingestion (10 min)

Test the ingestion pipeline manually:

```bash
# Execute ingestion for 2026, US & GB regions
curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "regions": ["US", "GB"],
    "year": 2026,
    "force_refresh": false
  }'

# Expected response (background job):
# {
#   "job_id": "uuid...",
#   "status": "QUEUED",
#   "message": "Ingestion cycle started"
# }

# Monitor job status
curl http://localhost:8080/api/v1/mdm/jobs/$(JOB_ID) \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Watch logs for progress
docker-compose -f docker-compose.mdm.yml logs -f semantic-engine | grep -i "ingest\|holiday\|source"
```

---

## Step 7: Verify Data in Database (10 min)

After ingestion completes (2-5 minutes), check the data:

```bash
# Connect to database
psql -h 100.84.126.19 -U usice_app -d alpha

# 1. Check source records were created
SELECT source_name, COUNT(*) as count 
FROM edm.mdm_calendar_source 
GROUP BY source_name;

# Expected: 4 rows (one per active source)
# nager_date | 50+ records
# open_holidays | 40+ records
# workalendar | 30+ records
# holidays_pypi | 50+ records

# 2. Check golden records (survivorship rules applied)
SELECT region_code, COUNT(*) as business_days, 
       COUNT(CASE WHEN NOT is_business_day THEN 1 END) as holidays
FROM edm.mdm_calendar_golden 
WHERE calendar_date BETWEEN '2026-01-01' AND '2026-12-31'
  AND tenant_id = '00000000-0000-0000-0000-000000000001'
GROUP BY region_code
ORDER BY region_code;

# Expected: US, GB with proper counts (~250 business days each)

# 3. Check for conflicts (high-priority sources disagreed)
SELECT COUNT(*) as conflicts
FROM edm.mdm_stewardship_queue 
WHERE status = 'PENDING'
  AND created_at > NOW() - INTERVAL '10 minutes';

# Expected: 0-5 (very few conflicts from trusted sources)

# 4. Verify lineage (audit trail)
SELECT semantic_term_name, rule_applied, COUNT(*) 
FROM edm.mdm_calendar_lineage 
GROUP BY semantic_term_name, rule_applied;

# Expected: Shows which rules were applied to different fields

# 5. Sample a day to verify data quality
SELECT calendar_date, is_business_day, holiday_name, 
       confidence_score, source_system 
FROM edm.mdm_calendar_golden 
WHERE calendar_date = '2026-12-25'  -- Christmas
  AND region_code = 'US'
  AND tenant_id = '00000000-0000-0000-0000-000000000001';

# Expected: December 25 marked as non-business day (is_business_day = false)
```

---

## Step 8: Test Calendar Service Integration (5 min)

```bash
# 1. Get business days for date range
curl "http://localhost:8080/api/v1/calendar/business-days?region=US&start_date=2026-01-01&end_date=2026-01-31" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Expected: Array of business days in January 2026

# 2. Check if specific date is business day
curl "http://localhost:8080/api/v1/calendar/is-business-day?date=2026-12-25&region=US" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Expected: false (Christmas is not a business day)

# 3. Get holidays for month
curl "http://localhost:8080/api/v1/calendar/holidays?region=US&month=12&year=2026" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Expected: Array of holidays including Christmas, Thanksgiving, etc.
```

---

## Step 9: Set Up Scheduled Ingestion (5 min)

For ongoing automated ingestion:

```bash
# Option A: Cron job on host machine
# Add to crontab (-e)
0 2 * * * curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"regions":["US","GB","JP","DE","FR"],"year":2026}'

# Option B: Docker container with scheduler built-in
# Already configured in docker-compose.mdm.yml

# Verify scheduler is running
docker-compose -f docker-compose.mdm.yml logs semantic-engine | grep -i schedule
```

---

## Monitoring & Verification Dashboard

Create a monitoring script to watch the system:

```bash
# Create monitoring script
cat > monitor_mdm.sh << 'EOF'
#!/bin/bash
while true; do
  clear
  echo "=== MDM System Status ==="
  echo ""
  echo "1. Docker Containers:"
  docker-compose -f docker-compose.mdm.yml ps | tail -n +2
  echo ""
  echo "2. Service Health:"
  echo -n "Workalendar: "
  curl -s http://localhost:8000/health | jq -r '.status' || echo "FAILED"
  echo -n "Holidays: "
  curl -s http://localhost:8001/health | jq -r '.status' || echo "FAILED"
  echo -n "API Gateway: "
  curl -s http://localhost:8080/health | jq -r '.status' || echo "FAILED"
  echo ""
  echo "3. Database Records:"
  psql -h 100.84.126.19 -U usice_app -d alpha -t -c \
    "SELECT source_name, COUNT(*) FROM edm.mdm_calendar_source GROUP BY source_name ORDER BY source_name;" 2>/dev/null || echo "DB connection failed"
  echo ""
  echo "4. Golden Records:"
  psql -h 100.84.126.19 -U usice_app -d alpha -t -c \
    "SELECT COUNT(*) as total_golden_records FROM edm.mdm_calendar_golden WHERE tenant_id = '00000000-0000-0000-0000-000000000001';" 2>/dev/null || echo "DB connection failed"
  echo ""
  echo "Press Ctrl+C to exit"
  sleep 10
done
EOF

chmod +x monitor_mdm.sh
./monitor_mdm.sh
```

---

## Success Criteria - All Must Pass ✅

- [ ] 4 free sources registered in `mdm_source_registry`
- [ ] All Docker containers running and healthy
- [ ] At least 300+ source records in `mdm_calendar_source`
- [ ] At least 250+ golden records in `mdm_calendar_golden` (per region)
- [ ] Lineage table populated with rule applications
- [ ] Calendar service successfully queries MDM
- [ ] Zero conflicts for trusted sources (< 5 in stewardship queue)
- [ ] Performance: Ingestion completes in < 2 minutes
- [ ] API endpoints responding with correct data

---

## Common Issues & Solutions

### Issue: Docker containers fail to start
**Solution:** Check logs with `docker-compose logs [service]`. Most common: port already in use.

### Issue: "Connection refused" when connecting to Postgres
**Solution:** Verify Postgres is running on 100.84.126.19 and firewall allows 172.28.0.0/16.

### Issue: Ingestion runs but creates zero records
**Solution:** Check semantic-engine logs for API call failures. Verify source endpoints are reachable from Docker.

### Issue: High confidence_score but data seems wrong
**Solution:** Review the survivorship rules in WASM. Check lineage table to see which rule won.

---

## Next Steps (Phase 2)

Once Phase 1 is validated:

1. **Add event streaming** - Publish calendar changes to Redpanda
2. **Build React UI** - Display calendar from MDM
3. **Add commercial sources** - Toggle TradingHours, EODHD on in UI
4. **Production hardening** - Monitoring, alerting, backups

---

## Support

**Stuck?** Check these in order:
1. Are all Docker containers running? → `docker-compose ps`
2. Can you reach the endpoints? → `curl http://localhost:8000/health`
3. Does the database have data? → `SELECT COUNT(*) FROM edm.mdm_calendar_source;`
4. Are the logs telling you what's wrong? → `docker-compose logs semantic-engine`
5. Is the calendar-service binary actually running? → `ps aux | grep calendar-service`

---

**Estimated Time to Complete:** 2-4 hours  
**Success Rate:** ~95% if Python/Docker already installed  
**Go/No-Go:** Proceed to Phase 2 once all criteria pass ✅
