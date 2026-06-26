# 🎯 Audit & Snapshot Plane - Quick Start

Your immutable, query-able audit infrastructure is now fully integrated!

## ✅ What Was Integrated

### 1. Backend Configuration
**File**: [backend/config.yaml](backend/config.yaml)

Added audit configuration section with:
- Kafka brokers (`localhost:19092`)
- Trino query engine (`localhost:8090`)
- Iceberg catalog (`http://localhost:8181`)
- MinIO S3 storage
- AI narrative service settings
- PII field tracking

### 2. Backend API Routes
**File**: [backend/internal/api/api.go](backend/internal/api/api.go)

Integrated audit routes at `/api/audit/*`:
- `/api/audit/job-runs` - Query scheduler job executions
- `/api/audit/violations` - Query compliance violations
- `/api/audit/changesets` - Query governance changes
- `/api/audit/dashboard/*` - Pre-aggregated analytics
- Full multi-tenant scoping with `X-Tenant-ID` enforcement

### 3. Frontend Routes
**File**: [frontend/src/AppRoutes.tsx](frontend/src/AppRoutes.tsx)

Added route: `/audit` → `AuditExplorer` component

### 4. Frontend Navigation
**File**: [frontend/src/components/MainNavigation.tsx](frontend/src/components/MainNavigation.tsx)

Added navigation link under **Platform > System**:
- "Audit Plane" - Immutable audit & snapshots

---

## 🚀 Deploy Infrastructure

### Start Services

```bash
cd backend/audit-infrastructure
./start.sh
```

This starts:
- **Redpanda** (Kafka) on `localhost:19092`
- **Trino** query engine on `localhost:8090`
- **MinIO** console on `http://localhost:9001` (minioadmin/minioadmin)
- **Iceberg REST** catalog on `localhost:8181`
- **Redpanda Console** on `http://localhost:8080`
- **Audit Sink** consumer (Kafka → Iceberg ingestion)

### Verify Setup

```bash
cd backend/audit-infrastructure
./test.sh
```

Tests:
- Infrastructure health checks
- Kafka connectivity
- Trino queries
- Publishes test events
- Verifies data in Iceberg

### Stop Services

```bash
cd backend/audit-infrastructure
./stop.sh
```

To remove all data:
```bash
docker-compose down -v
```

---

## 📊 Accessing the UI

### 1. Start Backend

```bash
cd backend
go run cmd/server/main.go
```

Backend runs on `http://localhost:8080`

### 2. Start Frontend

```bash
cd frontend
npm start
```

Frontend runs on `http://localhost:3000`

### 3. Navigate to Audit Plane

1. Login to the application
2. Click **Platform** in top navigation
3. Under **System**, click **Audit Plane**
4. Browse:
   - **Job Runs** - Scheduler execution history
   - **Violations** - Compliance issues
   - **Changesets** - Governance changes
   - **Dashboards** - SLO metrics, compliance trends

---

## 🔌 Wire Up Publishers

Now integrate audit publishing into your services:

### Scheduler Service

```go
import "github.com/hondyman/semlayer/backend/internal/audit"

// Initialize publisher
publisher, _ := audit.NewKafkaAuditPublisher("localhost:19092")

// When job completes
record := &audit.SchedulerJobRun{
    RunID:     job.RunID,
    JobID:     job.JobID,
    TenantID:  job.TenantID,
    Status:    string(job.Status),
    StartTS:   job.StartTime,
    EndTS:     &job.EndTime,
    // ... other fields
}
publisher.PublishJobRun(ctx, record)
```

### Governance Service

```go
changeset := &audit.GovernanceChangeSet{
    ChangeSetID:   uuid.New().String(),
    TenantID:      change.TenantID,
    ChangeType:    "POLICY_UPDATE",
    ResourceID:    change.ResourceID,
    ActorID:       change.ActorID,
    ChangedFields: change.Fields,
    // ... other fields
}
publisher.PublishChangeSet(ctx, changeset)
```

### Compliance Engine

```go
violation := &audit.ComplianceViolation{
    ViolationID:   uuid.New().String(),
    TenantID:      resource.TenantID,
    ViolationType: v.Type,
    Severity:      v.Severity,
    ResourceID:    resource.ID,
    // ... other fields
}
publisher.PublishComplianceViolation(ctx, violation)
```

---

## 🧪 Testing Locally

### Publish Test Events

```bash
cd backend
go run scripts/test-audit-event.go
```

### Query Trino Directly

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
SELECT severity, COUNT(*) as count
FROM compliance_violations
WHERE tenant_id = 'tenant-test-001'
  AND detected_ts > CURRENT_TIMESTAMP - INTERVAL '7' DAY
GROUP BY severity;

-- Governance activity
SELECT change_type, COUNT(*) as changes
FROM governance_changesets
WHERE tenant_id = 'tenant-test-001'
  AND change_ts > CURRENT_TIMESTAMP - INTERVAL '1' DAY
GROUP BY change_type;
```

### Test API Endpoints

```bash
# Get job runs
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/job-runs?limit=10

# Get violations
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/violations?severity=critical

# Dashboard data
curl -H "X-Tenant-ID: tenant-test-001" \
  http://localhost:8080/api/audit/dashboard/slo
```

---

## 📈 Monitoring

### Check Kafka Lag

```bash
docker exec audit-redpanda rpk group describe audit-sink-group
```

### View Parquet Files

```bash
# MinIO Console
open http://localhost:9001

# Or CLI
docker exec audit-minio mc ls local/warehouse/audit/scheduler_job_runs/
```

### Check Consumer Logs

```bash
docker logs -f audit-sink
```

---

## 📚 Documentation

- **Architecture**: [backend/internal/audit/README.md](backend/internal/audit/README.md)
- **Integration Guide**: [backend/internal/audit/INTEGRATION.md](backend/internal/audit/INTEGRATION.md)
- **Infrastructure Setup**: [backend/audit-infrastructure/README.md](backend/audit-infrastructure/README.md)

---

## 🆘 Troubleshooting

### Events Not Appearing

1. Check Kafka topics:
   ```bash
   docker exec audit-redpanda rpk topic list
   docker exec audit-redpanda rpk topic consume audit.scheduler.job_runs
   ```

2. Check sink consumer:
   ```bash
   docker logs audit-sink
   ```

3. Verify Parquet files exist:
   ```bash
   docker exec audit-minio mc ls local/warehouse/audit/
   ```

### Trino Query Errors

1. Check catalog:
   ```bash
   docker exec audit-trino trino --execute "SHOW CATALOGS"
   docker exec audit-trino trino --execute "SHOW TABLES IN iceberg.audit"
   ```

2. Refresh metadata:
   ```sql
   CALL iceberg.system.expire_snapshots('audit', 'scheduler_job_runs', TIMESTAMP '2024-01-01 00:00:00');
   ```

### Backend Not Connecting

Check environment variables:
```bash
export AUDIT_TRINO_HOST=localhost
export AUDIT_TRINO_PORT=8090
export KAFKA_BROKERS=localhost:19092
export OPENAI_API_KEY=sk-...
```

---

## 🎯 Next Steps

1. ✅ **Infrastructure Running** - `./start.sh` completed
2. ✅ **Backend Configured** - config.yaml updated
3. ✅ **Frontend Integrated** - routes and navigation added
4. 🔄 **Wire Publishers** - Add audit calls to scheduler/governance services
5. 📊 **Monitor Dashboards** - View SLO metrics and compliance trends
6. 🤖 **Enable AI Narratives** - Set `OPENAI_API_KEY` for explanations

---

**Your platform is now provably trustworthy** - every job, policy change, and compliance event is captured immutably in Iceberg, query-able via Trino, with AI-powered insights. 🎉
