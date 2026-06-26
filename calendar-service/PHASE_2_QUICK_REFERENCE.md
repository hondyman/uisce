# ⚡ Phase 2 Quick Reference

## Status: ✅ COMPLETE & COMPILING

**Build**: `✅ PASSING` - 0 errors, 0 warnings  
**Lines**: 572 lines total (added ~300 lines of Phase 2 code)  
**Date**: February 18, 2026

---

## Phase 2: What Was Implemented

### 1. Hasura GraphQL Queries ✅
- `fetchScheduleProfile()` - Get profile + linked calendars
- `fetchHolidaysForCalendars()` - Get holidays from JSONB
- `fetchAndExpandBlackouts()` - Get and expand recurring blackouts

### 2. Recurring Blackout Expansion ✅
- `expandRecurringBlackout()` - Using rrule-go (RFC 5545)
- Supports: FREQ=DAILY, WEEKLY, MONTHLY, YEARLY with complex patterns
- Configurable time range expansion

### 3. Resolution Pipeline ✅
```
computeResolvedProfile()
├─ Fetch profile + calendars
├─ Fetch holidays (JSONB)
├─ Fetch & expand blackouts
├─ Apply conflict resolution
└─ Convert to cache format (time.Time + TimeRange)
```

### 4. Helper Functions ✅
- `applyConflictResolution()` - Route by strategy
- `deduplicateHolidays()` - Merge by severity
- `deduplicateBlackouts()` - Merge by severity
- `isHigherSeverity()` - Severity comparison

---

## 🚀 How to Test

### Quick Start (No Database Required)
```bash
# Just verify compilation
cd calendar-service
go build ./internal/availability
echo "✅ Compiled OK"
```

### Full Integration Test (Database Required)
```bash
# 1. Create database schema
PGPASSWORD=postgres psql -h 100.84.126.19 -U postgres -d alpha < docs/schema.sql

# 2. Populate test data
./scripts/phase2-test-setup.sh

# 3. Start service
/path/to/bin/calendar-service -port 8081

# 4. Run tests
./scripts/phase2-integration-test.sh

# 5. Check metrics
curl http://localhost:8081/metrics | grep calendar_profile_resolution
```

---

## 📊 Code Structure

| File | Changes | Purpose |
|------|---------|---------|
| [checker.go](internal/availability/checker.go) | +300 lines | Main resolution logic |
| [types.go](internal/availability/types.go) | No changes | Type definitions (from Phase 1) |
| [test-resolution.sh](scripts/test-resolution.sh) | No changes | Basic tests |
| [phase2-test-setup.sh](scripts/phase2-test-setup.sh) | Created | Database setup |
| [phase2-integration-test.sh](scripts/phase2-integration-test.sh) | Created | Integration tests |

---

## 🔍 Key Functions Reference

### computeResolvedProfile(ctx, tenantID, region, profileName)
**Purpose**: Main orchestrator - fetches holidays/blackouts and builds cache
**Returns**: ResolvedCalendar with holidays ([]time.Time) and blackouts ([]TimeRange)
**Caching**: Auto-caches in L1 (5min) and L2 Redis (1hr)

### fetchScheduleProfile(ctx, tenantID, profileName)
**Purpose**: Query Hasura for schedule profile + linked calendars
**Returns**: ScheduleProfile with CalendarIDs, Priorities, ConflictResolution strategy
**Error**: Returns nil on not found (not fatal)

### fetchHolidaysForCalendars(ctx, tenantID, calendarIDs, start, end)
**Purpose**: Query calendars.holidays JSONB field and parse
**Returns**: []Holiday merged and deduplicated by severity
**Fallback**: Returns empty slice on error (continues without holidays)

### fetchAndExpandBlackouts(ctx, tenantID, calendarIDs, start, end)
**Purpose**: Query blackouts and expand recurring ones with RRULE
**Returns**: []Blackout with all occurrences expanded
**Key**: Calls expandRecurringBlackout() for FREQ patterns

### expandRecurringBlackout(blackout, start, end)
**Purpose**: Convert RRULE string to individual blackout instances
**Returns**: []Blackout with one entry per occurrence
**Library**: Uses github.com/teambition/rrule-go for RFC 5545

---

## 🎯 Testing Scenarios

### Test 1: Availability Check (Normal Day)
```bash
curl -X POST http://localhost:8081/api/v1/availability \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Region: US" \
  -d '{
    "profile_name": "default",
    "start_time": "2026-02-20T09:00:00Z",
    "end_time": "2026-02-20T10:00:00Z"
  }'
```
**Expected**: `{"available": true}`

### Test 2: Holiday Block
```bash
curl -X POST http://localhost:8081/api/v1/availability \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Region: US" \
  -d '{
    "profile_name": "default",
    "start_time": "2026-07-04T09:00:00Z",
    "end_time": "2026-07-04T10:00:00Z"
  }'
```
**Expected**: `{"available": false, "reasons": ["Holiday: 2026-07-04"]}`

### Test 3: Recurring Blackout Expansion
- Blackout: `FREQ=WEEKLY;BYDAY=MO` (Every Monday)
- Range: Feb 1 - Mar 31, 2026
- Expected: 8 Monday occurrences expanded
- Stored in cache as individual TimeRange entries

---

## 💾 Database Tables Required

```sql
-- Essential columns:
calendars.holidays          -- JSONB array with {date, name, type, severity}
blackouts.recurrence_rule   -- VARCHAR with RRULE string
blackouts.start_time        -- TIMESTAMPTZ
blackouts.end_time          -- TIMESTAMPTZ

-- Must exist:
schedule_profiles           -- Profile master records
profile_calendars           -- Profile→calendar links
```

---

## 📈 Expected Performance

| Metric | Target | Notes |
|--------|--------|-------|
| First call | <100ms | Hasura query +orrule expansion |
| L1 cache hit  | <5ms | Local memory |
| L2 cache hit | <20ms | Redis |
| Cache rate | >90% | After warmup |
| Recurring expansion (52 items) | <50ms | rrule-go is efficient |

---

## 🔧 Debugging Tips

### If Hasura query fails
```go
// Check Hasura endpoint
curl -H "X-Hasura-Admin-Secret: <admin-secret>" \
  http://hasura:8080/healthz

// Verify tenant has data
SELECT * FROM calendars WHERE tenant_id = '<id>';
```

### If RRULE parsing fails
```go
// Check recurrence_rule format
SELECT recurrence_rule FROM blackouts LIMIT 1;

// Should be RFC 5545 format:
// FREQ=WEEKLY;BYDAY=MO
// FREQ=MONTHLY;BYMONTHDAY=15
// FREQ=YEARLY;UNTIL=20261231
```

### If cache not populating
```bash
# Check Redis connection
redis-cli -h localhost -p 6379 ping
# Should respond: PONG

# Check cache keys
redis-cli -h localhost -p 6379 KEYS "*resolved*"
```

---

## 🚀 Next Steps (Phase 3)

- [ ] Full database schema deployment
- [ ] Production test data population
- [ ] Load testing with real recurring patterns
- [ ] Performance benchmarking
- [ ] Multi-tenant isolation verification
- [ ] CDC invalidation testing
- [ ] Production deployment

---

## 📞 Quick Issue Resolution

| Error | Solution |
|-------|----------|
| "relation does not exist" | Run database schema: `psql < docs/schema.sql` |
| "failed to parse recurrence rule" | Check RRULE format is RFC 5545 compliant |
| "connection refused" | Verify Hasura and calendar-service ports |
| Cache not hitting | Check Redis connection and TTL values |
| Slow first request | This is expected (~100ms for Hasura + rrule expansion) |

---

## 📋 Files Modified/Created

```
internal/availability/
├─ checker.go            ← MODIFIED (+300 lines, Phase 2)
├─ types.go              ← NO CHANGES (Phase 1)
└─ *.go                  ← NO CHANGES

scripts/
├─ phase2-test-setup.sh               ← CREATED (DB setup)
├─ phase2-integration-test.sh         ← CREATED (API tests)
├─ test-resolution.sh                 ← CREATED (Phase 1)
└─ setup-test-data.sh                 ← CREATED (Phase 1)

docs/
└─ PHASE_2_COMPLETION_SUMMARY.md      ← CREATED (this summary)
```

---

## ✅ Verification Checklist

- [x] Code compiles without errors
- [x] No unused variables or imports
- [x] All functions have error handling
- [x] Logging integrated throughout
- [x] Follows existing code patterns
- [x] Type-safe throughout
- [x] Cache integration points wired
- [x] Test scripts created
- [x] Documentation complete
- [x] Ready for production testing

---

**Status**: 🟢 **READY FOR PHASE 3 TESTING**  
**Build**: ✅ **PASSING**  
**Date**: February 18, 2026  
**Next**: Database integration testing
