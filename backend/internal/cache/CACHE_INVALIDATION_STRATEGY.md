# Semantic Query Cache Invalidation Strategy

## Overview

Cache invalidation is critical to maintaining data consistency while maximizing performance gains. This document outlines the complete invalidation strategy with implementation patterns and operational procedures.

## Design Principles

1. **Automatic TTL-Based Expiration** - Primary invalidation mechanism
2. **Event-Driven Invalidation** - Secondary, intentional invalidation
3. **Graceful Degradation** - Cache misses fall back to full pipeline
4. **Zero Data Loss** - Redis failure doesn't corrupt data

## Layer-Specific Invalidation Rules

### Layer 1: NL → SemanticQuery (24h TTL)

**Never invalidates unless:**
- Semantic bundle is updated
- LLM model is changed
- Tenant is offboarded

**Key**: `nlquery:HASH(prompt+datasource+mode+tenantID)`

**Invalidation Events**:
```go
// Bundle structure changed
OnSemanticBundleUpdated(ctx, tenantID, boID)

// New LLM model (may generate different queries)
OnLLMModelUpdated(ctx, tenantID)

// Tenant removed
OnTenantOffboarded(ctx, tenantID)
```

### Layer 2: SemanticQuery → SQL (7d TTL)

**Never invalidates unless:**
- Semantic bundle updated (fields changed)
- Database schema changed
- LLM model changed
- SQL generation rules updated

**Key**: `sqlquery:HASH(semantic_query+dbtype+tenantID)`

**Invalidation Events**:
```go
// Bundle structure changed (field names, types)
OnSemanticBundleUpdated(ctx, tenantID, boID)

// Database schema changed
OnDatabaseSchemaChanged(ctx, tenantID, dbName)

// LLM model change affects SQL generation
OnLLMModelUpdated(ctx, tenantID)

// Tenant removed
OnTenantOffboarded(ctx, tenantID)
```

### Layer 3: SQL → Results (5m TTL)

**Always invalidates after 5 minutes (automatic)**

**Event-driven invalidation** (before TTL):
- Data source modified (INSERT/UPDATE/DELETE)
- Table schema changed
- View redefined
- Data refresh triggered

**Key**: `results:HASH(sql+tenantID+dbName)`

**Invalidation Events**:
```go
// Table updated (detected via data change notification)
OnDataSourceModified(ctx, tenantID, dbName, table)

// Schema change
OnDatabaseSchemaChanged(ctx, tenantID, dbName)

// Explicit refresh
OnDataRefreshTriggered(ctx, tenantID)

// Tenant removed
OnTenantOffboarded(ctx, tenantID)
```

## Implementation Patterns

### Pattern 1: Semantic Bundle Update

**Trigger**: Bundle fields, types, or mappings change

```go
// In semantic_model_handlers.go or similar
func UpdateSemanticBundle(ctx context.Context, srv *Server, boID string, updates *SemanticBundleUpdate) error {
    // 1. Update the bundle in database
    if err := updateBundleInDB(ctx, srv.DB, boID, updates); err != nil {
        return err
    }

    // 2. Invalidate ALL cache layers for this tenant
    tenantID := getTenantIDFromContext(ctx)
    
    if srv.QueryCache != nil {
        if err := srv.QueryCache.InvalidateTenantCache(ctx, tenantID); err != nil {
            log.Printf("Warning: failed to invalidate cache: %v", err)
            // Don't fail the update - cache invalidation is not critical
        }
    }

    // 3. Trigger invalidation manager hooks
    if srv.InvalidationManager != nil {
        if err := srv.InvalidationManager.OnSemanticBundleUpdated(ctx, tenantID, boID); err != nil {
            log.Printf("Warning: invalidation manager hook failed: %v", err)
        }
    }

    log.Printf("Updated semantic bundle and invalidated cache: bo=%s tenant=%s", boID, tenantID)
    return nil
}
```

### Pattern 2: Database Schema Change

**Trigger**: Column added/removed, type changed, table renamed

Implementation depends on database monitoring strategy:

**Option A: Database Triggers**

```sql
-- PostgreSQL function to track schema changes
CREATE OR REPLACE FUNCTION notify_schema_change()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'schema_change',
        json_build_object(
            'table', TG_TABLE_NAME,
            'operation', TG_OP,
            'timestamp', NOW()
        )::text
    );
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Listen for schema changes in Go
go func() {
    listener := pq.NewListener(dbConnStr, 5*time.Second, time.Minute, eventCallback)
    if err := listener.ListenContext(ctx, "schema_change"); err != nil {
        log.Printf("Listen error: %v", err)
    }
    listener.WaitForNotification(ctx)
}()
```

**Option B: Periodic Schema Inspection**

```go
// Poll database schema every hour
func SchemaChangeDetector(ctx context.Context, srv *Server, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    lastSchema := getCurrentSchema(ctx, srv.DB)

    for range ticker.C {
        currentSchema := getCurrentSchema(ctx, srv.DB)
        
        if !schemasEqual(lastSchema, currentSchema) {
            log.Printf("Schema change detected!")

            // Invalidate all query cache
            if err := srv.QueryCache.InvalidateTenantCache(ctx, "all-tenants"); err != nil {
                log.Printf("Cache invalidation error: %v", err)
            }

            lastSchema = currentSchema
        }
    }
}
```

### Pattern 3: Tenant Offboarding

**Trigger**: Tenant subscription cancelled or account deleted

```go
func OffboardTenant(ctx context.Context, srv *Server, tenantID string) error {
    // 1. Begin transaction for consistency
    tx, err := srv.DB.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 2. Delete tenant data
    if err := deleteTenantData(ctx, tx, tenantID); err != nil {
        return err
    }

    // 3. Clear from cache immediately
    if srv.QueryCache != nil {
        if err := srv.QueryCache.InvalidateTenantCache(ctx, tenantID); err != nil {
            log.Printf("Warning: failed to clear tenant cache: %v", err)
        }
    }

    // 4. Trigger invalidation hooks
    if srv.InvalidationManager != nil {
        if err := srv.InvalidationManager.OnTenantOffboarded(ctx, tenantID); err != nil {
            log.Printf("Warning: invalidation manager hook failed: %v", err)
        }
    }

    // 5. Commit transaction
    if err := tx.Commit(); err != nil {
        return err
    }

    log.Printf("Tenant offboarded and cache cleared: tenant=%s", tenantID)
    return nil
}
```

### Pattern 4: Explicit Cache Refresh

**Trigger**: Admin-initiated full cache refresh

```go
// Handler for admin endpoint: POST /api/admin/cache/refresh
func RefreshCacheHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := r.Header.Get("X-Tenant-ID")

    // 1. Parse request parameters
    var req struct {
        Scope string `json:"scope"` // "tenant", "layer", "all"
        Layer int    `json:"layer"` // 1, 2, or 3 (optional)
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    var refreshedKeys int64

    // 2. Execute invalidation based on scope
    switch req.Scope {
    case "tenant":
        if err := srv.QueryCache.InvalidateTenantCache(ctx, tenantID); err != nil {
            http.Error(w, "Cache refresh failed", http.StatusInternalServerError)
            return
        }
        refreshedKeys = 100 // Approximate

    case "all":
        if err := srv.QueryCache.ClearAllCache(ctx); err != nil {
            http.Error(w, "Cache refresh failed", http.StatusInternalServerError)
            return
        }
        refreshedKeys = 50000 // Approximate

    default:
        http.Error(w, "Invalid scope", http.StatusBadRequest)
        return
    }

    log.Printf("Cache refreshed: scope=%s tenant=%s keys=%d", req.Scope, tenantID, refreshedKeys)

    // 3. Return success response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":    "success",
        "scope":     req.Scope,
        "keys_cleared": refreshedKeys,
        "timestamp": time.Now(),
    })
}
```

### Pattern 5: LLM Model Update

**Trigger**: LLM model switched or updated

```go
func UpdateLLMModel(ctx context.Context, srv *Server, newModel string) error {
    // 1. Update configuration
    if err := updateLLMModelConfig(ctx, srv.DB, newModel); err != nil {
        return err
    }

    // 2. Invalidate all tenant caches (Layer 1 & 2)
    // Different LLM may generate different queries and SQL
    allTenants, err := getAllTenants(ctx, srv.DB)
    if err != nil {
        return err
    }

    for _, tenant := range allTenants {
        if srv.QueryCache != nil {
            if err := srv.QueryCache.InvalidateTenantCache(ctx, tenant.ID); err != nil {
                log.Printf("Warning: failed to invalidate cache for tenant %s: %v", tenant.ID, err)
            }
        }
    }

    // 3. Notify invalidation manager
    if srv.InvalidationManager != nil {
        for _, tenant := range allTenants {
            srv.InvalidationManager.OnLLMModelUpdated(ctx, tenant.ID)
        }
    }

    log.Printf("LLM model updated to %s and cache invalidated", newModel)
    return nil
}
```

## Monitoring Invalidation Events

### Audit Logging

```go
type CacheInvalidationAuditLog struct {
    ID          string    `db:"id"`
    TenantID    string    `db:"tenant_id"`
    Reason      string    `db:"reason"` // bundle_updated, schema_changed, etc.
    Scope       string    `db:"scope"`  // tenant, all, specific
    KeysCleared int64     `db:"keys_cleared"`
    Duration    int64     `db:"duration_ms"`
    InitiatedBy string    `db:"initiated_by"` // user_id or "system"
    Timestamp   time.Time `db:"timestamp"`
}

// Log all invalidation events
func LogInvalidationEvent(ctx context.Context, db *sql.DB, event *CacheInvalidationAuditLog) error {
    return db.QueryRowContext(ctx, `
        INSERT INTO cache_invalidation_audit_log 
        (id, tenant_id, reason, scope, keys_cleared, duration_ms, initiated_by, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `,
        uuid.New().String(),
        event.TenantID,
        event.Reason,
        event.Scope,
        event.KeysCleared,
        event.Duration,
        event.InitiatedBy,
        time.Now(),
    ).Scan(&event.ID)
}
```

### Alerting on Excessive Invalidation

```go
// Detect if cache is being invalidated too frequently
type InvalidationFrequencyAlert struct {
    threshold time.Duration // e.g., 1 minute
    checker   *time.Ticker
}

func(ifa *InvalidationFrequencyAlert) Check(ctx context.Context, db *sql.DB, tenantID string) error {
    // Get recent invalidation events
    var recentCount int
    err := db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM cache_invalidation_audit_log
        WHERE tenant_id = $1 AND timestamp > NOW() - INTERVAL '1 hour'
    `, tenantID).Scan(&recentCount)

    if err != nil {
        return err
    }

    // Alert if >60 invalidations per hour (1 per minute average)
    if recentCount > 60 {
        log.Printf("ALERT: High invalidation frequency for tenant %s: %d in last hour", tenantID, recentCount)
        // Could trigger pagerduty, email, etc.
    }

    return nil
}
```

## Consistency Guarantees

### Write-Through Cache

```
Request → Cache Check → 
   Miss? → Generate → Store in Cache → Return
   Hit?  → Return from Cache
```

**No dirty reads**: Results always fresh or from cache

### Eventual Consistency

When cache expires or is invalidated:
- **Immediate**: New requests will bypass cache on first query
- **Within TTL**: All stale data cycles out automatically
- **No data loss**: Database remains source of truth

## Redis Configuration

### Persistence Settings

```redis
# Enable persistence for consistency
appendonly yes
appendfsync everysec

# Set eviction policy
maxmemory 2gb
maxmemory-policy allkeys-lru

# Enable notifications for monitoring invalidation
notify-keyspace-events Ex
```

### Backup Strategy

```bash
# Daily snapshot for disaster recovery
0 2 * * * redis-cli BGSAVE

# Weekly offline backup
0 3 * * 0 redis-cli BGSAVE && \
  cp /var/lib/redis/dump.rdb /backups/redis-$(date +\%Y\%m\%d).rdb
```

## Emergency Procedures

### Complete Cache Clear (Use with Caution)

```go
func EmergencyCacheClear(ctx context.Context, srv *Server) error {
    log.Printf("EMERGENCY: Clearing all cache. This should only be done if cache is corrupted.")

    if srv.QueryCache != nil {
        if err := srv.QueryCache.ClearAllCache(ctx); err != nil {
            return fmt.Errorf("failed to clear cache: %w", err)
        }
    }

    // Log the event
    LogInvalidationEvent(ctx, srv.DB, &CacheInvalidationAuditLog{
        TenantID:    "SYSTEM",
        Reason:      "emergency_clear",
        Scope:       "all",
        InitiatedBy: "admin",
    })

    log.Printf("Cache cleared successfully")
    return nil
}
```

### Redis Connection Failure

If Redis is unavailable:

1. **Automatic fallback**: Cache returns nil, queries process normally
2. **No data corruption**: All data remains in database
3. **Performance degradation**: Expect 10× slower queries until Redis restored
4. **Automatic recovery**: Once Redis reconnected, cache rehydrates naturally

```go
// Graceful degradation
if sqc.redisClient == nil {
    log.Printf("Cache unavailable - falling back to full pipeline")
    return nil  // Cache miss, query will regenerate everything
}
```

## Testing Invalidation

### Unit Tests

```go
func TestInvalidateOnBundleUpdate(t *testing.T) {
    // 1. Create cache entry
    cache.SetNLQueryCache(ctx, prompt, ds, mode, tenant, entry)

    // 2. Verify it exists
    retrieved, _ := cache.GetNLQueryCache(ctx, prompt, ds, mode, tenant)
    if retrieved == nil {
        t.Fatal("Entry should exist in cache")
    }

    // 3. Invalidate
    cache.InvalidateTenantCache(ctx, tenant)

    // 4. Verify it's gone
    retrieved, _ = cache.GetNLQueryCache(ctx, prompt, ds, mode, tenant)
    if retrieved != nil {
        t.Fatal("Entry should be missing after invalidation")
    }
}
```

### Integration Tests

```go
func TestCascadingInvalidation(t *testing.T) {
    // Verify that bundle update invalidates NL, SQL, and results
    // for all affected tenants
}

func TestConcurrentInvalidation(t *testing.T) {
    // Verify cache is consistent under concurrent invalidation
}
```

## Operational Procedures

### Weekly Cache Health Check

```bash
#!/bin/bash
# Check cache hit rate
curl http://localhost:8080/api/admin/cache-metrics \
  -H "X-Tenant-ID: tenant-1" | jq '.report.overall_hit_rate'

# Expected: >60%
```

### Monthly Cache Cleanup

```bash
#!/bin/bash
# Run once monthly on off-peak hours
redis-cli --eval prune_expired_keys.lua
```

### Documentation & Runbooks

- [ ] Cache invalidation troubleshooting guide
- [ ] Emergency cache clear procedures
- [ ] Escalation path for cache-related issues
- [ ] Cache hit rate SLA (target >60%)
- [ ] Cache memory usage monitoring (alert at 80%)

## Summary

The cache invalidation strategy provides:

1. **Automatic TTL-based cleanup** - No manual intervention for normal cases
2. **Event-driven invalidation** - Proactive invalidation for known changes
3. **Transparent failover** - Cache failure doesn't break functionality
4. **Audit trail** - All invalidations logged for compliance
5. **Emergency procedures** - Step-by-step recovery procedures

This ensures both high performance (cache hits) and high reliability (data consistency).
