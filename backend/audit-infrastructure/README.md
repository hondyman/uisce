# 🚀 Audit Infrastructure Quick Start

## Prerequisites
- Docker & Docker Compose installed
- Postgres running on localhost:5432 (your existing database)
- Frontend running on localhost (your existing frontend)

## Infrastructure Components

This Docker Compose stack provides:
- **Redpanda** (Kafka) - Message bus for audit events (port 19092)
- **Redpanda Console** - Kafka UI (port 8080)
- **MinIO** - S3-compatible object storage (ports 9000, 9001)
- **Iceberg REST Catalog** - Table metadata service (port 8181)
- **Trino** - SQL query engine (port 8090)
- **Audit Sink** - Kafka → Iceberg consumer

## Quick Start

### 1. Start Infrastructure

```bash
cd backend/audit-infrastructure
chmod +x start.sh
./start.sh
```

This will:
- Start all Docker containers
- Wait for services to be healthy
- Create Kafka topics
- Create Iceberg tables and views
- Display service endpoints

### 2. Verify Services

```bash
# Check all containers are running
docker-compose ps

# View logs
docker-compose logs -f

# Test Kafka
docker exec audit-redpanda rpk topic list

# Test Trino
docker exec -it audit-trino trino
> USE iceberg.audit;
> SHOW TABLES;
> quit;

# View MinIO Console
open http://localhost:9001
# Login: minioadmin / minioadmin
```

### 3. Integration Points

#### A. Update config.yaml

Add to your `backend/config.yaml`:

```yaml
# Audit Infrastructure
audit:
  enabled: true
  kafka_brokers: "localhost:19092"
  trino_host: "localhost"
  trino_port: 8090
  
# AI Service (for audit narratives)
ai:
  endpoint: "http://localhost:8000"  # Your AI orchestration endpoint
```

#### B. Wire Up Publishers

In your scheduler service (when a job completes):

```go
import "github.com/hondyman/semlayer/backend/internal/audit"

// Initialize once at startup
auditPublisher, err := audit.InitializeAuditPublisher("localhost:19092")
if err != nil {
    log.Fatal(err)
}
defer auditPublisher.Close()

// Publish job run events
err = auditPublisher.PublishJobRun(ctx, audit.JobRunCompletedEvent{
    RunID:    runID,
    JobID:    jobID,
    TenantID: tenantID,
    StartTS:  startTime,
    EndTS:    time.Now(),
    Status:   "FAILED",
    ErrorMessage: err.Error(),
    SemanticContext: json.RawMessage(semanticJSON),
    ComplianceContext: json.RawMessage(complianceJSON),
    SLOContext: json.RawMessage(sloJSON),
})
```

#### C. Add Audit API Routes

In your main API server:

```go
import "github.com/hondyman/semlayer/backend/internal/audit"

// Initialize Trino querier
trinoQuerier, err := audit.NewTrinoAuditQuerier("localhost", 8090, "iceberg", "audit")
if err != nil {
    log.Fatal(err)
}

// Create audit handler
auditHandler := audit.NewAuditAPIHandler(trinoQuerier)

// Register routes
api := router.Group("/api")
api.Use(audit.TenantScopeMiddleware())
auditHandler.RegisterRoutes(api)
```

#### D. Add Frontend Route

In your frontend router:

```tsx
import AuditExplorer from '@/components/audit/AuditExplorer';

// Add route
<Route 
  path="/audit" 
  element={
    <AuditExplorer 
      tenantId={selectedTenant.id} 
      tenantName={selectedTenant.name} 
    />
  } 
/>
```

## Service Endpoints

| Service | Endpoint | Purpose |
|---------|----------|---------|
| Kafka | localhost:19092 | Publish audit events |
| Redpanda Console | http://localhost:8080 | View Kafka topics/messages |
| MinIO S3 API | http://localhost:9000 | Parquet file storage |
| MinIO Console | http://localhost:9001 | Browse S3 buckets |
| Iceberg REST | http://localhost:8181 | Table metadata |
| Trino | http://localhost:8090 | Query audit data |

## Testing the Pipeline

### 1. Publish a Test Event

```bash
# From backend directory
go run scripts/test-audit-event.go
```

Or manually via Kafka:

```bash
docker exec -it audit-redpanda rpk topic produce audit.scheduler.job_runs
# Paste JSON, press Ctrl+D
```

### 2. Query via Trino

```sql
docker exec -it audit-trino trino

-- Check job runs
USE iceberg.audit;
SELECT * FROM scheduler_job_runs LIMIT 10;

-- Check materialized views
SELECT * FROM mv_tenant_scheduler_slo 
WHERE tenant_id = 'your-tenant-id' 
ORDER BY run_date DESC 
LIMIT 7;
```

### 3. Test API

```bash
# Get job runs
curl -H "X-Tenant-ID: your-tenant-id" \
  http://localhost:8080/api/audit/job-runs

# Get compliance violations
curl -H "X-Tenant-ID: your-tenant-id" \
  http://localhost:8080/api/audit/violations

# Get SLO dashboard
curl -H "X-Tenant-ID: your-tenant-id" \
  http://localhost:8080/api/audit/dashboard/slo
```

### 4. View in UI

Navigate to: http://localhost:3000/audit

## Monitoring

### View Kafka Messages

```bash
# List topics
docker exec audit-redpanda rpk topic list

# Consume messages
docker exec audit-redpanda rpk topic consume audit.scheduler.job_runs
```

### View Parquet Files

```bash
# List files in MinIO
docker exec audit-minio-init mc ls myminio/audit/scheduler_job_runs/
```

### View Trino Queries

```bash
# Show running queries
docker exec audit-trino trino --execute "SELECT * FROM system.runtime.queries;"
```

## Troubleshooting

### Services Won't Start

```bash
# Check logs
docker-compose logs redpanda
docker-compose logs trino

# Restart specific service
docker-compose restart redpanda
```

### Kafka Connection Issues

```bash
# Test from host
docker exec audit-redpanda rpk cluster info

# Verify topic creation
docker exec audit-redpanda rpk topic list
```

### Trino Can't Query Tables

```bash
# Check catalog
docker exec audit-trino trino --execute "SHOW CATALOGS;"

# Check schemas
docker exec audit-trino trino --execute "SHOW SCHEMAS IN iceberg;"

# Recreate tables
cd backend/internal/audit
docker exec -i audit-trino trino < iceberg_schema.sql
```

### MinIO Connection Issues

```bash
# Check buckets
docker exec audit-minio-init mc ls myminio/

# Recreate buckets
docker exec audit-minio-init mc mb myminio/audit --ignore-existing
```

## Stopping the Infrastructure

```bash
# Stop but keep data
docker-compose stop

# Stop and remove containers (keeps volumes)
docker-compose down

# Stop and remove everything (WARNING: deletes all audit data)
docker-compose down -v
```

## Performance Tuning

### For High Volume (>10k events/sec)

1. **Increase Kafka partitions**:
```bash
docker exec audit-redpanda rpk topic alter-config audit.scheduler.job_runs \
  --set num.partitions=12
```

2. **Scale Trino workers**:
Edit `docker-compose.yml` to add worker nodes.

3. **Tune Iceberg compaction**:
```sql
-- Run compaction during off-peak hours
CALL iceberg.system.rewrite_data_files(
  schema => 'audit',
  table => 'scheduler_job_runs'
);
```

### For Large Tenants

1. **Per-tenant topic partitioning**:
Use tenant_id as Kafka partition key (already configured in publisher).

2. **Separate Iceberg namespaces**:
```sql
CREATE SCHEMA iceberg.tenant_001_audit;
-- Move high-volume tenant tables to separate schema
```

## Next Steps

1. ✅ Infrastructure running
2. ⬜ Wire up publishers in your scheduler
3. ⬜ Add audit routes to API server
4. ⬜ Deploy frontend audit explorer
5. ⬜ Test end-to-end flow
6. ⬜ Set up alerting on compliance violations
7. ⬜ Schedule materialized view refreshes
8. ⬜ Configure tenant retention policies

See [backend/internal/audit/README.md](../internal/audit/README.md) for full documentation.
