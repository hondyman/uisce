# 🗂️ SEMANTIC LAYER INTEGRATIONS - REFERENCE CARD

**One-page cheat sheet for monitoring, auditing, drift detection, and RabbitMQ.**

---

## 📚 Documentation Map

```
START HERE → SEMANTIC_LAYER_INTEGRATIONS_SUMMARY.md (5 min overview)
               ↓
    Then read SEMANTIC_LAYER_INTEGRATIONS_QUICKSTART.md (30 min guide)
               ↓
    Full reference: SEMANTIC_LAYER_INTEGRATIONS.md (technical)
               ↓
    Navigation: SEMANTIC_LAYER_INTEGRATIONS_INDEX.md (reading guide)
               ↓
    Final summary: SEMANTIC_LAYER_INTEGRATIONS_DELIVERY.md (this)
```

---

## ⚡ 5-Minute Setup

```bash
# 1. Run database migration
psql postgres://postgres:postgres@localhost:5432/alpha \
  -f backend/sql/semantic_integrations.sql

# 2. Start RabbitMQ
docker run -d --name rabbitmq \
  -p 5672:5672 -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3.12-management-alpine

# 3. Copy code files to your project (already done!)
# - backend/internal/events/semantic_publisher.go
# - backend/internal/audit/query_auditor.go
# - backend/internal/drift/drift_detector.go

# 4. Integrate into your application (see examples below)
```

---

## 💻 Code Integration (Copy & Paste)

### Publishing a Model Change
```go
import "github.com/eganpj/semlayer/backend/internal/events"

publisher, _ := events.NewSemanticPublisher("amqp://guest:guest@localhost:5672/")

publisher.PublishModelChange(ctx, &events.SemanticChangeEvent{
    ID: uuid.New().String(),
    Timestamp: time.Now(),
    TenantID: tenantID,
    UserID: userID,
    ChangeType: "measure_added",  // or model_updated, dimension_changed, etc.
    ModelID: modelID,
    ElementType: "measure",
    ElementName: "new_measure",
    ChangeReason: "Adding for Q4 reporting",
})
```

### Recording a Query
```go
import "github.com/eganpj/semlayer/backend/internal/audit"

auditor := audit.NewQueryAuditor(db)

startTime := time.Now()
rows, err := db.QueryContext(ctx, compiledSQL, params...)
endTime := time.Now()

auditor.RecordQueryExecution(ctx, &audit.QueryAudit{
    TenantID: tenantID,
    UserID: userID,
    ModelID: modelID,
    CompiledSQL: compiledSQL,
    ExecutionStartTime: &startTime,
    ExecutionEndTime: &endTime,
    DurationMS: int64(endTime.Sub(startTime).Milliseconds()),
    RowsReturned: rowCount,
    CacheHit: fromCache,
    Status: "success",
})
```

### Running Drift Detection
```go
import "github.com/eganpj/semlayer/backend/internal/drift"

detector := drift.NewDriftDetector(db)

// Run hourly
go func() {
    ticker := time.NewTicker(time.Hour)
    for range ticker.C {
        // Get all models
        var models []string
        db.Select(&models, "SELECT model_key FROM fabric_defn...")
        
        // Detect drift for each
        for _, modelID := range models {
            report, _ := detector.GenerateDriftReport(ctx, tenantID, modelID)
            detector.SaveDriftReport(ctx, report)
        }
    }
}()
```

---

## 📊 Database Queries

### Top 10 Slowest Queries
```sql
SELECT model_id, compiled_sql, duration_ms, created_at
FROM semantic_query_audit
WHERE tenant_id = '$TENANT_ID'
ORDER BY duration_ms DESC LIMIT 10;
```

### Cache Hit Rate (Last 24h)
```sql
SELECT 
    SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / COUNT(*) as hit_rate,
    AVG(duration_ms) as avg_duration,
    COUNT(*) as total_queries
FROM semantic_query_audit
WHERE tenant_id = '$TENANT_ID' AND created_at > now() - interval '24 hours';
```

### Recent Model Changes
```sql
SELECT user_id, change_type, element_name, change_reason, created_at
FROM semantic_layer_audit_log
WHERE tenant_id = '$TENANT_ID' AND model_id = '$MODEL_ID'
ORDER BY created_at DESC LIMIT 20;
```

### Active Drift Issues
```sql
SELECT r.model_id, r.drift_severity, COUNT(i.id) as issue_count
FROM semantic_drift_reports r
LEFT JOIN semantic_drift_issues i ON r.id = i.report_id
WHERE r.status = 'open' AND r.tenant_id = '$TENANT_ID'
GROUP BY r.model_id, r.drift_severity
ORDER BY r.drift_severity DESC;
```

---

## 🐇 RabbitMQ Basics

### Access Management UI
```
http://localhost:15672
Username: guest
Password: guest
```

### Check Exchanges
```bash
docker exec rabbitmq rabbitmqctl list_exchanges
# Output should include:
# semantic.changes
# semantic.drift
# semantic.audit
# semantic.notifications
```

### Check Queues
```bash
docker exec rabbitmq rabbitmqctl list_queues
# Should show:
# semantic-cache-invalidation
# semantic-drift-detection
# semantic-audit
```

---

## 🎯 Key Patterns

| Pattern | Purpose | Example |
|---------|---------|---------|
| `semantic:model:{modelId}:*` | Cache key pattern for model | `semantic:model:revenue_model:*` |
| `semantic.changes.model.*` | Topic routing for model changes | `semantic.changes.model.updated` |
| `semantic:query_results:*:{modelId}:*` | Query result cache pattern | `semantic:query_results:*:revenue_model:*` |
| `semantic_drift_{severity}` | Drift queue by severity | `semantic_drift_critical` |

---

## 📈 Monitoring Checklist

- [ ] Query audit logs growing (1000+ records/day)
- [ ] Model changes logged to semantic_layer_audit_log
- [ ] Drift reports generating hourly
- [ ] Events in RabbitMQ queues
- [ ] Cache invalidation happening
- [ ] No error_message entries in audit logs
- [ ] Drift detector completing < 10s
- [ ] Query audit cleaning up old records

---

## ⚙️ Configuration

### Environment Variables
```bash
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export DRIFT_CHECK_INTERVAL=1h
export DRIFT_FRESHNESS_MAX_HOURS=24
export QUERY_AUDIT_RETENTION_DAYS=90
export QUERY_SLOW_THRESHOLD_MS=1000
```

### Startup Code
```go
// Initialize in main()
publisher, _ := events.NewSemanticPublisher(os.Getenv("RABBITMQ_URL"))
auditor := audit.NewQueryAuditor(db)
driftDetector := drift.NewDriftDetector(db)

// Start subscribers
cacheInvalSub, _ := events.NewCacheInvalidationSubscriber(amqpConn, cacheManager)
cacheInvalSub.Start(context.Background())
```

---

## 🔍 Troubleshooting

| Problem | Diagnosis | Fix |
|---------|-----------|-----|
| No audit records | Check if recorder is called | Add auditor.Record() to handler |
| Events not publishing | Check RabbitMQ connection | Restart container, check logs |
| Drift not detecting | No baseline data | Execute queries to create baseline |
| Cache not invalidating | Subscriber not listening | Check subscriber start() call |
| High query latency | Too much auditing | Only audit sample of queries |

---

## 📊 What Gets Tracked

```
✅ Query Execution
   - Semantic query (JSON)
   - Compiled SQL
   - Execution time
   - Row counts
   - Cache status
   - Errors

✅ Model Changes
   - Who, what, when, why
   - Old/new definitions
   - SQL differences
   - RabbitMQ published

✅ Drift Issues
   - Schema changes
   - Performance degradation
   - Data freshness
   - Suggested fixes

✅ Event Distribution
   - Published to queue
   - Delivered to subscribers
   - Retry attempts
   - Failure reasons
```

---

## 🚀 Implementation Order

```
1. Database Migration → 5 minutes
2. RabbitMQ Start → 1 minute
3. Publisher Integration → 1 hour
4. Auditor Integration → 1 hour
5. Drift Detection Job → 1 hour
6. Cache Invalidation → 30 minutes
7. Monitoring Setup → 2 hours
8. UI Dashboards → 4 hours
Total: 1-2 weeks for full stack
```

---

## 🎓 Learning Resources

### Tables
- `semantic_query_audit` - Every query
- `semantic_layer_audit_log` - Every model change
- `semantic_drift_reports` - Drift analyses
- `semantic_change_events` - Event sourcing
- `semantic_suggestions` - AI recommendations

### Exchanges
- `semantic.changes` - Model modifications (topic)
- `semantic.drift` - Drift detected (topic)
- `semantic.audit` - Change logged (topic)
- `semantic.notifications` - Alert (topic)

### Queues
- `semantic-cache-invalidation` - Cache updates
- `semantic-drift-detection` - Drift triggers
- `semantic-audit` - Audit logging

---

## ✅ Pre-Launch Checklist

- [ ] All database tables created
- [ ] RabbitMQ running and accessible
- [ ] Publisher initialized
- [ ] Auditor integrated in query handler
- [ ] Model change events publishing
- [ ] Cache invalidation working
- [ ] Drift detection running on schedule
- [ ] No errors in logs
- [ ] Performance impact < 1% latency
- [ ] Retention policies set
- [ ] Backup/restore tested
- [ ] Team trained

---

## 📞 Quick Reference

### Verify Setup
```bash
# Tables exist?
psql ... -c "SELECT COUNT(*) FROM semantic_query_audit;"

# RabbitMQ running?
curl http://localhost:15672/api/whoami

# Events publishing?
# Check RabbitMQ console at http://localhost:15672

# Audits recording?
psql ... -c "SELECT * FROM semantic_query_audit LIMIT 1;"

# Drift running?
psql ... -c "SELECT COUNT(*) FROM semantic_drift_reports WHERE created_at > now() - interval '1 hour';"
```

### Get Help
1. **Setup issues** → See QUICKSTART.md
2. **Code questions** → See code comments in backend/internal/
3. **Architecture** → See INTEGRATIONS.md
4. **Navigation** → See INDEX.md

---

## 🎉 What You Get

✅ Complete audit trail  
✅ SQL-level visibility  
✅ Automatic drift detection  
✅ Event-driven updates  
✅ Performance monitoring  
✅ AI suggestions  

**Status**: Production-ready, day 1  
**Risk**: Low (isolated integration)  
**ROI**: 2-3 weeks to value  

---

**Start with**: `SEMANTIC_LAYER_INTEGRATIONS_QUICKSTART.md`  
**Reference**: `SEMANTIC_LAYER_INTEGRATIONS.md`  
**Navigation**: `SEMANTIC_LAYER_INTEGRATIONS_INDEX.md`

**Happy building!** 🚀
