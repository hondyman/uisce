# Production Calc Engine - Integration Guide

## Architecture Components

The calc engine integrates three key layers:

### 1. Frontend Layer (React/TypeScript)
Tenant-scoped registry, metric CRUD, PoP/anomaly charts, compute triggers, runs/triage views.

### 2. API Gateway + Backend (Go + Chi Router)
- Tenant isolation with `X-Tenant-ID` header
- RBAC enforcement
- REST endpoints for metric CRUD and compute triggers
- Routes registered in `backend/internal/api/api.go`

### 3. Orchestration Layer (Temporal)
- `MetricComputeWorkflow`: Parent orchestrator branching on calc_type
- Activities: SQL generation, Trino execution, status tracking, event publishing

### 4. Compute Layer (Trino/Iceberg)
- Pushes calculation near data
- Executes MERGE INTO for PoP (period-over-period) with delta/percent_change
- Executes MERGE INTO for z-score anomalies with rolling 90-day windows

### 5. Storage Layers
- **Postgres**: `metric_registry` (source of truth), `metric_job_runs` (transactional), `anomaly_events` (lifecycle mgmt)
- **Iceberg**: `metrics_atomic` (daily facts), `metrics_pop` (monthly aggregations), `metrics_anomalies` (z-score detections)

## Integration Steps

### Step 1: Verify Postgres Schema

```bash
psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c "\dt metric_registry"

# Should show:
#          List of relations
#  Schema |      Name       | Type  | Owner
# --------+-----------------+-------+----------
#  public | metric_registry | table | postgres
```

### Step 2: Verify Backend Routes

```bash
curl http://localhost:8080/_routes | jq '.routes[]' | grep metrics

# Should show:
# "POST /api/metrics"
# "GET /api/metrics"
# "GET /api/metrics/{metricID}"
# etc.
```

### Step 3: Create Test Metric

```bash
METRIC_ID=$(curl -s -X POST http://localhost:8080/api/metrics \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-ID: test-user" \
  -d '{
    "name": "test_metric",
    "domain": "analytics",
    "aggregation_function": "sum"
  }' | jq -r '.metric_id')

echo "Created metric: $METRIC_ID"
```

### Step 4: Trigger PoP Computation

```bash
curl -X POST http://localhost:8080/api/metrics/$METRIC_ID/compute/pop \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: test-tenant" \
  -H "X-User-ID: test-user" \
  -d '{"period_label": "2024-08"}'

# Returns: {"run_id": "uuid", "status": "pending"}
```

### Step 5: Monitor Computation

The system records the job run in `metric_job_runs` immediately, but actual Temporal workflow execution requires wiring up the Temporal client. For now:

```bash
# Check job run status in Postgres
psql postgres://postgres:postgres@host.docker.internal:5432/alpha -c \
  "SELECT run_id, status, started_at FROM metric_job_runs ORDER BY started_at DESC LIMIT 5"
```

## Connecting Temporal for Background Computation

To enable actual background workflow execution:

### 1. Initialize Temporal Client in Backend

Update `backend/cmd/server/main.go` to initialize Temporal client:

```go
import (
  "go.temporal.io/sdk/client"
)

// In main()
temporalClient, err := client.Dial(client.Options{
  HostPort: os.Getenv("TEMPORAL_ADDRESS"),
})
if err != nil {
  logging.GetLogger().Sugar().Fatalf("Failed to connect to Temporal: %v", err)
}
defer temporalClient.Close()

// Pass to handlers
srv.TemporalClient = temporalClient
```

### 2. Update Handler to Trigger Workflow

In `backend/internal/api/calc-engine_handlers.go`, modify `triggerCompute()`:

```go
// TODO: Replace this with actual Temporal workflow trigger:
// Instead of just creating the record, also start the workflow:

workflowOpts := client.StartWorkflowOptions{
  ID:        req.RunID,
  TaskQueue: "metrics-compute",
}

we, err := h.temporalClient.ExecuteWorkflow(ctx, workflowOpts, 
  workflows.MetricComputeWorkflow, req)
if err != nil {
  return fmt.Errorf("failed to start workflow: %w", err)
}
```

### 3. Register Workflow in Worker

Create `backend/cmd/worker/main.go` to run the Temporal worker:

```go
package main

import (
  "go.temporal.io/sdk/client"
  "go.temporal.io/sdk/worker"
  "github.com/hondyman/semlayer/backend/internal/calc-engine/workflows"
  "github.com/hondyman/semlayer/backend/internal/calc-engine/activities"
)

func main() {
  c, err := client.Dial(client.Options{HostPort: "localhost:7233"})
  if err != nil {
    panic(err)
  }
  defer c.Close()

  w := worker.New(c, "metrics-compute", worker.Options{})

  // Register workflows
  w.RegisterWorkflow(workflows.MetricComputeWorkflow)

  // Register activities
  w.RegisterActivity(activities.UpsertRunStatus)
  w.RegisterActivity(activities.ComputeAndMergePoP)
  w.RegisterActivity(activities.ComputeAndMergeAnomalies)
  w.RegisterActivity(activities.PublishCompletionEvent)
  w.RegisterActivity(activities.RefreshCubePartitions)

  if err := w.Start(); err != nil {
    panic(err)
  }
}
```

## Connecting Trino for SQL Execution

To enable actual Trino execution:

### 1. Test Trino Connection

```bash
# Install trino-cli
curl -L https://repo1.maven.org/maven2/io/trino/trino-cli/latest/trino-cli-latest-executable.jar \
  -o /usr/local/bin/trino && chmod +x /usr/local/bin/trino

# Connect and test
trino --server http://192.168.86.55:8090 --catalog iceberg --schema demo \
  -e "SELECT 'OK'"
```

### 2. Update Activity Config

In `backend/internal/calc-engine/activities/activities.go`, initialize Trino client:

```go
import "github.com/hondyman/semlayer/backend/internal/calc-engine/trino"

// In activity initialization:
trinoClient, err := trino.NewClient(&trino.ClientConfig{
  Host:     os.Getenv("TRINO_HOST"),
  Port:     8090,
  Database: "iceberg",
  Schema:   "demo",
  User:     "admin",
})
if err != nil {
  panic(err)
}

globalConfig = &ActivityConfig{
  DB:          db,
  TrinoClient: trinoClient,
}
```

### 3. Execute Actual Trino Merges

Replace the logging-only implementation in `ComputeAndMergePoP()`:

```go
// Current (logging only):
fmt.Printf("Executing PoP MERGE SQL:\n%s\n", mergeSQL)

// Should be replaced with:
stats, err := globalConfig.TrinoClient.ExecuteMerge(ctx, mergeSQL)
if err != nil {
  return fmt.Errorf("trino merge failed: %w", err)
}
```

## Connecting RabbitMQ for Event Publishing

To enable event publishing:

### 1. Update Activity to Publish Events

In `backend/internal/calc-engine/activities/activities.go`:

```go
import (
  "github.com/streadway/amqp"
)

// In PublishCompletionEvent():
conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
if err != nil {
  return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
}
defer conn.Close()

ch, err := conn.Channel()
if err != nil {
  return fmt.Errorf("failed to create channel: %w", err)
}
defer ch.Close()

event := map[string]interface{}{
  "event_id":     uuid.NewString(),
  "tenant_id":    req.TenantID,
  "metric_id":    req.MetricID,
  "calc_type":    req.CalcType,
  "completed_at": time.Now(),
}

payload, _ := json.Marshal(event)

return ch.PublishWithContext(ctx,
  "metrics.events",
  fmt.Sprintf("metrics.computed.%s", req.CalcType),
  false, false,
  amqp.Publishing{
    ContentType:  "application/json",
    DeliveryMode: amqp.Persistent,
    Body:         payload,
  },
)
```

## Connecting Cube.dev for Pre-aggregation Refresh

To enable automatic Cube refresh on computation completion:

### 1. Update Activity to Call Cube API

In `backend/internal/calc-engine/activities/activities.go`:

```go
// In RefreshCubePartitions():
client := &http.Client{Timeout: 30 * time.Second}

cubeURL := os.Getenv("CUBE_API_URL")
cubeToken := os.Getenv("CUBE_API_TOKEN")

refreshReq := map[string]interface{}{
  "cube":      "MetricsPopMonthly",
  "partition": []string{fmt.Sprintf("%s:%s", req.TenantID, req.PeriodLabel)},
}

payload, _ := json.Marshal(refreshReq)

req, _ := http.NewRequestWithContext(ctx, "POST",
  fmt.Sprintf("%s/v1/pre-aggregations/partitions", cubeURL),
  bytes.NewReader(payload))

req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cubeToken))
req.Header.Set("Content-Type", "application/json")

resp, err := client.Do(req)
if err != nil {
  return fmt.Errorf("cube refresh failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
  return fmt.Errorf("cube returned %d", resp.StatusCode)
}

return nil
```

## Frontend Integration (React)

### 1. Setup Apollo Client

```typescript
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';

const client = new ApolloClient({
  link: new HttpLink({
    uri: 'http://localhost:8080/api',
    credentials: 'include',
    headers: {
      'X-Tenant-ID': tenantId,
      'X-User-ID': userId,
    },
  }),
  cache: new InMemoryCache(),
});
```

### 2. Create Metric Query

```typescript
const GET_METRICS = gql`
  query GetMetrics($tenantId: String!) {
    metrics(where: { tenant_id: { _eq: $tenantId } }) {
      metric_id
      name
      display_name
      domain
      aggregation_function
      sla_freshness_hours
    }
  }
`;
```

### 3. Create Metric Component

```typescript
export const MetricsList: React.FC = () => {
  const { data, loading } = useQuery(GET_METRICS);

  return (
    <div>
      {loading && <p>Loading...</p>}
      {data?.metrics.map(m => (
        <div key={m.metric_id}>
          <h3>{m.display_name}</h3>
          <p>{m.domain} - {m.aggregation_function}</p>
          <button onClick={() => triggerPopCompute(m.metric_id)}>
            Compute PoP
          </button>
        </div>
      ))}
    </div>
  );
};
```

## Deployment Checklist

- [ ] Postgres schema initialized with DDL
- [ ] Backend routes registered and working
- [ ] Metric CRUD API endpoints tested
- [ ] Compute trigger endpoints tested
- [ ] Temporal client initialized and worker registered
- [ ] Temporal workflows executing background computations
- [ ] Trino client connected and SQL executing
- [ ] PoP results appearing in Iceberg
- [ ] Anomaly detections appearing in Iceberg
- [ ] RabbitMQ connected for event publishing
- [ ] Cube.dev connected for pre-agg refresh
- [ ] Frontend consuming metric APIs
- [ ] Frontend displaying PoP charts and anomalies
- [ ] Production Postgres backup configured
- [ ] Monitoring/alerting configured for SLA violations

## Production Considerations

### Monitoring
- Track computation latency (target: <5min monthly, <1min daily)
- Monitor pre-agg refresh lag (target: <1min after job completion)
- Alert on SLA violations (stale metrics, failed runs)

### Scaling
- Connection pooling tuned for concurrent workflows
- Trino query optimization for large datasets
- Iceberg partitioning by tenant + time for fast queries

### Reliability
- Temporal retry policies configured
- Transactional consistency via natural keys (tenant, metric, period)
- Dead letter queues for failed events
- Idempotent re-runs and backfill patterns

---

See `CALC_ENGINE_QUICKSTART.md` for detailed testing instructions and example API calls.
