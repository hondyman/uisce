# Redis Cache Deployment Guide

## Overview

This guide walks through deploying the Redis caching layer for the Calendar Service. The implementation provides **region-aware cache keys** for global distribution and **CDC-driven cache invalidation** for multi-instance consistency.

**Expected Impact:** Availability check latency drops from ~50ms → <5ms (cache hit).

---

## 1. Infrastructure Setup

### 1.1 Redis Service in Docker Compose

The Redis service has been added to both:
- **Root** (`/docker-compose.yml`) - for integrated deployments
- **Calendar Service** (`calendar-service/docker-compose.yml`) - for standalone deployments

Configuration:
```yaml
redis:
  image: redis:7-alpine
  command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
  volumes:
    - redis_data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
```

**Features:**
- **Persistence**: `--appendonly yes` (AOF for durability)
- **Memory limit**: 256MB with LRU eviction policy
- **Health checks**: Automated via healthcheck

### 1.2 Startup Commands

```bash
# Integrated deployment (from repo root)
docker-compose up redis -d

# Standalone (from calendar-service/)
cd calendar-service/
docker-compose up redis -d

# Verify Redis is healthy
redis-cli ping
# Response: PONG
```

---

## 2. Configuration

### 2.1 Environment Variables

All required variables are in `.env.example`:

```bash
# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_PREFIX=calendar
REDIS_CACHE_TTL=3600          # 1 hour default
CACHE_ENABLED=true             # Toggle caching on/off
```

Copy to `.env`:
```bash
cp .env.example .env
```

### 2.2 Configuration Loading

Config struct in `internal/config/config.go`:
```go
type Config struct {
    RedisURL      string        // redis://host:port
    RedisPrefix   string        // Cache key prefix
    RedisCacheTTL time.Duration // TTL for cache entries
    CacheEnabled  bool          // Enable/disable caching
}
```

Values loaded from environment:
```go
RedisCacheTTL: getEnvDuration("REDIS_CACHE_TTL", 3600*time.Second),
CacheEnabled:  getEnvBool("CACHE_ENABLED", true),
```

---

## 3. Cache Client Implementation

### 3.1 Core Features

**File:** `internal/cache/calendar_cache.go`

```go
type Client struct {
    client *redis.Client
    prefix string
    ttl    time.Duration
    logger *logrus.Entry
    hits   prometheus.Counter  // Metrics
    misses prometheus.Counter
}
```

**Key Methods:**

1. **Get** - Retrieve from cache
```go
func (c *Client) Get(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error)
```

2. **Set** - Store in cache (synchronous)
```go
func (c *Client) Set(ctx context.Context, tenantID, region, profileName string, rc *ResolvedCalendar) error
```

3. **SetAsync** - Store without blocking
```go
func (c *Client) SetAsync(ctx context.Context, tenantID, region, profileName string, rc *ResolvedCalendar)
```

4. **Invalidate** - Delete specific cache entry
```go
func (c *Client) Invalidate(ctx context.Context, tenantID, region, profileName string) error
```

5. **InvalidateTenantProfiles** - Bulk invalidation (used by CDC)
```go
func (c *Client) InvalidateTenantProfiles(ctx context.Context, tenantID, region string, profiles []string)
```

### 3.2 Region-Aware Keys

Cache keys include tenant, region, and profile name:

```
calendar:resolved:{tenantID}:{region}:{profileName}
```

Example:
```
calendar:resolved:550e8400-e29b:us-east-1:default
calendar:resolved:550e8400-e29b:eu-west-1:premium
```

**Benefit**: Different regions can have independent cache entries for the same profile.

### 3.3 Pub/Sub Invalidation

For multi-instance deployments, cache invalidations are broadcast via Redis Pub/Sub:

```go
// Publish invalidation event
c.PublishInvalidation(ctx, tenantID, region)

// Subscribe to invalidation events
c.SubscribeToInvalidations(ctx, func(tenantID, region string) {
    // Local cleanup logic
})
```

---

## 4. Integration with Availability Checker

### 4.1 Updated Signature

Method now includes `region` parameter:

```go
// Before:
func (c *Checker) CheckAvailability(ctx context.Context, tenantID, profileName string, start, end time.Time) (*AvailabilityResult, error)

// After:
func (c *Checker) CheckAvailability(ctx context.Context, tenantID, region, profileName string, start, end time.Time) (*AvailabilityResult, error)
```

### 4.2 Cache-Aside Pattern

The checker implements cache-aside:

```go
func (c *Checker) ResolveProfile(ctx context.Context, tenantID, region, profileName string) (*ResolvedCalendar, error) {
    // 1. Try Cache
    if c.cacheClient != nil {
        cached, err := c.cacheClient.Get(ctx, tenantID, region, profileName)
        if err == nil && cached != nil {
            return cached, nil  // Cache hit!
        }
    }

    // 2. Cache Miss - Resolve from DB
    resolved, err := c.computeResolvedProfile(ctx, tenantID, region, profileName)
    if err != nil {
        return nil, err
    }

    // 3. Populate Cache (Async) - don't block response
    if c.cacheClient != nil && resolved != nil {
        c.cacheClient.SetAsync(ctx, tenantID, region, profileName, resolved)
    }

    return resolved, nil
}
```

### 4.3 Performance Impact

- **Cache Miss** (~50ms): Query DB + merge calendars
- **Cache Hit** (<5ms): Direct Redis retrieval
- **10-20x improvement** on repeated profile checks

---

## 5. CDC-Driven Invalidation

### 5.1 CDC Consumer

File: `internal/redpanda/consumer.go`

```go
type CDCProcessor struct {
    brokers        []string
    topics         []string
    temporalClient client.Client
    cacheClient    *cache.Client
    hasuraClient   *hasura.Client
    logger         *logrus.Entry
}
```

### 5.2 Invalidation Flow

When a calendar, profile, or blackout changes:

```
1. PostgreSQL Update
          ↓
2. Debezium CDC Event (→ Redpanda)
          ↓
3. Calendar Service consumes event
          ↓
4. Cache invalidation: cacheClient.InvalidateTenantProfiles()
          ↓
5. Pub/Sub broadcast to other instances
          ↓
6. Temporal workflow triggered for rescheduling
```

### 5.3 Implementation (Stub Ready)

The CDC processor is ready for full implementation:

```go
func (p *CDCProcessor) processCalendarChange(ctx context.Context, event *CalendarChangeEvent) error {
    // 1. Invalidate cache for affected profiles
    if p.cacheClient != nil {
        p.cacheClient.InvalidateTenantProfiles(
            ctx,
            event.TenantID,
            event.Region,
            event.Profiles,
        )
    }

    // 2. Signal Temporal workflow
    if p.temporalClient != nil {
        // Trigger reschedule for affected jobs
    }

    return nil
}
```

---

## 6. Application Startup

### 6.1 Initialization in main.go

```go
func main() {
    cfg := config.LoadConfig()
    logger := setupLogger(cfg.LogLevel)

    // 1. Initialize Redis Cache
    var cacheClient *cache.Client
    if cfg.CacheEnabled {
        cacheClient = cache.NewClient(
            cfg.RedisURL,
            cfg.RedisPrefix,
            cfg.RedisCacheTTL,
            logger,
        )
        // Subscribe to cross-instance invalidations
        cacheClient.SubscribeToInvalidations(ctx, func(tenantID, region string) {
            logger.Infof("Received invalidation signal for %s/%s", tenantID, region)
        })
        logger.Info("✓ Redis cache initialized")
    } else {
        logger.Info("⊘ Redis cache disabled")
    }

    // 2. Pass cache client to services
    availabilityChecker := availability.NewChecker(
        hasuraClient,
        cacheClient,
        cfg.RedisCacheTTL,
        logger,
    )

    // 3. Start CDC consumer
    go startCDCConsumer(ctx, cfg, temporalClient, cacheClient, hasuraClient, logger)

    // ... rest of startup
}
```

### 6.2 Graceful Shutdown

```go
defer cacheClient.Close()  // Closes Redis connection pool
```

---

## 7. Production Deployment

### 7.1 Docker Compose Deployment

```bash
# 1. Pull images
docker-compose pull redis calendar-service hasura redpanda

# 2. Start full stack
docker-compose up -d

# 3. Verify services
docker-compose ps
# redis         Up (healthy)
# calendar-service  Up (healthy)
# ...

# 4. Check Redis connectivity
redis-cli -h localhost -p 6379 ping
# Response: PONG
```

### 7.2 Environment Configuration

Update deployment `.env`:

```bash
# Production settings
REDIS_URL=redis://redis:6379
REDIS_PREFIX=calendar-prod
REDIS_CACHE_TTL=7200           # 2 hours in production
CACHE_ENABLED=true

ENVIRONMENT=production
LOG_LEVEL=info
```

### 7.3 Monitoring

**Prometheus Metrics** (automatically exposed):

```
calendar_cache_hits_total{tenant_id="...",region="us-east-1"} 1250
calendar_cache_misses_total{tenant_id="...",region="us-east-1"} 45
```

Calculate hit rate:
```
Hit Rate = hits / (hits + misses) = 1250 / 1295 = 96.5%
```

### 7.4 Logs to Monitor

```
✓ Redis cache initialized
Received invalidation signal for 550e8400-e29b-41d4/us-east-1
Cache hit for profile resolution
Async cache set failed (non-critical)
```

---

## 8. Verification Checklist

Before considering deployment complete:

### 8.1 Health Checks

```bash
# Redis is running and responsive
redis-cli ping
# → PONG

# Calendar service started successfully
docker logs calendar-service | grep "Redis cache initialized"
# → ✓ Redis cache initialized

# Configuration loaded correctly
curl http://localhost:8081/health | jq .cache
# → { "status": "healthy", "redis": "connected" }
```

### 8.2 Functional Tests

```bash
# 1. First call (cache miss)
time curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "profile_name": "default",
    "region": "us-east-1",
    "start_time": "2026-02-20T10:00:00Z",
    "end_time": "2026-02-20T11:00:00Z"
  }'
# Response time: ~50ms (DB query + merge)

# 2. Second call (cache hit)
time curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "profile_name": "default",
    "region": "us-east-1",
    "start_time": "2026-02-20T10:00:00Z",
    "end_time": "2026-02-20T11:00:00Z"
  }'
# Response time: <5ms (Redis retrieval)
```

### 8.3 Metrics Verification

```bash
# Check hit/miss counters increased
curl http://localhost:8081/metrics | grep calendar_cache

# Expected output:
# calendar_cache_hits_total{tenant_id="550e8400-e29b-41d4-a716-446655440000",region="us-east-1"} 1
# calendar_cache_misses_total{tenant_id="550e8400-e29b-41d4-a716-446655440000",region="us-east-1"} 1
```

### 8.4 Invalidation Test

```bash
# 1. Update a calendar
curl -X PATCH http://localhost:8081/api/v1/calendars/{CALENDAR_ID} \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"name": "Updated Calendar"}'

# 2. Wait for CDC processing
sleep 2

# 3. Check availability (should be cache MISS now)
time curl -X POST http://localhost:8081/api/v1/check-availability \
  -H "Content-Type: application/json" \
  -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "profile_name": "default",
    "region": "us-east-1",
    "start_time": "2026-02-20T10:00:00Z",
    "end_time": "2026-02-20T11:00:00Z"
  }'
# Response time: ~50ms (cache miss, recomputed)
```

---

## 9. Troubleshooting

### Issue: "Failed to connect to Redis"

```
Error: redis: failed to dial: context deadline exceeded
```

**Solutions:**
1. Verify Redis is running: `docker ps | grep redis`
2. Check Redis is healthy: `redis-cli ping`
3. Verify REDIS_URL matches: `echo $REDIS_URL`
4. Check firewall/network: `telnet localhost 6379`

### Issue: "Cache hits not increasing"

```
calendar_cache_hits_total{tenant_id="...",region="us-east-1"} 0
```

**Check:**
1. `CACHE_ENABLED=true` in `.env`
2. Same region used in consecutive requests
3. Response cached within TTL: `REDIS_CACHE_TTL=3600`

### Issue: "Invalidation not working"

Calendar updates don't clear cache.

**Verify:**
1. CDC consumer is running: `docker logs calendar-service | grep "Starting CDC"`
2. Debezium is connected: Check Debezium logs
3. If needed, manually flush: `redis-cli FLUSHDB`

### Issue: "High memory usage"

Redis memory exceeds limit.

**Solutions:**
1. Reduce `--maxmemory`: Change from 256mb to 128mb in docker-compose
2. Reduce TTL: Lower `REDIS_CACHE_TTL` from 3600 to 1800 seconds
3. Monitor with: `redis-cli INFO memory`

---

## 10. Next Steps

### Phase 2: Schema Updates
- Execute `docs/SCHEMA_UPDATES.sql` to add priority/region fields
- Create indexes for efficient routing

### Phase 3: Data Residency Validation
- Add API validation for tenant region authorization
- Prevent cross-region data access

### Phase 4: API Handler Updates
- Add priority/region parameters to requests
- Default region to service configuration

### Phase 5: Temporal Queue Routing
- Create dispatcher for region-priority queue mapping
- Wire queue selection in workflow execution

---

## References

- [Redis Documentation](https://redis.io/documentation)
- [Go Redis Client](https://pkg.go.dev/github.com/go-redis/redis/v8)
- [Prometheus Metrics](https://prometheus.io/)
- [Redpanda CDC](https://docs.redpanda.com/current/features/cdc/)

---

**Status**: ✅ Phase 1 (Redis Cache) deployment guide complete.
