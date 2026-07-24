# Phase 3.15 — Dashboard UI, Real-time Analytics, Scheduled Reports, Predictive Health

Complete implementation guide for turning the Phase 3.13-3.14 analytics layer into a production-grade real-time platform with ML-powered predictions.

## Quick Start

### 1. Start Temporal Server
```bash
# Using Docker Compose (add to your docker-compose.yml)
docker-compose up -d temporal

# Verify server is running
curl http://localhost:7233/health
```

### 2. Build and Start Temporal Worker
```bash
cd backend
go build -o temporal-worker ./cmd/temporal-worker
./temporal-worker \
  --temporal-address localhost:7233 \
  --namespace default \
  --task-queue analytics-worker
```

### 3. Register Workflows with Cron Schedules
```bash
bash scripts/temporal/start_workflows.sh
```

This registers three workflows:
- **HourlyRollupWorkflow**: Runs at :05 every hour (e.g., 10:05, 11:05, ...)  
- **DailySLAWorkflow**: Runs daily at 06:00 UTC  
- **MLTrainingWorkflow**: Runs weekly on Sunday at 00:00 UTC

### 4. Create Materialized Analytics Tables in Trino
```bash
trino -f configs/trino/materialized_rollups.sql
```

Creates 6 Iceberg tables:
- `hourly_chain_rollup` — Aggregated metrics at hourly granularity
- `daily_chain_sla` — Daily SLA compliance metrics  
- `chain_health_report` — Computed health scores and recommendations
- `chain_predictions` — ML batch scoring predictions
- `chain_features` — Features for ML training
- `model_registry` — Trained model metadata and performance

Also creates 5 materialized views and stored procedures for refresh orchestration.

### 5. Monitor Active Workflows
```bash
# List active workflows
temporal workflow list --namespace default

# View specific workflow details
temporal workflow describe --namespace default --workflow-id <WORKFLOW_ID>

# View workflow execution history
temporal workflow show --namespace default --workflow-id <WORKFLOW_ID>

# Access Temporal Web UI (if available)
open http://localhost:8080
```

---

## Architecture Overview

```
Source Systems (Postgres, Events)
       │
       └──> CDC (Debezium) ──> Redpanda (streaming)
                                   │
                                   └──> Ingestion Layer
                                        (commit service → S3 → manifest)
                                             │
                                             └────> Iceberg Metadata Catalog
                                                         │
                    ┌────────────────────────────────────┼────────────────────────────────────┐
                    │                                    │                                    │
            Trino (Query Engine)            Materialized Rollups        Feature Store (ML)
                    │                          (hourly, daily)              (chain_features)
                    │                               │                             │
                    └───────────────────────────────┴─────────────────────────────┘
                                                    │
                                    ┌───────────────┼───────────────┐
                                    │               │               │
                            Dashboard API     WebSocket Hub    ML Training
                            (Go handlers)    (real-time events)  (Temporal)
                                    │               │               │
                                    └───────────────┼───────────────┘
                                                    │
                                         React Frontend Dashboard
                                    (charts, trends, predictions)
```

### Data Flow
1. **Real-time events** → Ingested to Iceberg via CDC/Redpanda
2. **Hourly job** (Temporal) → Computes `hourly_chain_rollup` via Trino
3. **Daily job** (Temporal) → Computes `daily_chain_sla` and health reports
4. **Weekly job** (Temporal) → Feature extraction → ML training → batch scoring → predictions published
5. **WebSocket hub** → Publishes events to connected dashboards for real-time updates
6. **Dashboard API** → fetches trends, health, predictions from Trino and Iceberg

---

## Component Details

### Temporal Workflows

#### 1. HourlyRollupWorkflow
- **Schedule**: Every hour at :05 (e.g., 10:05, 11:05)
- **Inputs**: run_id, regions list
- **Steps**:
  1. Publish "hourly_rollup_started" event
  2. Fan-out child workflows per region (us-east-1, eu-west-1, apac-1)
  3. Each region executes:
     - Trino INSERT INTO hourly_chain_rollup (aggregates last hour)
     - Validation query (count new records)
     - Publish "region_rollup_completed" event
  4. Publish "hourly_rollup_completed" event
- **Retry**: 8 attempts with exponential backoff (5s → 10m max)
- **Timeout**: 30 minutes per activity

#### 2. DailySLAWorkflow
- **Schedule**: Daily at 06:00 UTC
- **Inputs**: run_id, date (YYYY-MM-DD)
- **Steps**:
  1. Trino INSERT INTO daily_chain_sla (aggregates day's hourly rollup)
  2. Trino INSERT INTO chain_health_report (compute health scores)
  3. Publish "daily_sla_refreshed" event
- **Retry**: 5 attempts with exponential backoff
- **Timeout**: 2 hours per activity

#### 3. MLTrainingWorkflow
- **Schedule**: Weekly on Sunday at 00:00 UTC
- **Inputs**: run_id, model_name, training_date
- **Steps**:
  1. Run Python: `extract_features.py` → reads Iceberg, writes chain_features
  2. Run Python: `train_model.py` → trains XGBoost model on features
  3. Run Python: `evaluate_and_register.py` → evaluates, stores model artifact, registers in model_registry
  4. Publish "ml_training_completed" event
- **Retry**: 3 attempts with exponential backoff
- **Timeout**: 4 hours per activity

### Activities

#### RunTrinoQueryActivity
- **Purpose**: Execute Trino SQL queries with pagination support
- **Inputs**: runID (for tracing), region, SQL query
- **Outputs**: JSON with query_id, status, row_count
- **Error Handling**: Non-200 responses → error return; pagination failures propagated
- **Idempotency**: runID passed as X-Trino-User header for tracing

#### RunSparkJobActivity
- **Purpose**: Submit Spark job and poll for completion
- **Inputs**: runID, Spark submit config (jar, main_class, executor settings)
- **Outputs**: Submission ID and final status
- **Polling**: Max 600 attempts (10 minutes) with 1-second intervals
- **Error Handling**: Failed/error driver state → error return

#### RunPythonScriptActivity
- **Purpose**: Execute Python scripts for ML and data processing
- **Inputs**: runID, script path, args
- **Outputs**: JSON result with status and metadata
- **Implementation**: Placeholder for subprocess or external service call

#### PublishEventActivity
- **Purpose**: Publish events to WebSocket hub for real-time dashboard updates
- **Inputs**: runID, region, event type
- **Outputs**: None (non-fatal if hub unavailable)
- **HTTP**: POST to http://localhost:8081/events/publish

### WebSocket Hub

**Location**: `internal/realtime/hub.go`

**Features**:
- Multi-tenant support (tenant_id scoping)
- Region-based subscriptions
- Automatic client tracking and cleanups
- Periodic ping/pong for connection health
- 30-second timeout on idle connections

**HTTP Endpoints**:
- `GET /ws?tenant_id=<UUID>` — Upgrade WebSocket
  - Subscribe: `{"action":"subscribe","region":"us-east-1"}`
  - Unsubscribe: `{"action":"unsubscribe","region":"us-east-1"}`
- `POST /events/publish` — Publish event (for activities)
  - Payload: `{"type":"chain_update","tenant_id":"...","region":"us-east-1","data":{...}}`
- `GET /stats` — Hub statistics

**Event Types**:
- `hourly_rollup_started` / `hourly_rollup_completed`
- `region_rollup_completed`
- `daily_sla_refreshed`
- `ml_training_completed`
- `chain_update` (custom events from external systems)

---

## Materialized Tables in Iceberg

### hourly_chain_rollup
```
tenant_id, chain_id, region, window_hour,
success_count, failure_count, avg_latency_ms, p95_latency_ms, p99_latency_ms,
incident_count, computed_at
```
- Partition: region, year(window_hour), month, day, hour
- Updated hourly by HourlyRollupWorkflow
- TTL: 90 days

### daily_chain_sla
```
tenant_id, chain_id, region, day,
success_rate_pct, avg_latency_ms, p95_latency_ms, p99_latency_ms,
incident_count, sla_met, computed_at
```
- Partition: region, year(day), month, day
- Updated daily by DailySLAWorkflow
- TTL: 3 years

### chain_health_report
```
id, chain_id, tenant_id, region,
overall_health (0-100), last_execution_status,
consecutive_failures, is_healthy,
recommended_action ("investigate" | "retry" | "disable" | "none"),
action_executed, reported_at, created_at
```
- Partition: region, year(reported_at), month, day
- Updated daily by DailySLAWorkflow
- TTL: 1 year

### chain_predictions (ML)
```
id, chain_id, tenant_id, region,
prediction_ts, failure_prob (0.0-1.0),
recommended_action, model_version,
top_features (JSON), created_at
```
- Partition: region, year(prediction_ts), month, day
- Updated weekly by MLTrainingWorkflow (batch scoring)
- TTL: 1 year

---

## Trino Materialized Views

### sla_trend_30d
Last 30 days of SLA metrics with lag analysis:
- success_rate_change_pct: Daily change in success rate
- sla_status: "excellent" | "good" | "acceptable" | "critical"

### health_distribution
Distribution of chain health by region:
- health_category: "Excellent" | "Good" | "Fair" | "Poor"
- healthy_pct: % of chains in category

### actions_pending
Actionable insights (unexecuted recommendations):
- Shows chains needing investigation, retry, or disable

### high_risk_predictions
High-risk predictions from ML model:
- failure_prob >= 0.5
- risk_level: "Critical" | "High" | "Medium" | "Low"

---

## Configuration

**Temporal Worker** (`configs/temporal/worker-config.yaml`):
```yaml
temporal:
  server_address: localhost:7233
  namespace: default
  task_queue: analytics-worker
  max_concurrent_activity_executions: 100

trino:
  server_url: http://trino:8080
  catalog: iceberg
  schema: ops
  request_timeout_seconds: 300

ml:
  feature_store_table: iceberg.ops.chain_features
  model_registry_table: iceberg.ops.model_registry
```

**Environment Variables**:
- `TEMPORAL_SERVER_ADDRESS` — default: localhost:7233
- `TEMPORAL_NAMESPACE` — default: default
- `TEMPORAL_TASK_QUEUE` — default: analytics-worker
- `TRINO_URL` — default: http://trino:8080
- `SPARK_SUBMIT_URL` — default: http://spark-submit:6066
- `S3_BUCKET` — S3 bucket for artifacts
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` — AWS credentials

---

## Testing

### Unit Tests

**Workflows** (`internal/temporal/workflows/workflows_test.go`):
```bash
go test ./internal/temporal/workflows -v
```

Tests cover:
- ✅ HourlyRollupWorkflow — successful execution and partial failures
- ✅ RegionHourlyRollupWorkflow — retry behavior on Trino failures
- ✅ DailySLAWorkflow — date validation and normal execution
- ✅ MLTrainingWorkflow — full pipeline and failure propagation

**Activities** (`internal/temporal/activities/activities_test.go`):
```bash
go test ./internal/temporal/activities -v
```

Tests cover:
- ✅ RunTrinoQueryActivity — successful queries and error handling
- ✅ RunSparkJobActivity — job submission and missing config validation
- ✅ RunPythonScriptActivity — script execution
- ✅ PublishEventActivity — event publishing

### Integration Tests

**End-to-end workflow test** (requires services running):
```bash
# Start Temporal, Trino, S3, worker
docker-compose up -d temporal trino minio

# Build worker and start
go build -o temporal-worker ./cmd/temporal-worker
./temporal-worker &

# Manually trigger a workflow
temporal workflow start \
  --workflow-type HourlyRollupWorkflow \
  --task-queue analytics-worker \
  --input '{"run_id":"test-001","regions":["us-east-1"]}'

# Monitor execution
temporal workflow show --workflow-id <ID>

# Check results in Trino
trino -e "SELECT COUNT(*) FROM iceberg.ops.hourly_chain_rollup"
```

---

## Next Steps (Phase 3.16+)

1. **React Dashboard UI**
   - Pages: Home, Tenant View, Chain Detail, Live Feed, Reports
   - Components: Charts (Recharts), metric cards, WebSocket client
   - Real-time updates via WebSocket

2. **ML Model Explainability**
   - SHAP values for top contributing features
   - Feature importance visualization
   - Model drift monitoring

3. **Anomaly Detection**
   - Unsupervised methods for outlier detection
   - Alert on unexpected SLA deviations
   - Automated remediation triggers

4. **Distributed Tracing**
   - Integrate Jaeger/DataDog
   - Trace Temporal workflows, Trino queries, and chain executions
   - Correlation with business metrics

5. **Advanced Scheduling**
   - Backfill workflows for historical training
   - Conditional workflow triggers (e.g., on incident count threshold)
   - Per-tenant customization of schedules

---

## Troubleshooting

### Workflow Not Starting
- Check Temporal server: `curl http://localhost:7233/health`
- Verify worker is running and listening to task queue
- Check task queue name matches in workflow start command

### Trino Queries Failing
- Verify iceberg catalog and ops schema exist
- Check user permissions in Trino
- Test query directly: `trino -e "SELECT 1"`

### Events Not Publishing to WebSocket
- Ensure hub is running on localhost:8081
- Check firewall rules for port 8081
- Review logs for "event publish failed"

### ML Training Pipeline Hanging
- Monitor Spark logs for job failures
- Check S3 connectivity and bucket access
- Verify feature extraction script exists

---

## Monitoring and Observability

**Prometheus Metrics** (exposed on port 8090):
- `temporal_analytics_workflow_duration_seconds` — histogram
- `temporal_analytics_activity_duration_seconds` — histogram
- `temporal_analytics_trino_query_duration_seconds` — histogram
- `temporal_analytics_spark_job_duration_seconds` — histogram
- `temporal_analytics_events_published_total` — counter

**Grafana Dashboards**:
- Workflow Success Rates
- Activity Latencies
- Queue Depth and Worker Utilization
- Events Published Per Second
- Model Training History

**Structured Logs**:
All Temporal activities and workflows emit JSON logs with:
- `run_id`, `workflow_id`, `activity_name`
- `region`, `tenant_id`, `chain_id`
- `status` (success|failure|timeout)
- `duration_ms`, `error` (if failed)

---

## Performance Tuning

### For Faster Rollups
- Increase `max_concurrent_activity_executions` in worker config
- Fan out more regions in child workflows
- Pre-compute aggregate materialized views in Minio/S3

### For Faster Model Training
- Use Spark with more executors and memory
- Stream features from Iceberg with columnar pushdown
- Cache feature tables in memory (if feasible)

### For Real-time Dashboard
- Increase WebSocket hub buffer sizes
- Use Trino caching for frequently queried views
- Compress JSON payloads over WebSocket

---

## Files and Structure

```
backend/
├── cmd/
│   └── temporal-worker/
│       └── main.go                    # Worker entry point
├── internal/
│   ├── temporal/
│   │   ├── worker.go                  # Worker bootstrap
│   │   ├── workflows/
│   │   │   ├── hourly_rollup.go       # Hourly workflow
│   │   │   ├── daily_sla.go           # Daily & ML workflows
│   │   │   └── workflows_test.go      # Workflow tests
│   │   └── activities/
│   │       ├── activities.go          # All activity implementations
│   │       └── activities_test.go     # Activity tests
│   └── realtime/
│       └── hub.go                     # WebSocket hub
├── configs/
│   ├── temporal/
│   │   └── worker-config.yaml         # Worker configuration
│   └── trino/
│       └── materialized_rollups.sql   # Iceberg table DDL
└── scripts/
    └── temporal/
        └── start_workflows.sh         # Workflow registration script
```

---

For more details, see the inline code documentation and the comprehensive Copilot instructions in `.github/copilot-instructions.md`.
