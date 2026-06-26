# Phase 6: Integration Quick Start Guide

**Previous Phase:** ✅ Phase 5 - Security Hardening Complete  
**Current Phase:** Phase 6 - Integration & Deployment  
**Estimated Duration:** 3-4 hours

---

## 🎯 Phase 6 Objectives

1. ✅ Wire security components into main application
2. ✅ Integrate audit service calls into handlers
3. ✅ Run comprehensive test suite
4. ✅ Deploy to staging environment
5. ✅ Canary deploy to production

---

## 📋 Integration Checklist

### Step 1: Main Server Setup (30 minutes)

**File:** `cmd/server/main.go`

**Add rate limiter initialization:**
```go
import (
    "calendar-service/internal/middleware"
)

func main() {
    // ... existing setup ...
    
    // Initialize rate limiter
    rateLimiter := middleware.NewTenantRateLimiter(
        viper.GetFloat64("RATE_LIMIT_RPS"),      // From env: RATE_LIMIT_RPS
        viper.GetInt("RATE_LIMIT_BURST"),        // From env: RATE_LIMIT_BURST
        logger)
    
    // Wire into middleware stack
    r := mux.NewRouter()
    r.Use(middleware.RequestID)                  // Existing
    r.Use(middleware.JWTMiddleware(...))        // Existing
    r.Use(middleware.TenantGuardMiddleware(...)) // Existing
    r.Use(rateLimiter.RateLimit)                // NEW
    
    // Register handlers
    registerHandlers(r, ...)
}
```

**Required Environment Variables:**
```bash
RATE_LIMIT_RPS=10          # Requests per second per tenant
RATE_LIMIT_BURST=20        # Token bucket burst size
```

### Step 2: Handler Integration (1 hour)

**Pattern (repeat for all 4 handlers):**

**File:** `internal/handlers/calendar.go` (example)

```go
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Extract claims from context
    userID := middleware.ExtractUserIDFromContext(ctx)
    tenantID := middleware.ExtractTenantIDFromContext(ctx)
    
    if userID == "" || tenantID == "" {
        http.Error(w, "missing required context", http.StatusUnauthorized)
        return
    }
    
    // Parse request
    var req CreateCalendarRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Call service
    cal, err := h.service.CreateCalendar(ctx, tenantID, req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // NEW: Record audit entry
    h.auditSvc.RecordCreate(ctx, tenantID, "calendar", cal.ID,
        map[string]interface{}{
            "name":        cal.Name,
            "description": cal.Description,
            "timezone":    cal.Timezone,
        }, userID)
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(cal)
}
```

**Similar patterns for:**
- `Update()` - Use RecordUpdate with old/new values
- `Delete()` - Use RecordDelete with old values
- `Get()` - No audit logging (read-only)

**Checklist:**
- [ ] Calendar Create - Add RecordCreate
- [ ] Calendar Update - Add RecordUpdate  
- [ ] Calendar Delete - Add RecordDelete
- [ ] Blackout Create - Add RecordCreate
- [ ] Blackout Update - Add RecordUpdate
- [ ] Blackout Delete - Add RecordDelete
- [ ] Profile Create - Add RecordCreate
- [ ] Profile Update - Add RecordUpdate

### Step 3: Test & Verify (30 minutes)

**Run integration tests:**
```bash
# Unit tests
go test ./... -v

# Security tests
go test -tags=security -v ./tests/security/...

# Expected results: All tests passing
```

**Run E2E tests:**
```bash
# Build binary
go build -o bin/calendar-service ./cmd/server

# Start service
JWT_SECRET=test-secret DATABASE_URL=postgres://... \
  RATE_LIMIT_RPS=10 RATE_LIMIT_BURST=20 \
  ./bin/calendar-service &

# Run E2E tests
bash e2e-tests.sh

# Expected: 14/14 tests should still pass
```

### Step 4: Monitoring Setup (30 minutes)

**Prometheus metrics to track:**

```yaml
# Alert: High JWT validation failures
- alert: JWTValidationSpike
  expr: rate(jwt_validation_errors_total[5m]) > 50
  for: 5m

# Alert: High rate limiting triggers
- alert: RateLimitAbuse
  expr: rate(rate_limit_exceeded_total[5m]) > 100
  for: 5m

# Alert: Audit anomalies
- alert: AuditLogSpike
  expr: rate(audit_entries_total[5m]) > 500
  for: 5m
```

**Check Prometheus:**
```
http://localhost:9090
```

### Step 5: Staging Deployment (1 hour)

**Deploy to staging:**
```bash
cd /Users/eganpj/GitHub/semlayer/calendar-service

# Build
go build -o bin/calendar-service ./cmd/server

# Get staging config
source staging.env

# Deploy
docker build -t calendar-service:latest .
docker push registry/calendar-service:$VERSION
kubectl set image deployment/calendar-service \
  calendar-service=registry/calendar-service:$VERSION \
  -n staging
```

**Verify deployment:**
```bash
# Check health
curl -H "Authorization: Bearer $TEST_JWT" \
  https://staging-api.example.com/calendars

# Check rate limiting
for i in {1..25}; do
  curl -H "Authorization: Bearer $TEST_JWT" \
    https://staging-api.example.com/calendars
done
# Expected: First 20 succeed (burst), next 5 fail with 429

# Check audit logs
SELECT * FROM audit_logs 
WHERE action IN ('CREATE', 'UPDATE', 'DELETE') 
ORDER BY created_at DESC LIMIT 10;
```

### Step 6: Canary Deployment (1 hour)

**Deploy to 10% of production:**
```bash
kubectl set image deployment/calendar-service \
  calendar-service=registry/calendar-service:$VERSION \
  -n production --record

kubectl rollout status deployment/calendar-service -n production

# Set traffic split (10% canary, 90% stable)
kubectl patch virtualservice calendar-service \
  -p '{"spec":{"hosts":[{"name":"calendar-service","http":[
    {"match":[{"uri":{"prefix":"/v2"}}],"route":[{"destination":{"host":"calendar-service","subset":"canary"},"weight":10},
    {"destination":{"host":"calendar-service","subset":"stable"},"weight":90}]}
  ]}]}' -n production
```

**Monitor canary:**
```
- Error rate < 0.1%
- Latency p99 < 200ms
- No rate limit abuse
- Audit logs flowing normally
```

**Rollback if needed:**
```bash
kubectl rollout undo deployment/calendar-service -n production
```

---

## 🧪 Testing Checklist

### Before Staging
- [ ] All unit tests pass
- [ ] All security tests pass
- [ ] Build succeeds with no errors
- [ ] Rate limiter wired into main.go
- [ ] Audit service calls added to all handlers
- [ ] Environment variables documented

### Staging Smoke Tests
- [ ] Can create calendar (audit logged)
- [ ] Can update calendar (audit logged with diffs)
- [ ] Can delete calendar (audit logged)
- [ ] Rate limiting works (429 after burst)
- [ ] Cross-tenant access blocked (403)
- [ ] JWT validation enforced
- [ ] Audit logs are queryable

### Canary Monitoring
- [ ] Error rate normal
- [ ] Latency normal
- [ ] Rate limiting events tracked
- [ ] Audit entries flowing
- [ ] No JWT validation spikes
- [ ] No tenant isolation violations

---

## 📊 Validation Metrics

### Post-Deployment Checklist
- [ ] 0 errors in first hour
- [ ] Error rate < 0.1%
- [ ] P99 latency < 200ms
- [ ] Audit entries > 0 (mutations flowing)
- [ ] Rate limit events < 1% of requests
- [ ] No tenant isolation violations
- [ ] Prometheus alerts functioning

---

## 🆘 Troubleshooting

### Rate Limiter Not Working
```bash
# Check middleware is in stack
grep "rateLimiter.RateLimit" cmd/server/main.go

# Verify environment variables
echo $RATE_LIMIT_RPS $RATE_LIMIT_BURST

# Test directly
go test -tabs=security ./tests/security/... -run "TestRateLimitingSecurity"
```

### Audit Not Logging
```bash
# Check service calls exist
grep "auditSvc.Record" internal/handlers/*.go

# Check service initialized
grep "NewAuditService" cmd/server/main.go

# Test directly
go test -tags=security ./tests/security/... -run "TestAuditLoggingCompleteness"
```

### Cross-Tenant Access Not Blocked
```bash
# Check middleware order
# Order should be: JWT → TenantGuard → RateLimit

# Test directly
go test -tags=security ./tests/security/... -run "TestTenantGuardMiddlewareSecurity"
```

---

## 📞 Key Documents

- [Security Checklist](docs/deployment/SECURITY_CHECKLIST.md) - Pre-deployment validation
- [Security Runbook](docs/operations/SECURITY_RUNBOOK.md) - Incident response
- [Phase 5 Complete](docs/PHASE_5_COMPLETE.md) - Technical details
- [Phase Checklist](PHASE_5_CHECKLIST.md) - Acceptance criteria

---

## ⏱️ Estimated Time Breakdown

| Task | Duration | Status |
|------|----------|--------|
| Main server setup | 30 min | ⏳ To Do |
| Handler integration | 1 hour | ⏳ To Do |
| Testing & verification | 30 min | ⏳ To Do |
| Monitoring setup | 30 min | ⏳ To Do |
| Staging deployment | 1 hour | ⏳ To Do |
| Canary deployment | 1 hour | ⏳ To Do |
| **TOTAL** | **4-5 hours** | ⏳ To Do |

---

**Start Phase 6:** When ready, begin with Step 1 (Main Server Setup)  
**Previous Phases:** ✅ All phases 1-5 complete  
**Readiness:** ✅ Production-ready components developed and tested
