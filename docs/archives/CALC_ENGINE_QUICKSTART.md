# Production Calc Engine - Quick Start & Testing Guide

## Architecture Summary

This calc engine implements the production blueprint for metric computation with:
- **Postgres**: Metric registry (source of truth), transactional control plane, job tracking, anomaly lifecycle
- **Trino/Iceberg**: Near-data compute engine for PoP and z-score anomaly calculations
- **Temporal**: Durable orchestration with retries and observability
- **REST API**: Tenant-scoped metric CRUD and compute triggers
- **RabbitMQ**: Event publishing for downstream systems (Cube.dev, webhooks)

## 1. Prerequisites & Setup

### Environment Configuration

Update your `.env` or `config.yaml` with Trino connection details:

```bash
# Trino connection (your local setup)
TRINO_HOST=192.168.86.55
TRINO_PORT=8090
TRINO_DATABASE=iceberg
TRINO_SCHEMA=demo
TRINO_USER=admin
TRINO_PASSWORD=  # Leave empty if no password

# Postgres (already configured)
DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable

# Temporal (if using local Temporal)
TEMPORAL_ADDRESS=localhost:7233

# Redpanda (optional for event publishing)
KAFKA_BROKERS=localhost:9092
```

### Initialize Postgres Schema

Apply the calc-engine DDL to your Postgres database:

```bash
# From the workspace root
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable < backend/sql/calc-engine.sql
```

Or run migrations if using a migration system:

```bash
cd backend
go run ./cmd/migrate/main.go < sql/calc-engine.sql
```

### Verify Trino Connection

```bash
# Quick connection test
curl -X GET "http://192.168.86.55:8090/v1/info" \
  -H "X-Trino-User: admin"

# Should return:
# {"query.max-memory-per-node":"4GB", ...}
```

## 2. Start the Backend

```bash
cd backend

# Run the backend server (it will initialize Postgres tables and register routes)
go run ./cmd/server/main.go

# Verify routes are registered
curl http://localhost:8080/_routes | jq '.routes | .[] | select(contains("metrics"))'
```

Expected output:
```
"POST /api/metrics"
"GET /api/metrics"
"GET /api/metrics/{metricID}"
"PUT /api/metrics/{metricID}"
"DELETE /api/metrics/{metricID}"
"POST /api/metrics/{metricID}/compute/pop"
"POST /api/metrics/{metricID}/compute/anomaly"
"GET /api/metrics/{metricID}/runs"
"GET /api/metrics/{metricID}/anomalies"
```

## 3. Create a Test Metric

```bash
# Create a metric for PoP calculations
curl -X POST http://localhost:8080/api/metrics \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-001" \
  -H "X-User-ID: user@example.com" \
  -d '{
    "name": "revenue_daily",
    "display_name": "Daily Revenue",
    "domain": "finance",
    "category": "sales",
    "granularity": "day",
    "aggregation_function": "sum",
    "comparison_periods": ["previous_period", "yoy"],
    "sla_freshness_hours": 24,
    "sla_completeness_threshold": 95.0,
    "computation_type": "SQL",
    "computation_logic": "SELECT DATE(order_date) as date, SUM(amount) as value FROM orders GROUP BY DATE(order_date)"
  }'

# Response:
# {
#   "metric_id": "a1b2c3d4-...",
#   "name": "revenue_daily",
#   "display_name": "Daily Revenue",
#   "domain": "finance",
#   "granularity": "day",
#   "aggregation_function": "sum",
#   "golden_path": false,
#   "sla_freshness_hours": 24,
#   "sla_completeness_threshold": 95.0,
#   "created_at": "2025-11-04T...",
#   "updated_at": "2025-11-04T..."
# }

# Save the metric_id for next steps
METRIC_ID="a1b2c3d4-..."
```

## 4. List Metrics

```bash
curl -X GET http://localhost:8080/api/metrics \
  -H "X-Tenant-ID: tenant-001"

# Returns array of metrics for tenant
```

## 5. Trigger PoP Computation

```bash
# Trigger PoP calculation for August 2024
curl -X POST "http://localhost:8080/api/metrics/${METRIC_ID}/compute/pop" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-001" \
  -H "X-User-ID: user@example.com" \
  -d '{
    "period_label": "2024-08"
  }'

# Response:
# {
#   "run_id": "run-uuid-...",
#   "status": "pending"
# }

# Save the run_id
RUN_ID="run-uuid-..."
```

## 6. Check Job Run Status

```bash
# List all runs for the metric
curl -X GET "http://localhost:8080/api/metrics/${METRIC_ID}/runs" \
  -H "X-Tenant-ID: tenant-001"

# Expected output:
# [
#   {
#     "run_id": "run-uuid-...",
#     "metric_id": "a1b2c3d4-...",
#     "calc_type": "pop",
#     "period_label": "2024-08",
#     "status": "pending|running|success|failed",
#     "started_at": "2025-11-04T...",
#     "ended_at": null,
#     "stats": {}
#   }
# ]
```

## 7. Check Job Run in Postgres

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable

-- Check the job run status
SELECT run_id, calc_type, period_label, status, started_at, ended_at, stats 
FROM metric_job_runs 
WHERE metric_id = 'a1b2c3d4-...' 
ORDER BY started_at DESC;

-- Expected after computation completes:
-- run_id            | calc_type | period_label | status  | stats
-- run-uuid-...      | pop       | 2024-08      | success | {"success": true, "record_count": 100, ...}
```

## 8. Verify PoP Results in Trino/Iceberg

```bash
# Query the PoP results in Trino
trino:demo> SELECT * FROM iceberg.metrics_pop 
WHERE tenant_id = 'tenant-001' AND metric_id = 'a1b2c3d4-...'
ORDER BY period_label DESC;

-- Expected columns:
-- tenant_id | metric_id | period_start | period_end | period_label | current_value | previous_value | delta | percent_change | computation_status
```

## 9. Trigger Anomaly Detection

```bash
curl -X POST "http://localhost:8080/api/metrics/${METRIC_ID}/compute/anomaly" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-001" \
  -H "X-User-ID: user@example.com" \
  -d '{
    "period_label": "2024-08"
  }'

# Response:
# {
#   "run_id": "run-uuid-2-...",
#   "status": "pending"
# }
```

## 10. Retrieve Detected Anomalies

```bash
curl -X GET "http://localhost:8080/api/metrics/${METRIC_ID}/anomalies" \
  -H "X-Tenant-ID: tenant-001"

# Expected output:
# [
#   {
#     "id": "anomaly-uuid-...",
#     "anomaly_type": "z_score",
#     "detected_at": "2025-11-04T...",
#     "severity": "high|medium|low|critical",
#     "confidence": 0.95,
#     "actual_value": 15000.50,
#     "expected_value": 10000.00,
#     "expected_range_min": 5000.00,
#     "expected_range_max": 15000.00,
#     "status": "open|resolved|acknowledged",
#     "created_at": "2025-11-04T..."
#   }
# ]
```

## 11. Update a Metric

```bash
curl -X PUT "http://localhost:8080/api/metrics/${METRIC_ID}" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-001" \
  -H "X-User-ID: user@example.com" \
  -d '{
    "name": "revenue_daily",
    "display_name": "Daily Revenue (Updated)",
    "domain": "finance",
    "aggregation_function": "sum",
    "sla_freshness_hours": 12
  }'
```

## 12. Delete a Metric

```bash
curl -X DELETE "http://localhost:8080/api/metrics/${METRIC_ID}" \
  -H "X-Tenant-ID: tenant-001" \
  -H "X-User-ID: user@example.com"

# Returns 204 No Content on success
```

## Troubleshooting

### 404 Not Found on /api/metrics

1. Verify routes are registered:
   ```bash
   curl http://localhost:8080/_routes | grep metrics
   ```

2. Check backend logs for registration errors:
   ```bash
   grep -i "calc\|RegisterCalcEngine" *.log
   ```

3. Ensure Postgres schema is initialized:
   ```bash
   psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "\dt metric_registry"
   ```

### Metric Creation Fails with "Missing required fields"

Ensure all required fields are present in the JSON:
- `name` (unique per tenant)
- `domain`
- `aggregation_function`

Optional fields with defaults:
- `granularity` (default: "day")
- `sla_freshness_hours` (default: 24)
- `sla_completeness_threshold` (default: 95.0)

### Compute Trigger Returns Empty Body

1. Check that the metric_id exists:
   ```bash
   curl -X GET http://localhost:8080/api/metrics/$METRIC_ID \
     -H "X-Tenant-ID: tenant-001"
   ```

2. Verify X-Tenant-ID header matches the tenant that created the metric.

3. Check backend logs for SQL errors.

### Job Run Status Stays "pending"

1. Temporal worker may not be running. Start it:
   ```bash
   # In another terminal
   go run ./cmd/worker/main.go
   ```

2. Check Temporal UI:
   ```
   http://localhost:8080  # or your Temporal UI port
   ```

3. Check for Trino connectivity errors in logs.

## Next Steps

1. **Integrate Temporal Workflows**: Wire the Temporal client to actually trigger workflows on compute requests
2. **Connect Trino**: Test actual SQL execution against your Trino instance
3. **Set Up RabbitMQ**: Implement event publishing for pre-agg refresh and webhooks
4. **Deploy Cube.dev**: Auto-generate schemas and refresh partitions on computation completion
5. **Frontend Integration**: Build React UI with metric registry CRUD, PoP charts, and anomaly triage views

## Files Reference

| File | Purpose |
|------|---------|
| `backend/sql/calc-engine.sql` | Postgres DDL for all tables, functions, indexes |
| `backend/internal/calc-engine/trino/client.go` | Trino JDBC connection wrapper |
| `backend/internal/calc-engine/workflows/workflows.go` | Temporal workflow definitions |
| `backend/internal/calc-engine/activities/activities.go` | Temporal activity implementations (SQL generation, merge execution) |
| `backend/internal/api/calc-engine_handlers.go` | REST API handlers (CRUD, compute triggers) |
| `backend/internal/api/api.go` | Route registration (modified to include calc-engine routes) |

## Monitoring & Observability

### Postgres Queries for Monitoring

```sql
-- Most recent job runs
SELECT run_id, metric_id, calc_type, status, started_at, EXTRACT(EPOCH FROM (ended_at - started_at)) as duration_sec
FROM metric_job_runs
ORDER BY started_at DESC
LIMIT 20;

-- Metrics by domain
SELECT domain, COUNT(*) as count, COUNT(CASE WHEN golden_path THEN 1 END) as golden
FROM metric_registry
GROUP BY domain;

-- Anomaly trends
SELECT date_trunc('hour', detected_at) as hour, severity, COUNT(*) as count
FROM anomaly_events
WHERE detected_at > now() - interval '24 hours'
GROUP BY date_trunc('hour', detected_at), severity
ORDER BY hour DESC;

-- SLA violations (stale metrics)
SELECT mr.metric_id, mr.name, MAX(mjr.ended_at) as last_run, 
       EXTRACT(HOUR FROM now() - MAX(mjr.ended_at)) as hours_since_run,
       CASE WHEN EXTRACT(HOUR FROM now() - MAX(mjr.ended_at)) > mr.sla_freshness_hours THEN 'STALE' ELSE 'OK' END as status
FROM metric_registry mr
LEFT JOIN metric_job_runs mjr ON mr.metric_id = mjr.metric_id AND mjr.status = 'success'
GROUP BY mr.metric_id, mr.name, mr.sla_freshness_hours
HAVING EXTRACT(HOUR FROM now() - MAX(mjr.ended_at)) > mr.sla_freshness_hours
ORDER BY hours_since_run DESC;
```

### Trino Queries for Verification

```sql
-- PoP computation counts
SELECT period_label, COUNT(*) as record_count, COUNT(DISTINCT metric_id) as metric_count
FROM iceberg.metrics_pop
WHERE tenant_id = 'tenant-001'
GROUP BY period_label
ORDER BY period_label DESC;

-- Anomaly distribution by severity
SELECT severity, COUNT(*) as count
FROM iceberg.metrics_anomalies
WHERE tenant_id = 'tenant-001'
  AND detected_at > current_date - interval '7' day
GROUP BY severity;
```

---

## Success Checklist

- [x] Postgres schema initialized with metric_registry, metric_job_runs, anomaly_events
- [x] REST API routes registered (/api/metrics/*)
- [x] Metric CRUD working (Create, Read, Update, Delete)
- [x] PoP compute trigger working (creates job_run record)
- [x] Anomaly compute trigger working
- [x] Job run status retrieval working
- [ ] Temporal workflow actually executing on compute trigger
- [ ] Trino connection working and SQL executing
- [ ] PoP MERGE results visible in Iceberg
- [ ] Anomaly detection results visible in Iceberg
- [ ] RabbitMQ events publishing on completion
- [ ] Cube.dev pre-aggregations refreshing
- [ ] Frontend UI consuming APIs

You're now ready to test the production calc engine locally!
