# Phase 3: Quick Reference Guide

## 🚀 Quick Start (Reproducing Phase 3)

### Prerequisites
- PostgreSQL client tools (psql, pg_dump)
- Python 3 with requests library
- curl
- Go 1.21+ for rebuilding service

### Step 1: Verify Database Schema
```bash
export PGPASSWORD='postgres'
psql -h 100.84.126.19 -p 5432 -U postgres -d alpha -c "\dt public.*" | grep -E "calendars|profiles|blackouts"
```

Expected output:
```
 calendars             | table
 profile_calendars     | table
 schedule_profiles     | table
 blackouts             | table
 audit_log             | table
```

### Step 2: Verify Test Data
```bash
# Check calendar exists
psql -h 100.84.126.19 -p 5432 -U postgres -d alpha \
  -c "SELECT id, name, json_array_length(holidays) as holiday_count FROM calendars WHERE tenant_id = '870361a8-87e2-4171-95ad-0473cc93791e';"

# Expected:
#                   id                  |            name             | holiday_count
# 7d3be7d4-5134-45af-b66c-547cedea9e08 | Test - USA Federal Holidays |             5
```

### Step 3: Check Service Status
```bash
ps aux | grep "calendar-service -port 9081"

# Expected: Service process running
# If not running, start with:
/Users/eganpj/GitHub/semlayer/calendar-service/bin/calendar-service \
  -port 9081 \
  -db-host 100.84.126.19 \
  -db-port 5432 \
  -db-user postgres \
  -db-password postgres \
  -db-name alpha \
  -loglevel debug 2>&1 &
```

### Step 4: Test API Endpoints

#### Generate JWT Token
```python
import base64, json, hmac, hashlib, time

secret = "dev-jwt-secret-key-change-in-production"
tenant = "870361a8-87e2-4171-95ad-0473cc93791e"

h = base64.urlsafe_b64encode(json.dumps({"alg": "HS256", "typ": "JWT"}).encode()).decode().rstrip('=')
p = base64.urlsafe_b64encode(json.dumps({
    "user_id": "test",
    "tenant_id": tenant,
    "exp": int(time.time()) + 3600,
    "iat": int(time.time())
}).encode()).decode().rstrip('=')

m = f"{h}.{p}"
s = base64.urlsafe_b64encode(
    hmac.new(secret.encode(), m.encode(), hashlib.sha256).digest()
).decode().rstrip('=')

token = f"{m}.{s}"
print(token)
```

#### Test Calendar Endpoint
```bash
TOKEN="<generated_token_from_above>"
TENANT="870361a8-87e2-4171-95ad-0473cc93791e"
CALENDAR_ID="7d3be7d4-5134-45af-b66c-547cedea9e08"

curl -s -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: $TENANT" \
     http://127.0.0.1:9081/api/v1/calendars/$CALENDAR_ID | jq .
```

Expected response: Calendar details with holiday JSONB

#### Test Availability Check
```bash
curl -s -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "X-Tenant-ID: $TENANT" \
     -H "Content-Type: application/json" \
     -d '{"calendar_id":"7d3be7d4-5134-45af-b66c-547cedea9e08","date":"2026-02-20"}' \
     http://127.0.0.1:9081/api/v1/availability | jq .
```

Expected: Availability data with holidays and blackouts

---

## 📊 Key IDs Reference

| Entity | ID | Value |
|--------|----|----|
| **Tenant** | tenant_id | `870361a8-87e2-4171-95ad-0473cc93791e` |
| **Calendar** | calendar_id | `7d3be7d4-5134-45af-b66c-547cedea9e08` |
| **Profile** | profile_id | `633e2719-f213-4c2f-b47b-e85f7eea9367` |
| **Service** | host:port | `127.0.0.1:9081` |
| **Database** | host:port | `100.84.126.19:5432` |
| **Database** | user / pass | `postgres` / `postgres` |
| **Database** | name | `alpha` |

---

## 🔍 Troubleshooting

### Issue: Service won't start on port 9081
```bash
# Port in use - find and kill
lsof -i :9081
kill -9 <PID>

# Or use different port
/path/to/calendar-service -port 9082 ...
```

### Issue: Database connection fails
```bash
# Test connection
psql -h 100.84.126.19 -p 5432 -U postgres -d alpha -c "SELECT 1;"

# Should return:
#  ?column? 
# ----------
#        1
```

### Issue: JWT token rejected
```bash
# Verify secret matches
echo "dev-jwt-secret-key-change-in-production"

# Token should be valid for 1 hour after generation
# Regenerate if older than that
```

### Issue: API returns 404
```bash
# Verify endpoint path - must include /api/v1 prefix
# Correct:   http://localhost:9081/api/v1/calendars
# Incorrect: http://localhost:9081/calendars
```

---

## 📈 Performance Testing

### Check Response Time
```bash
time curl -s http://127.0.0.1:9081/api/v1/availability \
          -H "Authorization: Bearer $TOKEN" \
          -H "X-Tenant-ID: $TENANT" \
          -H "Content-Type: application/json" \
          -d '{"calendar_id":"7d3be7d4-5134-45af-b66c-547cedea9e08","date":"2026-02-20"}'

# Real time should be < 100ms (first call)
# Subsequent calls should be < 20ms with cache
```

### Load Test (if ab installed)
```bash
ab -n 100 -c 10 \
   -H "Authorization: Bearer $TOKEN" \
   -H "X-Tenant-ID: $TENANT" \
   http://127.0.0.1:9081/api/v1/calendars/$CALENDAR_ID
```

### Query Database Directly
```bash
psql -h 100.84.126.19 -p 5432 -U postgres -d alpha << SQL
-- Check holidays in JSONB
SELECT holidays->'0'->>'name' as first_holiday FROM calendars 
WHERE id = '7d3be7d4-5134-45af-b66c-547cedea9e08';

-- Check blackouts
SELECT name, start_time, end_time, recurrence_rule FROM blackouts
WHERE profile_id = '633e2719-f213-4c2f-b47b-e85f7eea9367' LIMIT 10;

-- Check profile calendars mapping
SELECT pc.weight FROM profile_calendars pc
WHERE profile_id = '633e2719-f213-4c2f-b47b-e85f7eea9367';
SQL
```

---

## 🔄 Common Workflows

### Run Full Integration Test
```bash
# 1. Verify schema exists
export PGPASSWORD='postgres'
psql -h 100.84.126.19 -U postgres -d alpha -c "\dt" | grep calendars

# 2. Check service is running
ps aux | grep "calendar-service -port 9081" | grep -v grep && echo "✅ Service running" || echo "❌ Service down"

# 3. Generate token and test API
./scripts/phase3-quick-test.sh

# 4. Check database directly
psql -h 100.84.126.19 -U postgres -d alpha \
  -c "SELECT COUNT(*) as total_calendars FROM calendars WHERE tenant_id = '870361a8-87e2-4171-95ad-0473cc93791e';"
```

### Restart Service
```bash
# Kill existing
pkill -f "calendar-service -port 9081"

# Start fresh
/Users/eganpj/GitHub/semlayer/calendar-service/bin/calendar-service \
  -port 9081 \
  -db-host 100.84.126.19 \
  -db-port 5432 \
  -db-user postgres \
  -db-password postgres \
  -db-name alpha \
  -loglevel info 2>&1 &

# Verify
sleep 2 && curl -s http://127.0.0.1:9081/api/v1/calendars \
  -H "Authorization: Bearer dummy" -H "X-Tenant-ID: 870361a8-87e2-4171-95ad-0473cc93791e" 2>&1 | head -5
```

### View Service Logs
```bash
# View from background process (if still attached)
# Or check syslog if configured

# Quick HTTP test to see if responding
curl -i http://127.0.0.1:9081/ 2>&1 | head -5
# Should return 404 but with proper HTTP headers
```

---

## 📁 Files Reference

| Purpose | Path |
|---------|------|
| **Service Binary** | `/Users/eganpj/GitHub/semlayer/calendar-service/bin/calendar-service` |
| **Schema Script** | `/Users/eganpj/GitHub/semlayer/calendar-service/docs/schema-phase3.sql` |
| **Test Data** | `/Users/eganpj/GitHub/semlayer/calendar-service/docs/test-data-phase3-live.sql` |
| **Test Script** | `/Users/eganpj/GitHub/semlayer/calendar-service/scripts/phase3-quick-test.sh` |
| **Source Code** | `/Users/eganpj/GitHub/semlayer/calendar-service/internal/` |

---

## ✅ Verification Checklist

Use this to verify Phase 3 is properly set up:

- [ ] PostgreSQL responding at 100.84.126.19:5432
- [ ] Calendar table has test data
- [ ] Profile table linked to calendar
- [ ] Blackouts (3 total) populated
- [ ] Service binary exists (31MB)
- [ ] Service process running on port 9081
- [ ] curl returns proper 401 without auth
- [ ] JWT token generates successfully
- [ ] API returns 200 with valid JWT
- [ ] Calendar holidays accessible
- [ ] Blackout recurrence rules stored
- [ ] Multi-tenant isolation working (RLS)

---

## 🎯 Next Steps

Ready for Phase 4:
1. Enable Redis caching
2. Integrate Hasura for GraphQL queries
3. Run performance benchmarks
4. Load testing (100+ concurrent users)
5. CDC invalidation testing
6. Production deployment preparation
