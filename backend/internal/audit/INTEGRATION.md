# Audit & Snapshot Plane Integration Guide

This guide shows how to wire the Audit Plane into your existing SemLayer services.

## 🚀 Quick Start

### 1. Start Infrastructure

```bash
cd backend/audit-infrastructure
chmod +x start.sh test.sh
./start.sh
```

This starts: Redpanda (Kafka), MinIO (S3), Iceberg REST Catalog, Trino, and the audit sink consumer.

### 2. Verify Setup

```bash
./test.sh
```

Checks all services, publishes test events, queries Trino.

---

## 📦 Wire Up Publishers

### Scheduler Integration

**File**: `backend/scheduler-engine/scheduler.go`

```go
import (
    "github.com/semlayer/backend/internal/audit"
)

type Scheduler struct {
    auditPublisher *audit.KafkaPublisher
    // ... existing fields
}

func NewScheduler(cfg *config.Config) (*Scheduler, error) {
    // Initialize audit publisher
    pub, err := audit.InitializeAuditPublisher(cfg.Kafka.Brokers[0])
    if err != nil {
        return nil, fmt.Errorf("init audit: %w", err)
    }
    
    return &Scheduler{
        auditPublisher: pub,
        // ... other init
    }, nil
}

// When a job completes
func (s *Scheduler) onJobComplete(job *Job) {
    record := &audit.SchedulerJobRun{
        RunID:          job.RunID,
        JobID:          job.JobID,
        TenantID:       job.TenantID,
        Status:         string(job.Status),
        StartTS:        job.StartTime,
        EndTS:          &job.EndTime,
        DurationMS:     job.Duration.Milliseconds(),
        InputParams:    job.InputParams,
        OutputArtifact: job.OutputPath,
        ErrorMsg:       job.Error,
        RetryCount:     job.RetryCount,
        WorkerNodeID:   job.WorkerNode,
    }
    
    if err := s.auditPublisher.PublishJobRun(context.Background(), record); err != nil {
        log.Errorf("Failed to publish job audit: %v", err)
    }
}

// When a DAG completes
func (s *Scheduler) onDAGComplete(dag *DAG) {
    record := &audit.SchedulerDAGRun{
        DAGRunID:    dag.RunID,
        DAGName:     dag.Name,
        TenantID:    dag.TenantID,
        Status:      string(dag.Status),
        StartTS:     dag.StartTime,
        EndTS:       &dag.EndTime,
        TotalTasks:  dag.TotalTasks,
        TasksFailed: dag.FailedTasks,
        TriggerType: dag.TriggerType,
    }
    
    s.auditPublisher.PublishDAGRun(context.Background(), record)
}
```

### Governance Integration

**File**: `backend/governance/governance_service.go`

```go
import "github.com/semlayer/backend/internal/audit"

type GovernanceService struct {
    auditPublisher *audit.KafkaPublisher
}

func (g *GovernanceService) ApplyPolicy(ctx context.Context, change PolicyChange) error {
    // ... apply policy
    
    // Audit the change
    changeset := &audit.GovernanceChangeSet{
        ChangeSetID:    uuid.New().String(),
        TenantID:       change.TenantID,
        ChangeType:     "POLICY_UPDATE",
        ResourceType:   change.ResourceType,
        ResourceID:     change.ResourceID,
        ActorID:        change.ActorID,
        ActorType:      change.ActorType,
        ChangedFields:  change.Fields,
        BeforeSnapshot: change.Before,
        AfterSnapshot:  change.After,
        ChangeTS:       time.Now(),
        AuditReason:    change.Reason,
    }
    
    return g.auditPublisher.PublishChangeSet(ctx, changeset)
}
```

### Semantic Engine Integration

**File**: `backend/semantic-engine/term_service.go`

```go
func (s *SemanticService) UpdateTerm(ctx context.Context, termID string, updates map[string]interface{}) error {
    before := s.GetTerm(termID)
    
    // ... apply updates
    
    after := s.GetTerm(termID)
    
    // Audit semantic snapshot
    snapshot := &audit.SemanticSnapshot{
        SnapshotID:       uuid.New().String(),
        TenantID:         before.TenantID,
        ObjectType:       "TERM",
        ObjectID:         termID,
        ObjectVersion:    after.Version,
        SnapshotTS:       time.Now(),
        FullPayload:      after,
        SchemaEvolutionID: after.SchemaID,
        LineageGraphID:   after.LineageID,
    }
    
    return s.auditPublisher.PublishSemanticSnapshot(ctx, snapshot)
}
```

### Compliance Engine Integration

**File**: `backend/compliance-engine/violation_detector.go`

```go
func (c *ComplianceEngine) DetectViolations(ctx context.Context, resource Resource) {
    violations := c.runChecks(resource)
    
    for _, v := range violations {
        record := &audit.ComplianceViolation{
            ViolationID:    uuid.New().String(),
            TenantID:       resource.TenantID,
            ViolationType:  v.Type,
            Severity:       v.Severity,
            ResourceType:   resource.Type,
            ResourceID:     resource.ID,
            PolicyID:       v.PolicyID,
            DetectedTS:     time.Now(),
            RuleViolated:   v.Rule,
            ActualValue:    v.ActualValue,
            ExpectedValue:  v.ExpectedValue,
            RemediationSLA: v.SLA,
        }
        
        c.auditPublisher.PublishComplianceViolation(ctx, record)
    }
}
```

---

## 🌐 Wire Up API

**File**: `backend/internal/api/api.go`

```go
import (
    "github.com/semlayer/backend/internal/audit"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config) {
    // ... existing routes
    
    // Initialize audit query service
    trinoQuerier, err := audit.NewTrinoQuerier(cfg.Trino.Host, cfg.Trino.Port)
    if err != nil {
        log.Fatalf("Failed to init Trino: %v", err)
    }
    
    aiService := audit.NewAIAuditNarrativeService(cfg.AI.APIKey, cfg.AI.Model)
    
    // Register audit routes
    audit.RegisterAuditRoutes(r, trinoQuerier, aiService)
    
    log.Info("Audit API routes registered at /api/audit/*")
}
```

---

## ⚙️ Configuration

**File**: `backend/config.yaml`

```yaml
# Existing config...

kafka:
  brokers:
    - "localhost:19092"
  consumer_group: "audit-sink-group"

trino:
  host: "localhost"
  port: 8090
  catalog: "iceberg"
  schema: "audit"

iceberg:
  catalog_uri: "http://localhost:8181"
  warehouse: "s3://warehouse/audit"
  
s3:
  endpoint: "http://localhost:9000"
  access_key: "minioadmin"
  secret_key: "minioadmin"
  bucket: "warehouse"

ai:
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4"
```

---

## 🎨 Wire Up Frontend

### Add Route

**File**: `frontend/src/App.tsx`

```tsx
import AuditExplorer from './components/audit/AuditExplorer';

function App() {
  return (
    <Routes>
      {/* existing routes */}
      <Route path="/audit" element={<AuditExplorer />} />
    </Routes>
  );
}
```

### Add Navigation

**File**: `frontend/src/components/Sidebar.tsx`

```tsx
import { FileSearch } from 'lucide-react';

<NavLink to="/audit">
  <FileSearch className="w-5 h-5" />
  <span>Audit Trail</span>
</NavLink>
```

---

## 🧪 Testing

### 1. Manual Test

```bash
cd backend
go run scripts/test-audit-event.go
```

### 2. Query Trino

```bash
docker exec -it audit-trino trino
```

```sql
USE iceberg.audit;

-- View recent job runs
SELECT run_id, job_id, status, start_ts, duration_ms 
FROM scheduler_job_runs 
WHERE tenant_id = 'tenant-test-001'
ORDER BY start_ts DESC 
LIMIT 10;

-- Compliance violations by severity
SELECT severity, COUNT(*) as count, AVG(detection_lag_ms) as avg_lag
FROM compliance_violations
WHERE tenant_id = 'your-tenant'
  AND detected_ts > CURRENT_TIMESTAMP - INTERVAL '7' DAY
GROUP BY severity;

-- Governance activity
SELECT change_type, COUNT(*) as changes
FROM governance_changesets
WHERE tenant_id = 'your-tenant'
  AND change_ts > CURRENT_TIMESTAMP - INTERVAL '1' DAY
GROUP BY change_type;
```

### 3. Test API

```bash
# Get job runs
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/job-runs?limit=10

# Get violations
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/violations?severity=critical

# Get AI narrative
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/ai/explain/job/run-123

# Dashboard data
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/dashboard/slo
```

---

## 📊 Monitoring

### Check Kafka Lag

```bash
docker exec audit-redpanda rpk group describe audit-sink-group
```

### Check Parquet Files

```bash
# View files in MinIO
docker exec audit-minio mc ls local/warehouse/audit/scheduler_job_runs/
```

### Check Trino Query Performance

```sql
-- View recent queries
SELECT query_id, query, state, elapsed_time_millis
FROM system.runtime.queries
WHERE state = 'FINISHED'
ORDER BY end_time DESC
LIMIT 10;
```

### Check Sink Consumer Logs

```bash
docker logs -f audit-sink
```

---

## 🔒 Security Considerations

### Tenant Isolation

All queries **MUST** include `tenant_id` in WHERE clause:

```go
// WRONG - query across all tenants
query := "SELECT * FROM scheduler_job_runs"

// RIGHT - scoped to single tenant
query := fmt.Sprintf(
    "SELECT * FROM scheduler_job_runs WHERE tenant_id = '%s'",
    tenantID,
)
```

The API middleware enforces `X-Tenant-ID` header on all endpoints.

### PII Handling

Mark PII fields in config:

```yaml
audit:
  pii_fields:
    - "user_email"
    - "customer_name"
    - "ssn"
  retention_days: 90
```

The compliance reporter tracks PII exposure.

---

## 📈 Scaling

### Increase Kafka Partitions

```bash
docker exec audit-redpanda rpk topic alter-config audit.scheduler.job_runs \
  --set partition.count=12
```

### Add More Sink Consumers

```bash
docker-compose up -d --scale audit-sink=3
```

### Scale Trino Workers

Add to `docker-compose.yml`:

```yaml
  trino-worker-1:
    image: trinodb/trino:435
    environment:
      - TRINO_ENVIRONMENT=production
      - TRINO_COORDINATOR=false
      - TRINO_DISCOVERY_URI=http://audit-trino:8090
```

---

## 🎯 Production Checklist

- [ ] Kafka authentication enabled (SASL/SCRAM)
- [ ] Trino TLS configured
- [ ] S3/MinIO access keys rotated
- [ ] Iceberg table retention policies set
- [ ] Monitoring dashboards deployed
- [ ] Alert rules configured
- [ ] Backup strategy defined
- [ ] Disaster recovery tested
- [ ] PII redaction rules active
- [ ] Tenant quotas enforced
- [ ] API rate limiting enabled
- [ ] Audit log rotation configured

---

## 🆘 Troubleshooting

### Events Not Appearing in Iceberg

1. Check Kafka topics:
   ```bash
   docker exec audit-redpanda rpk topic list
   docker exec audit-redpanda rpk topic consume audit.scheduler.job_runs
   ```

2. Check sink consumer logs:
   ```bash
   docker logs audit-sink
   ```

3. Verify Parquet files:
   ```bash
   docker exec audit-minio mc ls local/warehouse/audit/scheduler_job_runs/
   ```

### Trino Query Errors

1. Check catalog connection:
   ```bash
   docker exec audit-trino trino --execute "SHOW CATALOGS"
   ```

2. Refresh metadata:
   ```sql
   CALL iceberg.system.expire_snapshots('audit', 'scheduler_job_runs', TIMESTAMP '2024-01-01 00:00:00');
   ```

### High Kafka Lag

1. Check consumer group:
   ```bash
   docker exec audit-redpanda rpk group describe audit-sink-group
   ```

2. Scale up consumers:
   ```bash
   docker-compose up -d --scale audit-sink=3
   ```

3. Increase batch size in `iceberg_sink.go`:
   ```go
   batchSize := 1000 // increase from 100
   ```

---

## 📚 Next Steps

1. **Extend Models**: Add custom audit models for your domain
2. **Custom Views**: Create materialized views for your dashboards
3. **AI Narratives**: Fine-tune AI prompts in `ai_narrative_service.go`
4. **Compliance Rules**: Add regulatory report formats
5. **Alerts**: Wire up Kafka streams to alerting systems
6. **Export**: Add S3/GCS export for long-term archival

---

**Questions?** Check the main [README.md](README.md) or review [models.go](models.go) for schema details.
