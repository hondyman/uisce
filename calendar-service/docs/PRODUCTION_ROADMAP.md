# 🎯 Epic 31 Production Readiness: Implementation Roadmap

**Status**: 90% production-ready. These are the final critical implementations needed before deployment.

---

## 📊 Implementation Priority Matrix

| Priority | Item | Effort | Impact | Do First? |
|----------|------|--------|--------|-----------|
| 🔴 **CRITICAL-1** | Redis caching layer | 2h | **10-20x latency↓** | ✅ **START HERE** |
| 🔴 **CRITICAL-2** | Schema: priority + region fields | 30m | Enable routing | ✅ Then this |
| 🔴 **CRITICAL-3** | Data residency validation (RLS) | 30m | Security gate | Then this |
| 🟠 **HIGH-1** | API: Accept priority/region params | 1h | Link to schema | Then this |
| 🟠 **HIGH-2** | Temporal: Queue routing by priority | 1h | Global dist. | Then this |
| 🟡 **MEDIUM** | Batch availability endpoint | 2h | 5-10% gain | After HIGH |
| 🟢 **NICE** | ICS import/export + webhooks | 4h | 0% core path | Polish phase |

---

## 🚀 Phase 1: Redis Caching (2 hours) - DO THIS FIRST

**Target**: 10-20x latency improvement on core operation (availability checking)

### Files Created ✅
- ✅ `internal/cache/calendar_cache.go` - Redis client with cache-aside pattern
- ✅ `docs/CACHE_INTEGRATION.md` - Complete integration guide
- ✅ `docs/SCHEMA_UPDATES.sql` - Schema additions

### Files to Modify

**1. `internal/availability/checker.go`**
- Inject `redisCache *cache.CalendarCache` into struct
- In `ResolveProfile()`: Check cache → compute → SetAsync()
- In `CheckAvailability()`: Accept `priority`, `region` parameters

**2. `cmd/server/main.go`**
```go
// Initialize cache
redisCache := cache.NewCalendarCache(cfg.RedisURL, cfg.RedisPrefix, cfg.RedisCacheTTL, logger)
go redisCache.SubscribeToInvalidations(ctx)

// Pass to checker
availabilityChecker := availability.NewChecker(hasuraClient, redisCache, cfg.RedisCacheTTL, logger)
```

**3. `internal/redpanda/consumer.go`**
- In `processRecord()` after Temporal signal:
```go
// Invalidate cache for affected profiles
profileNames, _ := p.resolveAffectedProfiles(ctx, signal.TenantID, signal.EntityID)
for _, profileName := range profileNames {
    p.cacheClient.Invalidate(ctx, signal.TenantID, profileName)
    p.cacheClient.PublishInvalidationEvent(ctx, signal.TenantID, profileName)
}
```

**4. `.env.example`** - Add Redis variables
```bash
REDIS_URL=redis://localhost:6379
REDIS_CACHE_TTL=3600
REDIS_PREFIX=calendar
CACHE_ENABLED=true
```

### Testing
```bash
# First call (cache miss ~50ms)
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"profile_name":"default","start_time":"2026-02-18T09:00:00Z","end_time":"2026-02-18T10:00:00Z"}'

# Second call (cache hit <5ms) ⚡
curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"profile_name":"default","start_time":"2026-02-18T09:00:00Z","end_time":"2026-02-18T10:00:00Z"}'

# Verify metrics
curl http://localhost:8081/metrics | grep calendar_cache_hits
```

---

## 🔧 Phase 2: Schema Updates (30 minutes)

**Target**: Enable global distribution + priority queue routing

### Execute Schema
```bash
cd calendar-service
psql -f docs/SCHEMA_UPDATES.sql \
  -h localhost -U postgres -d calendar_db \
  -v DB_HOST=localhost -v DB_PORT=5432 -v DB_USER=postgres -v DB_NAME=calendar_db
```

This adds:
- ✅ `jobs.priority` (1-10)
- ✅ `jobs.region` (us-east-1, eu-west-1, ap-southeast-1, etc.)
- ✅ `jobs.resource_profile` (minimal, standard, high-memory, cpu-intensive)
- ✅ `jobs.sla_deadline` for deadline-aware scheduling
- ✅ Indexes for efficient routing
- ✅ `tenant_region_authorizations` table

### Verify
```sql
-- Check columns added
SELECT * FROM information_schema.columns 
WHERE table_name='jobs' 
AND column_name IN ('priority', 'region', 'resource_profile', 'sla_deadline');

-- Check indexes
SELECT * FROM information_schema.indexes 
WHERE table_name='jobs' AND indexname LIKE 'idx_jobs%';
```

---

## 🔐 Phase 3: Data Residency Validation (30 minutes)

**Target**: Prevent cross-region data access (security gate)

### Add API Validation

**File**: `internal/api/availability_handlers.go`

```go
// Validate tenant is authorized for region
func (h *AvailabilityHandler) validateRegion(tenantID, region string) error {
	// Query tenant_region_authorizations
	query := `
		SELECT COUNT(*) FROM tenant_region_authorizations 
		WHERE tenant_id = $1 AND region = $2
	`
	
	var count int
	if err := h.db.QueryRow(query, tenantID, region).Scan(&count); err != nil || count == 0 {
		return fmt.Errorf("region %s not authorized for tenant", region)
	}
	return nil
}

// In Check handler:
if req.Region != "" {
	if err := h.validateRegion(req.TenantID, req.Region); err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}
```

---

## 📋 Phase 4: API Handler Updates (1 hour)

**Target**: Wire priority + region through all endpoints

### Update Request Structs

**File**: `internal/api/availability_handlers.go`

```go
type CheckAvailabilityRequest struct {
	ProfileName string    `json:"profile_name" required:"true"`
	StartTime   time.Time `json:"start_time" required:"true"`
	EndTime     time.Time `json:"end_time" required:"true"`
	
	// ADD THESE:
	Priority int    `json:"priority,omitempty"` // 1-10, default 5
	Region   string `json:"region,omitempty"`   // default from config
}

// In handler:
priority := req.Priority
if priority == 0 {
	priority = 5 // default
}
region := req.Region
if region == "" {
	region = h.cfg.DefaultRegion
}

// Pass to checker
available, reasons, _ := h.checker.CheckAvailability(
	ctx, tenantID, req.ProfileName,
	req.StartTime, req.EndTime,
	priority, region, // NEW PARAMS
)
```

### Update all API endpoints:
- ✅ `/api/v1/check-availability` (POST)
- ✅ `/api/v1/calendars` (GET, POST, PATCH)
- ✅ `/api/v1/profiles` (GET, POST, PATCH)
- ✅ `/api/v1/jobs` (POST) - new endpoint for job submission

---

## 🎯 Phase 5: Temporal Queue Routing (1 hour)

**Target**: Route jobs to correct priority queue by region

### Create Dispatcher

**File**: `internal/temporal/dispatcher.go` (NEW)

```go
package temporal

import "fmt"

// GetTaskQueueName returns Temporal task queue for routing
// Format: {region}-{priority_tier}-queue
func GetTaskQueueName(region string, priority int) string {
	tier := "standard"
	if priority <= 2 {
		tier = "critical"
	} else if priority >= 8 {
		tier = "bulk"
	}
	return fmt.Sprintf("%s-%s-queue", region, tier)
}
```

### Use in Workflow Execution

**File**: `internal/temporal/workflows/calendar_changed.go`

```go
// In CalendarChangedWorkflow or when submitting jobs:
opts := client.StartWorkflowOptions{
	ID:        fmt.Sprintf("job-%s", jobID),
	TaskQueue: dispatcher.GetTaskQueueName(job.Region, job.Priority),
	// Existing options...
}

w, err := c.ExecuteWorkflow(ctx, opts, CalendarChangedWorkflow, params)
```

### Start Workers for Each Queue

**File**: `cmd/server/main.go`

```go
// For each region + priority tier combination:
for _, region := range cfg.WorkerRegions {
	for _, tier := range []string{"critical", "standard", "bulk"} {
		queueName := fmt.Sprintf("%s-%s-queue", region, tier)
		
		// Determine worker count
		workerCount := cfg.StandardQueueWorkers
		if tier == "critical" {
			workerCount = cfg.CriticalQueueWorkers
		} else if tier == "bulk" {
			workerCount = cfg.BulkQueueWorkers
		}
		
		// Register worker
		worker := worker.New(temporalClient, queueName, worker.Options{
			MaxConcurrentActivityExecutionSize: workerCount,
		})
		
		worker.RegisterWorkflow(workflows.CalendarChangedWorkflow)
		worker.RegisterActivity(&activities.OptimizationActivities{})
		
		go func() {
			if err := worker.Start(); err != nil {
				logger.WithError(err).WithField("queue", queueName).Error("Worker failed to start")
			}
		}()
		
		logger.WithField("queue", queueName).WithField("workers", workerCount).Info("Worker started")
	}
}
```

---

## ✅ Pre-Production Checklist

```bash
# Phase 1: Caching
[ ] Redis cache implementation working
[ ] Cache hit latency <5ms verified
[ ] CDC invalidation tested
[ ] Multi-instance Pub/Sub working

# Phase 2: Schema
[ ] Schema updates applied
[ ] Indexes created and verified
[ ] tenant_region_authorizations seeded

# Phase 3: Validation
[ ] Region validation enforced
[ ] Unauthorized region requests rejected (403)
[ ] Data residency compliance verified

# Phase 4: API
[ ] Priority parameter accepted
[ ] Region parameter accepted
[ ] Defaults applied when missing
[ ] Request validation working

# Phase 5: Routing
[ ] Dispatcher routing jobs correctly
[ ] Workers processing from correct queues
[ ] Priority queue ordering verified
[ ] Regional isolation verified

# Testing
[ ] Load test: 1000+ checks/sec with cache
[ ] Redis failover: graceful degradation
[ ] Cache invalidation: timely and accurate
[ ] Multi-tenant isolation: verified
[ ] Authorization checks: all pass
```

---

## 📊 Performance Targets (Post-Implementation)

| Metric | Current | Target | Status |
|--------|---------|--------|--------|
| Availability check latency (p95) | 50-100ms | <5ms | 🎯 10-20x |
| DB queries per check | 3-5 | 0 (on cache hit) | 🎯 90% reduction |
| Max throughput (checks/sec) | ~200 | 2000+ | 🎯 10x scale |
| Time to resolve profile | 50ms | <1ms | 🎯 50x faster |
| Tenant scale (same hardware) | 100 | 1000+ | 🎯 10x capacity |

---

## 🚀 Expected Timeline

| Phase | Tasks | Effort | Timeline |
|-------|-------|--------|----------|
| **1** | Redis cache | 2h | Day 1 |
| **2** | Schema updates | 30m | Day 1 |
| **3** | Data residency | 30m | Day 1-2 |
| **4** | API handlers | 1h | Day 2 |
| **5** | Temporal routing | 1h | Day 2 |
| **Testing** | Integration + load | 3h | Day 2-3 |
| **Deployment** | Staging → Production | 2h | Day 3 |
| | **TOTAL** | | **~12 hours** |

---

## 🎯 Next Immediate Actions

1. ✅ Review `docs/CACHE_INTEGRATION.md` (5 min read)
2. ✅ Implement Phase 1 : Update `checker.go` + `main.go` + `consumer.go`
3. ✅ Test Phase 1 in dev environment (verify cache hits)
4. ✅ Execute Phase 2: Run `SCHEMA_UPDATES.sql`
5. ✅ Implement Phases 3-5

**Estimated completion**: 12 hours for all 5 phases + testing.

---

## 💡 Key Success Metrics

**Verify after Phase 1:**
```bash
# Cache hit should be <5ms
redis-benchmark -h localhost -n 1000 | grep avg

# Latency improvement should be visible 
curl -w "@curl-format.txt" -o /dev/null \
  http://localhost:8081/api/v1/check-availability

# Should see ~99% cache hit rate after warm-up
curl http://localhost:8081/metrics | grep calendar_cache_hits
```

---

**You're at 90%. These final 12 hours of implementation get you to 100% production-ready.** 🚀

Start with Phase 1 (Redis cache) - that's where the biggest ROI is. Once that's verified working and fast, move through phases 2-5 methodically.

Questions on any phase? I can provide exact code snippets for any file that needs modification.
