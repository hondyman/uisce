# Temporal + RabbitMQ Integration Guide

Your metric computation engine now routes to both **Temporal workflows** (for durable orchestration) and **RabbitMQ** (for event publishing). Trino, Iceberg, and Spark integration are deferred.

## Architecture

```
Frontend/API
    ↓
POST /api/metrics/{id}/compute/pop
    ↓
CalcEngineHandler.triggerCompute()
    ├─ Create job_run in Postgres (status: pending)
    ├─ Execute Temporal workflow (asynchronously)
    └─ Return 202 Accepted immediately
    
Temporal Worker (background)
    ├─ MetricComputeWorkflow
    │   ├─ UpsertRunStatus("running")
    │   ├─ Branch on calc_type
    │   ├─ Execute Activity (PoP or Anomaly)
    │   ├─ UpsertRunStatus("success")
    │   ├─ PublishCompletionEvent (→ RabbitMQ)
    │   └─ RefreshCubePartitions (placeholder)
    └─ Each activity logs SQL for manual execution

RabbitMQ (event queue)
    └─ Listens on queue: metrics.computations
    └─ Events consumed by downstream systems
```

## Setup

### 1. Start Temporal Server (if not already running)

```bash
# Using temporal-cli
temporal server start-dev

# Or using Docker
docker run -d --name temporal \
  -p 7233:7233 \
  -p 8233:8233 \
  temporalio/auto-setup:latest
```

### 2. Start RabbitMQ (if not already running)

```bash
# Using Docker
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:4.0-management

# Access management UI at http://localhost:15672
# Default user: guest / password: guest
```

### 3. Set Environment Variables

```bash
# In your .env file or shell:
export TEMPORAL_HOSTPORT=localhost:7233
export RABBIT_URL=amqp://guest:guest@localhost:5672/
```

### 4. Initialize Database

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha < backend/sql/calc-engine.sql
```

### 5. Start Backend

```bash
cd backend
go run ./cmd/server/main.go
```

The backend will automatically:
- Initialize Temporal client and worker
- Connect to RabbitMQ and declare queue
- Register workflows and activities
- Start listening for compute requests

## API Usage

### Create a Metric

```bash
curl -X POST -H "X-Tenant-ID: tenant1" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/metrics \
  -d '{
    "name": "Revenue",
    "domain": "finance",
    "aggregation_function": "sum",
    "computation_logic": "SELECT SUM(amount) FROM transactions"
  }'
```

Response:
```json
{
  "metric_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Revenue",
  "domain": "finance",
  ...
}
```

### Trigger Computation

```bash
curl -X POST -H "X-Tenant-ID: tenant1" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/metrics/{metric_id}/compute/pop \
  -d '{"period_label": "2024-11"}'
```

Response (202 Accepted - immediately returned):
```json
{
  "run_id": "550e8400-e29b-41d4-a716-446655440001",
  "status": "pending"
}
```

### Check Status

```bash
curl -H "X-Tenant-ID: tenant1" \
  http://localhost:8080/api/metrics/{metric_id}/runs
```

Response:
```json
[
  {
    "run_id": "550e8400-e29b-41d4-a716-446655440001",
    "metric_id": "550e8400-e29b-41d4-a716-446655440000",
    "calc_type": "pop",
    "period_label": "2024-11",
    "status": "success",  // or "pending", "running", "failed"
    "started_at": "2025-11-04T10:00:00Z",
    "ended_at": "2025-11-04T10:00:05Z"
  }
]
```

## How It Works

### 1. REST Request → Postgres (Synchronous)
- User calls `POST /api/metrics/{id}/compute/pop`
- Handler immediately creates `metric_job_runs` record with status: `pending`
- Returns `202 Accepted` with run ID

### 2. Temporal Workflow Execution (Asynchronous)
- Handler also triggers Temporal workflow with the run ID
- Workflow executes in background on worker
- Each activity updates Postgres and publishes events

### 3. Activities Generate SQL (Deferred Execution)
- **ComputeAndMergePoP**: Generates MERGE SQL for PoP calculation
  - Currently logs SQL (not executing against Trino yet)
  - When Trino is wired: will execute actual MERGE statement
  
- **ComputeAndMergeAnomalies**: Generates MERGE SQL for z-score detection
  - Currently logs SQL (not executing against Trino yet)
  - When Trino is wired: will execute actual MERGE statement

### 4. RabbitMQ Event Publishing (On Completion)
- When workflow succeeds, **PublishCompletionEvent** fires
- Event is published to `metrics.computations` queue
- Payload includes: event_id, tenant_id, metric_id, calc_type, run_id, completed_at

### 5. Status Tracking in Postgres
- All job runs visible in `metric_job_runs` table
- Status transitions: pending → running → success/failed
- Timestamps track execution duration

## Observability

### Temporal Web UI
Open http://localhost:8233 to see:
- All workflow executions
- Activity details and logs
- Error traces and retry history
- Task queue statistics

### RabbitMQ Management UI
Open http://localhost:15672 to see:
- Queue statistics
- Published messages
- Consumer connections
- Dead letter exchanges

### Postgres Queries

**Check all job runs:**
```sql
SELECT run_id, metric_id, status, started_at, ended_at, duration_ms 
FROM metric_job_runs 
WHERE tenant_id = 'tenant1' 
ORDER BY started_at DESC;
```

**Check pending jobs:**
```sql
SELECT metric_id, calc_type, period_label, status, created_at 
FROM metric_job_runs 
WHERE status IN ('pending', 'running');
```

**Check failures:**
```sql
SELECT run_id, metric_id, status, error_message, error_details 
FROM metric_job_runs 
WHERE status = 'failed' 
ORDER BY started_at DESC;
```

## Example: End-to-End Flow

```bash
# 1. Create metric
METRIC_ID=$(curl -s -X POST -H "X-Tenant-ID: test" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/metrics \
  -d '{"name":"Revenue","domain":"finance","aggregation_function":"sum"}' \
  | jq -r .metric_id)

echo "Created metric: $METRIC_ID"

# 2. Trigger computation
RUN_ID=$(curl -s -X POST -H "X-Tenant-ID: test" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/metrics/$METRIC_ID/compute/pop \
  -d '{"period_label":"2024-11"}' \
  | jq -r .run_id)

echo "Started computation: $RUN_ID"

# 3. Poll status (watch transitions from pending → running → success)
for i in {1..10}; do
  STATUS=$(curl -s -H "X-Tenant-ID: test" \
    http://localhost:8080/api/metrics/$METRIC_ID/runs \
    | jq -r '.[0].status')
  echo "[$i] Status: $STATUS"
  [ "$STATUS" = "success" ] && break
  sleep 1
done

# 4. View final results
curl -s -H "X-Tenant-ID: test" \
  http://localhost:8080/api/metrics/$METRIC_ID/runs | jq
```

## Deferred Components (Not Yet Wired)

### Trino SQL Execution
Currently in activities:
```go
// TODO: Activate when Trino is ready
// err := globalConfig.TrinoClient.ExecuteMerge(ctx, mergeSQL)
```

Replace logging with actual Trino execution when ready.

### Iceberg Tables
Expected tables (to be created):
- `metrics_pop` - Period-over-period results
- `metrics_anomalies` - Anomaly detection results
- `metrics_atomic` - Raw metric values

### Spark Transformations
Can be triggered via Temporal activity once Spark cluster is available.

## Testing

### Test Metric Creation
```bash
go test ./backend/internal/api -run TestCalcEngine -v
```

### Test Workflow Execution
```bash
# Once Temporal is running, workflows will auto-execute
# Monitor at: http://localhost:8233
```

### Test RabbitMQ Events
```bash
# Monitor queue depth
rabbitmqctl list_queues name messages consumers

# Consume test message
python3 -c "
import pika
conn = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
ch = conn.channel()
ch.queue_declare(queue='metrics.computations', durable=True)
def callback(ch, method, properties, body):
    print(f'Event: {body.decode()}')
ch.basic_consume(queue='metrics.computations', on_message_callback=callback, auto_ack=True)
ch.start_consuming()
"
```

## Configuration Reference

| Env Var | Default | Purpose |
|---------|---------|---------|
| `TEMPORAL_HOSTPORT` | `localhost:7233` | Temporal server address |
| `RABBIT_URL` | *(not set - RabbitMQ disabled)* | RabbitMQ connection string |

## Next Steps

1. **Verify integration is working:**
   - Backend starts without errors
   - Temporal worker initializes
   - RabbitMQ connects (or gracefully skips if not running)

2. **Wire Trino execution:**
   - Uncomment Trino calls in `activities.go`
   - Test actual MERGE statements against Iceberg
   - Verify results in Postgres

3. **Add monitoring:**
   - Prometheus metrics for workflow duration
   - Alerts for failed computations
   - Event publishing SLAs

4. **Scale up:**
   - Run multiple workers for parallel computation
   - Add persistent Temporal history storage
   - Scale RabbitMQ to multi-node cluster
