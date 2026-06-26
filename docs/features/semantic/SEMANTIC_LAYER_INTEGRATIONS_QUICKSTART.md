# 🚀 Semantic Layer Integrations - Quick Start

Complete implementation guide for monitoring, SQL auditing, drift detection, and Redpanda/Kafka integration.

---

## ⚡ 5-Minute Setup

### 1. Create Database Tables

```bash
# Run the migration
psql postgres://postgres:postgres@localhost:5432/alpha \
  -f backend/sql/semantic_integrations.sql

# Verify tables created
psql postgres://postgres:postgres@localhost:5432/alpha -c \
  "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE 'semantic_%' ORDER BY table_name;"
```

### 2. Start RabbitMQ

```bash
# Option A: Docker
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3.12-management-alpine

# Option B: Docker Compose (add to docker-compose.yml)
docker-compose up -d rabbitmq

# Access Management UI
# http://localhost:15672 (guest/guest)
```

### 3. Update Go Application

**In your query handler** (e.g., `internal/api/query.go`):

```go
import (
    "github.com/eganpj/semlayer/backend/internal/events"
    "github.com/eganpj/semlayer/backend/internal/audit"
    "github.com/eganpj/semlayer/backend/internal/drift"
)

// In main or init:
publisher, _ := events.NewSemanticPublisher("amqp://guest:guest@localhost:5672/")
auditor := audit.NewQueryAuditor(db)
driftDetector := drift.NewDriftDetector(db)

// In query endpoint:
func QueryHandler(c *gin.Context) {
    // ... compile query ...
    
    // Record audit
    auditRecord := &audit.QueryAudit{
        TenantID: tenantID,
        UserID: userID,
        ModelID: modelID,
        CompiledSQL: sql,
        ExecutionStartTime: startTime,
        ExecutionEndTime: endTime,
        DurationMS: durationMs,
        CacheHit: cacheHit,
        Status: "success",
    }
    auditor.RecordQueryExecution(c, auditRecord)
    
    // Publish change event if model modified
    if modelModified {
        event := &events.SemanticChangeEvent{
            TenantID: tenantID,
            UserID: userID,
            ChangeType: "model_updated",
            ModelID: modelID,
            ElementType: "model",
            NewDefinition: newDef,
        }
        publisher.PublishModelChange(c, event)
    }
}
```

---

## 📊 What Gets Tracked

### 1. Query Auditing
✅ Every semantic query compilation  
✅ Compiled SQL with parameters  
✅ Execution time & row counts  
✅ Cache hits/misses  
✅ Query plans (EXPLAIN output)  
✅ Errors & timeouts  

**Access**: `semantic_query_audit` table

### 2. Change Tracking
✅ Model creates/updates/deletes  
✅ Measure additions/changes  
✅ Dimension modifications  
✅ Join edits  
✅ User who made change  
✅ Change reason  

**Access**: `semantic_layer_audit_log` + RabbitMQ events

### 3. Drift Detection
✅ Schema drift (missing columns)  
✅ Performance drift (50%+ slowdown)  
✅ Freshness drift (stale data)  
✅ Logic drift (computation changes)  
✅ Lineage drift (broken joins)  

**Access**: `semantic_drift_reports` & `semantic_drift_issues`

### 4. Event Publishing
✅ All model changes → RabbitMQ  
✅ Drift detected → alerts  
✅ Query anomalies → notifications  
✅ Cache invalidations → propagated  

**Access**: RabbitMQ exchanges:
- `semantic.changes` (topic)
- `semantic.drift` (topic)
- `semantic.audit` (topic)
- `semantic.notifications` (topic)

---

## 🔍 Viewing Audit Data

### 1. Query Performance

```sql
-- Top 10 slowest queries
SELECT model_id, compiled_sql, duration_ms, created_at
FROM semantic_query_audit
WHERE tenant_id = '...'
ORDER BY duration_ms DESC
LIMIT 10;

-- Cache hit rate
SELECT 
    SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / COUNT(*) as cache_hit_rate,
    AVG(duration_ms) as avg_duration_ms
FROM semantic_query_audit
WHERE tenant_id = '...'
AND created_at > now() - interval '24 hours';

-- Errors in past 24h
SELECT model_id, error_message, count(*) as error_count
FROM semantic_query_audit
WHERE status = 'error' AND created_at > now() - interval '24 hours'
GROUP BY model_id, error_message
ORDER BY error_count DESC;
```

### 2. Model Changes

```sql
-- Recent changes to a model
SELECT user_id, change_type, element_name, change_reason, created_at
FROM semantic_layer_audit_log
WHERE model_id = '...'
ORDER BY created_at DESC
LIMIT 20;

-- Who changed what and when
SELECT 
    DISTINCT ON (user_id, change_type) user_id, 
    change_type, 
    COUNT(*) as change_count,
    MAX(created_at) as last_change
FROM semantic_layer_audit_log
WHERE tenant_id = '...'
GROUP BY user_id, change_type
ORDER BY last_change DESC;
```

### 3. Drift Issues

```sql
-- Latest drift report for a model
SELECT * FROM semantic_drift_reports
WHERE model_id = '...'
ORDER BY report_time DESC
LIMIT 1;

-- All active drift issues
SELECT 
    r.model_id,
    r.drift_severity,
    i.issue_type,
    i.description,
    i.proposed_fix
FROM semantic_drift_reports r
JOIN semantic_drift_issues i ON r.id = i.report_id
WHERE r.status = 'open' AND r.tenant_id = '...'
ORDER BY r.drift_severity DESC, r.report_time DESC;

-- Count issues by severity
SELECT drift_severity, COUNT(*) as count
FROM semantic_drift_reports
WHERE tenant_id = '...' AND created_at > now() - interval '7 days'
GROUP BY drift_severity;
```

---

## 🐇 RabbitMQ Integration

### Publishing a Model Change

```go
import "github.com/eganpj/semlayer/backend/internal/events"

publisher, _ := events.NewSemanticPublisher("amqp://guest:guest@localhost:5672/")

event := &events.SemanticChangeEvent{
    ID: uuid.New().String(),
    Timestamp: time.Now(),
    TenantID: "tenant-123",
    UserID: "user-456",
    ChangeType: "measure_added",
    ModelID: "model-789",
    ModelName: "revenue_analytics",
    ElementType: "measure",
    ElementID: "m_total_revenue",
    ElementName: "total_revenue",
    ChangeReason: "Added new financial metric for quarterly reporting",
    NewDefinition: json.RawMessage(`{"type": "sum", "field": "amount"}`),
}

publisher.PublishModelChange(context.Background(), event)
```

### Consuming Events (Cache Invalidation Example)

```go
import "github.com/eganpj/semlayer/backend/internal/events"

// In your startup code:
sub, _ := events.NewCacheInvalidationSubscriber(amqpConn, cacheManager)
sub.Start(context.Background())
```

The subscriber will automatically:
- Invalidate `semantic:model:{modelID}:*` patterns
- Clear query result caches
- Invalidate aggregation caches
- Update metadata caches

---

## 📈 Monitoring Dashboard (Grafana)

### Queries to Add to Dashboard

```
# 1. Query execution rate (QPS)
rate(semantic_queries_total[5m])

# 2. Query latency (p95)
histogram_quantile(0.95, semantic_query_duration_ms)

# 3. Cache hit rate
semantic_cache_hit_rate

# 4. Model changes per hour
rate(semantic_model_changes_total[1h])

# 5. Drift issues detected
increase(semantic_drift_issues_total[1h])

# 6. Slow queries (>1s)
rate(semantic_slow_queries_total[5m])

# 7. Query compilation time
histogram_quantile(0.95, semantic_compilation_duration_ms)

# 8. Error rate
rate(semantic_query_errors_total[5m])
```

### Create Alerts

```yaml
groups:
  - name: semantic_layer
    rules:
      - alert: HighQueryLatency
        expr: histogram_quantile(0.95, semantic_query_duration_ms) > 1000
        for: 5m
        
      - alert: LowCacheHitRate
        expr: semantic_cache_hit_rate < 0.7
        for: 10m
        
      - alert: HighDriftSeverity
        expr: semantic_drift_issues_total{severity="critical"} > 0
        for: 1m
        
      - alert: HighErrorRate
        expr: rate(semantic_query_errors_total[5m]) > 0.05
        for: 5m
```

---

## 🤖 AI Suggestions (Optional)

### Enable Auto-Suggestions

```go
import "github.com/eganpj/semlayer/backend/internal/suggestions"

suggester := suggestions.NewSemanticSuggester(db)

// Suggest join optimizations
joinOptimizations, _ := suggester.SuggestJoinOptimization(ctx, tenantID, modelID)

// Suggest pre-aggregations
preAggs, _ := suggester.SuggestPreAggregation(ctx, tenantID)

// Suggest measure reuse
reuse, _ := suggester.SuggestMeasureReuse(ctx, tenantID, modelID)
```

### View Suggestions in Database

```sql
-- All new suggestions
SELECT 
    suggestion_type, 
    priority, 
    title, 
    confidence_score,
    expected_benefit
FROM semantic_suggestions
WHERE status = 'new'
ORDER BY confidence_score DESC
LIMIT 10;
```

---

## 🔧 Configuration

### Environment Variables

```bash
# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Database
DATABASE_URL=postgres://postgres:postgres@localhost:5432/alpha

# Drift Detection
DRIFT_CHECK_INTERVAL=1h  # How often to run drift detection
DRIFT_FRESHNESS_MAX_HOURS=24  # Max age before freshness drift

# Query Audit
QUERY_AUDIT_RETENTION_DAYS=90  # How long to keep audit logs
QUERY_SLOW_THRESHOLD_MS=1000  # What's considered "slow"
```

### Startup Configuration

```go
// In main.go
func init() {
    // Initialize RabbitMQ publisher
    publisher, err := events.NewSemanticPublisher(os.Getenv("RABBITMQ_URL"))
    if err != nil {
        log.Fatal(err)
    }
    
    // Initialize subscribers
    cache := initCache() // your cache implementation
    cacheInvalSub, _ := events.NewCacheInvalidationSubscriber(amqpConn, cache)
    cacheInvalSub.Start(context.Background())
    
    driftDetector := drift.NewDriftDetector(db)
    go driftDetector.ScheduleDriftDetection(
        context.Background(), 
        "tenant-id", 
        time.Hour,
    )
    
    // Cleanup old audits daily
    go func() {
        ticker := time.NewTicker(24 * time.Hour)
        for range ticker.C {
            auditor := audit.NewQueryAuditor(db)
            auditor.CleanupOldAudits(context.Background(), 90)
        }
    }()
}
```

---

## ✅ Verification Checklist

- [ ] RabbitMQ running (`docker ps | grep rabbitmq`)
- [ ] All tables created (`psql ... -c "SELECT COUNT(*) FROM semantic_query_audit"`)
- [ ] Events being published to RabbitMQ
- [ ] Audit logs being recorded
- [ ] Drift detection running on schedule
- [ ] Cache invalidation working
- [ ] Query performance visible in database
- [ ] Suggestions being generated

---

## 📊 API Endpoints (To Implement)

```bash
# Publish a model change
POST /api/v1/semantic/models/{modelId}/publish-change
{
  "changeType": "measure_added",
  "elementType": "measure",
  "elementName": "revenue_total",
  "changeReason": "Q4 reporting requirement"
}

# Get query audit trail
GET /api/v1/semantic/models/{modelId}/audit?limit=50

# Get drift report
GET /api/v1/semantic/drift/reports?model_id={modelId}

# Get query performance stats
GET /api/v1/semantic/performance/stats?model_id={modelId}&hours=24

# Get suggestions
GET /api/v1/semantic/suggestions?model_id={modelId}&type=pre_aggregation

# Accept/reject suggestion
POST /api/v1/semantic/suggestions/{suggestionId}/feedback
{
  "action": "accepted",
  "reason": "Implemented pre-aggregation"
}
```

---

## 🐛 Troubleshooting

### RabbitMQ Connection Issues

```bash
# Check if RabbitMQ is running
docker logs rabbitmq

# Check connection
rabbitmq-diagnostics -q ping

# Check queues
docker exec rabbitmq rabbitmqctl list_queues
```

### Drift Detection Not Running

```bash
# Check logs for errors
tail -f /var/log/semantic-layer.log | grep "drift"

# Manually trigger detection
curl -X POST http://localhost:8080/api/v1/semantic/drift/detect
```

### Query Audits Not Recording

```sql
-- Check for errors
SELECT error_message, COUNT(*) 
FROM semantic_query_audit 
WHERE status = 'error' 
GROUP BY error_message;

-- Check if user has permissions
SELECT * FROM semantic_layer_audit_log 
WHERE tenant_id = '...' 
LIMIT 5;
```

---

## 📚 Further Reading

- [RabbitMQ Tutorials](https://www.rabbitmq.com/getstarted.html)
- [PostgreSQL JSONB Guide](https://www.postgresql.org/docs/current/datatype-json.html)
- [Prometheus Metrics](https://prometheus.io/docs/concepts/metric_types/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)

---

**Status**: ✅ Ready to implement  
**Time to Production**: 1-2 weeks  
**Team Size**: 1-2 engineers  

Start with Phase 1 (RabbitMQ & Events), then build audit/drift/monitoring incrementally.
