# 🚀 MDM Phase 1: Quick Start (Copy & Paste Commands)

**Goal:** Get free calendar sources flowing in **2 hours max**

---

## PRE-FLIGHT CHECK (5 min)

```bash
# 1. Verify Postgres running
psql -h 100.84.126.19 -U postgres -d alpha -c "SELECT 1;" && echo "✅ Postgres OK" || echo "❌ Postgres FAILED"

# 2. Verify Docker running
docker ps >/dev/null 2>&1 && echo "✅ Docker OK" || echo "❌ Docker FAILED"

# 3. Verify Python 3.11+
python3 --version | grep -E "3\.(11|12)" && echo "✅ Python OK" || echo "❌ Python FAILED"

# 4. Verify you're in correct directory
pwd | grep -q "calendar-service" && echo "✅ Directory OK" || echo "❌ Not in calendar-service dir"
```

---

## SETUP (10 min)

```bash
# 1. Create .env.mdm
cat > .env.mdm << 'EOF'
DB_HOST=100.84.126.19
DB_PORT=5432
DB_NAME=alpha
DB_USER=usice_app
DB_PASSWORD=change_me_in_production
MDM_ENABLED=true
PORT=8080
LOG_LEVEL=info
EOF

# 2. Seed database (if not already seeded)
psql -h 100.84.126.19 -U postgres -d alpha << 'SQL'
INSERT INTO edm.mdm_source_registry (source_name, source_type, endpoint_url, is_active, priority_score, confidence_base) 
VALUES
('NagerDate', 'API', 'https://date.nager.at/api/v3', true, 4, 70),
('OpenHolidays', 'API', 'https://openholidaysapi.org', true, 4, 70),
('Workalendar', 'PYTHON_SERVICE', 'http://workalendar-service:8000', true, 3, 65),
('HolidaysPyPI', 'PYTHON_SERVICE', 'http://holidays-service:8001', true, 3, 65),
('TradingHours', 'API', 'https://api.tradinghours.com/v1', false, 1, 95),
('EODHD', 'API', 'https://eodhd.com/api', false, 2, 90),
('Xignite', 'API', 'https://api.xignite.com', false, 2, 90),
('Finnhub', 'API', 'https://finnhub.io/api', false, 2, 85)
ON CONFLICT (source_name) DO NOTHING;
SQL
echo "✅ Sources seeded"

# 3. Build Docker images
docker build -t semlayer/workalendar-adapter:latest services/workalendar-adapter && echo "✅ Workalendar built" || echo "❌ Workalendar build FAILED"
docker build -t semlayer/holidays-adapter:latest services/holidays-adapter && echo "✅ Holidays built" || echo "❌ Holidays build FAILED"
```

---

## START SERVICES (5 min)

```bash
# 1. Start Docker stack
docker-compose -f docker-compose.mdm.yml up -d && echo "✅ Services started" || echo "❌ Docker started FAILED"

# 2. Wait for services to be ready (takes ~30 seconds)
echo "⏳ Waiting for services to be healthy..."
sleep 30

# 3. Check all containers are running
docker-compose -f docker-compose.mdm.yml ps
# Should show 9 containers with "Up" status
```

---

## HEALTH CHECKS (5 min)

```bash
# Test each service
echo "Testing Workalendar..."
curl -s http://localhost:8000/health | jq . || echo "❌ Workalendar failed"

echo "Testing Holidays..."
curl -s http://localhost:8001/health | jq . || echo "❌ Holidays failed"

echo "Testing API Gateway..."
curl -s http://localhost:8080/health | jq . || echo "❌ API Gateway failed"

echo ""
echo "All healthy? Continue to INGEST step."
```

---

## INGEST CALENDAR DATA (2 min)

```bash
# Trigger ingestion for US & GB in 2026
curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "regions": ["US", "GB"],
    "year": 2026,
    "force_refresh": false
  }' | jq .

# Expected response:
# {
#   "job_id": "...",
#   "status": "QUEUED"
# }

echo "⏳ Ingestion running (watch logs)..."
docker-compose -f docker-compose.mdm.yml logs -f semantic-engine | grep -E "(ingest|holiday|source)" &
TAIL_PID=$!
sleep 60
kill $TAIL_PID 2>/dev/null
```

---

## VERIFY DATA (3 min)

```bash
# 1. Source records
psql -h 100.84.126.19 -U usice_app -d alpha -c "
SELECT source_name, COUNT(*) as count 
FROM edm.mdm_calendar_source 
GROUP BY source_name 
ORDER BY count DESC;
"
# Expected: 4 rows with 30+ records each

# 2. Golden records
psql -h 100.84.126.19 -U usice_app -d alpha -c "
SELECT COUNT(*) as golden_records 
FROM edm.mdm_calendar_golden 
WHERE tenant_id = '00000000-0000-0000-0000-000000000001';
"
# Expected: 250+ records

# 3. Sample data
psql -h 100.84.126.19 -U usice_app -d alpha -c "
SELECT calendar_date, is_business_day, holiday_name 
FROM edm.mdm_calendar_golden 
WHERE calendar_date = '2026-12-25' 
AND region_code = 'US'
AND tenant_id = '00000000-0000-0000-0000-000000000001';
"
# Expected: 2026-12-25 | false | Christmas
```

---

## TEST CALENDAR API (2 min)

```bash
# 1. Is Dec 25 a business day?
curl "http://localhost:8080/api/v1/calendar/is-business-day?date=2026-12-25&region=US" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
# Expected: "is_business_day": false

# 2. Get January business days
curl "http://localhost:8080/api/v1/calendar/business-days?region=US&start_date=2026-01-01&end_date=2026-01-31" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq '.business_days | length'
# Expected: 21 (approximately)

# 3. Get holidays
curl "http://localhost:8080/api/v1/calendar/holidays?region=US&month=12&year=2026" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq '.holidays | length'
# Expected: 2-3 (Christmas, possibly others)
```

---

## SUCCESS CRITERIA ✅

If ALL of these pass, Phase 1 is COMPLETE:

```bash
# Run this validation script
cat > validate_phase1.sh << 'VALIDATE'
#!/bin/bash
ERRORS=0

# 1. Docker containers running
echo -n "1. Docker containers: "
RUNNING=$(docker-compose -f docker-compose.mdm.yml ps -q | wc -l)
if [ $RUNNING -eq 9 ]; then
  echo "✅ PASS"
else
  echo "❌ FAIL (found $RUNNING, expected 9)"
  ERRORS=$((ERRORS + 1))
fi

# 2. Source records
echo -n "2. Source records: "
COUNT=$(psql -h 100.84.126.19 -U usice_app -d alpha -t -c "SELECT COUNT(*) FROM edm.mdm_calendar_source;")
if [ $COUNT -gt 100 ]; then
  echo "✅ PASS ($COUNT records)"
else
  echo "❌ FAIL ($COUNT records, expected 100+)"
  ERRORS=$((ERRORS + 1))
fi

# 3. Golden records
echo -n "3. Golden records: "
COUNT=$(psql -h 100.84.126.19 -U usice_app -d alpha -t -c "SELECT COUNT(*) FROM edm.mdm_calendar_golden WHERE tenant_id = '00000000-0000-0000-0000-000000000001';")
if [ $COUNT -gt 200 ]; then
  echo "✅ PASS ($COUNT records)"
else
  echo "❌ FAIL ($COUNT records, expected 200+)"
  ERRORS=$((ERRORS + 1))
fi

# 4. API responding
echo -n "4. API Gateway: "
STATUS=$(curl -s http://localhost:8080/health | jq -r '.status')
if [ "$STATUS" = "healthy" ]; then
  echo "✅ PASS"
else
  echo "❌ FAIL (status: $STATUS)"
  ERRORS=$((ERRORS + 1))
fi

# 5. Christmas is not business day
echo -n "5. Christmas check: "
RESULT=$(curl -s "http://localhost:8080/api/v1/calendar/is-business-day?date=2026-12-25&region=US" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq -r '.is_business_day')
if [ "$RESULT" = "false" ]; then
  echo "✅ PASS"
else
  echo "❌ FAIL (Christmas is showing as: $RESULT)"
  ERRORS=$((ERRORS + 1))
fi

echo ""
if [ $ERRORS -eq 0 ]; then
  echo "🎉 PHASE 1 COMPLETE - ALL CRITERIA PASSED"
  echo "Next step: Read COMPLETE_MDM_ROADMAP.md for Phase 2"
  exit 0
else
  echo "❌ PHASE 1 INCOMPLETE - $ERRORS CRITERIA FAILED"
  echo "Review failures above and retry"
  exit 1
fi
VALIDATE

chmod +x validate_phase1.sh
./validate_phase1.sh
```

---

## TROUBLESHOOTING

| Issue | Command | Solution |
|-------|---------|----------|
| Containers won't start | `docker-compose logs [service]` | Check port conflicts: `lsof -i :[port]` |
| Postgres no connect | `psql -h 100.84.126.19 -U postgres -d alpha` | Verify firewall: `sudo ufw allow 5432` |
| No data after ingest | `docker-compose logs semantic-engine` | Check source endpoints in logs |
| Port already in use | `lsof -i :[port]` | Kill existing process: `kill -9 [PID]` |
| Python install missing | `python3 -m pip install -r services/workalendar-adapter/requirements.txt` | Install dependencies |

---

## NEXT STEPS

Once all criteria pass:

```bash
# 1. Stop current setup (optional)
docker-compose -f docker-compose.mdm.yml down

# 2. Read roadmap for Phase 2
cat COMPLETE_MDM_ROADMAP.md

# 3. Start Phase 2 (Redpanda event streaming)
# Follows similar pattern with new implementation
```

---

**Status:** Ready to start? → Begin at PRE-FLIGHT CHECK section above ☝️

**Time Estimate:** 2 hours total  
**Difficulty:** Medium (mostly copy-paste commands)  
**Support:** Check PHASE_1_DEPLOYMENT_GUIDE.md for detailed explanations
