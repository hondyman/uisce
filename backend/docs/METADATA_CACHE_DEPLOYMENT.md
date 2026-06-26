# Workday-Standard Business Objects - Production Deployment Guide

## Overview

This guide covers deploying the Workday-standard business object system with in-memory metadata caching to production.

## Prerequisites

- PostgreSQL database with `business_objects` table
- Go 1.21+ runtime
- Sufficient memory for metadata caching (estimate: 1KB per BO + 100 bytes per field)

## Configuration

### Environment Variables

```bash
# Enable metadata caching
METADATA_CACHE_ENABLED=true

# Warmup configuration
METADATA_WARMUP_ENABLED=true
METADATA_WARMUP_TIMEOUT=30s

# Optional: Periodic cache refresh (0 = disabled)
METADATA_CACHE_REFRESH_INTERVAL=1h
```

### Server Initialization

Add to your `main.go` or server startup:

```go
import (
    "github.com/hondyman/semlayer/backend/pkg/meta"
    "github.com/hondyman/semlayer/backend/internal/server"
)

func main() {
    // ... existing setup ...

    // Initialize metadata cache
    cache, err := server.WarmupMetadata(ctx, db, server.DefaultMetadataWarmupConfig())
    if err != nil {
        log.Fatalf("Failed to warmup metadata: %v", err)
    }

    // Create service with cache
    metaService := meta.NewServiceWithCache(db.DB, cache)

    // Optional: Schedule periodic warmup
    go server.SchedulePeriodicWarmup(ctx, cache, db, 1*time.Hour)

    // ... continue with server setup ...
}
```

## API Endpoints

### Cache Management

#### Get Cache Statistics
```bash
GET /api/admin/metadata/cache/stats

Response:
{
  "hits": 10000,
  "misses": 50,
  "hit_rate": 0.995,
  "item_count": 150,
  "memory_bytes": 153600,
  "load_time_ms": 245
}
```

#### Warm Cache
```bash
POST /api/admin/metadata/cache/warm
Content-Type: application/json

{
  "tenant_id": "tenant-123"
}
```

#### Invalidate Cache
```bash
POST /api/admin/metadata/cache/invalidate
Content-Type: application/json

{
  "tenant_id": "tenant-123"
}
```

### Business Object Access

#### Get Business Object (Cache-Backed)
```bash
GET /api/metadata/business-objects?name=Worker
X-Tenant-ID: tenant-123

Response Headers:
X-Cache-Hit: true

Response: <business object definition>
```

## Monitoring

### Key Metrics to Monitor

1. **Cache Hit Rate**: Should be >95% after warmup
2. **Memory Usage**: Monitor `memory_bytes` in cache stats
3. **Load Time**: Initial warmup should complete in <30s
4. **Evictions**: Should be minimal in steady state

### Prometheus Metrics (Optional)

```go
// Add to your metrics collector
metadataCacheHits.Inc()
metadataCacheMisses.Inc()
metadataCacheMemoryBytes.Set(float64(metrics.MemoryBytes))
```

### Health Check

```go
func healthCheck(cache *meta.MetadataCache) error {
    metrics := cache.GetMetrics()
    
    if metrics.ItemCount == 0 {
        return fmt.Errorf("cache is empty")
    }
    
    if metrics.HitRate < 0.90 {
        return fmt.Errorf("cache hit rate too low: %.2f%%", metrics.HitRate*100)
    }
    
    return nil
}
```

## Performance Tuning

### Memory Sizing

Estimate memory requirements:
```
Total Memory = (Number of BOs × 1KB) + (Number of Fields × 100 bytes)

Example:
- 100 Business Objects
- 10 fields per BO average
= (100 × 1KB) + (1000 × 100 bytes)
= 100KB + 100KB = 200KB
```

### Cache Refresh Strategy

**Option 1: On-Demand** (Recommended)
- Invalidate cache when metadata changes
- Warm cache immediately after invalidation

**Option 2: Periodic**
- Schedule cache refresh every 1-6 hours
- Good for read-heavy workloads

**Option 3: Hybrid**
- Invalidate on changes + periodic refresh as backup

## Troubleshooting

### Cache Not Warming on Startup

**Symptom**: `ItemCount = 0` in cache stats

**Solutions**:
1. Check database connectivity
2. Verify `business_objects` table exists
3. Check timeout configuration
4. Review startup logs for errors

### High Cache Miss Rate

**Symptom**: Hit rate <90%

**Solutions**:
1. Verify cache is warmed for all active tenants
2. Check if cache is being invalidated too frequently
3. Increase warmup timeout if loading is slow

### Memory Issues

**Symptom**: High memory usage or OOM errors

**Solutions**:
1. Reduce number of tenants preloaded
2. Implement selective caching (only active tenants)
3. Increase server memory allocation

## Rollback Plan

If issues occur in production:

1. **Disable Caching**:
   ```bash
   METADATA_CACHE_ENABLED=false
   ```
   Service will fall back to database queries

2. **Restart Service**: No data loss, cache is in-memory only

3. **Monitor**: Check database query performance

## Security Considerations

1. **Cache Isolation**: Each tenant's metadata is isolated
2. **Access Control**: Use existing ABAC/RBAC for API endpoints
3. **Audit Trail**: Cache operations are logged

## Best Practices

✅ **DO**:
- Monitor cache hit rate daily
- Set up alerts for hit rate <90%
- Warm cache on startup
- Invalidate cache when metadata changes
- Use cache for read-heavy operations

❌ **DON'T**:
- Cache instance data (only metadata)
- Rely on cache for critical writes
- Skip warmup in production
- Ignore cache metrics

## Support

For issues or questions:
1. Check cache stats: `GET /api/admin/metadata/cache/stats`
2. Review server logs for warmup errors
3. Verify database schema is up to date
4. Test cache invalidation manually
