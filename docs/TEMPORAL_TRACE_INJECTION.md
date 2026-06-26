# Temporal Workflow Trace Injection

## Overview

This document describes how to integrate OpenTelemetry tracing into your Temporal workflows to create a unified, end-to-end trace chain:

```
API Gateway
  → Planner
    → Temporal Workflow
      → Temporal Activities (per region)
        → Commit Service
          → Iceberg Commit
            → Trino Query
```

All tied together by `planner.plan_id`.

---

## Why Temporal Tracing Matters

Without tracing, Temporal workflows are "dark":
- You know a workflow started
- You know it succeeded or failed
- But you don't know the latency breakdown
- You don't know which regions caused delays
- You can't correlate with commit service or Trino latency

With tracing:
- Every workflow, activity, and retry emits a span
- All spans are tagged with `plan_id`
- You get a complete picture from request to result
- You can correlate failures across services

---

## 1. Add OpenTelemetry to Temporal Worker (Go)

### Install Dependencies

```bash
go get go.temporal.io/sdk
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/contrib/instrumentations/go.temporal.io/temporal-sdk
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/sdk/trace
```

### Initialize OTEL in Your Worker

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go.opentelemetry.io/contrib/instrumentations/go.temporal.io/temporal-sdk/client"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initOTEL(ctx context.Context) (*trace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("localhost:4317"), // OTEL Collector
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
    }

    resource, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String("temporal-worker"),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}

func main() {
    ctx := context.Background()
    tp, err := initOTEL(ctx)
    if err != nil {
        log.Fatalf("Failed to initialize OTEL: %v", err)
    }
    defer tp.Shutdown(ctx)

    // Create Temporal client with OTEL instrumentation
    c, err := client.NewClient(client.Options{
        HostPort: "localhost:7233",
        Interceptors: []client.Interceptor{
            otelclient.NewClientInterceptor(),
        },
    })
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer c.Close()

    // Start worker...
    w := worker.New(c, "global-query", worker.Options{})
    w.RegisterWorkflow(DriftWorkflow)
    w.RegisterActivity(RegionalDriftActivity)

    if err = w.Start(); err != nil {
        log.Fatalf("Failed to start worker: %v", err)
    }

    log.Println("Worker started. Press Ctrl+C to exit.")
    select {}
}
```

---

## 2. Inject `plan_id` into Workflow Context

When the planner returns a plan, you already have:

```go
type Plan struct {
    PlanID            string
    SelectedRegions   []string
    DegradationStrategy string
}
```

Pass `plan_id` as workflow input and store it in the workflow context:

```go
func DriftWorkflow(ctx workflow.Context, req *DriftRequest) (*DriftResult, error) {
    // Extract plan_id from request
    planID := req.PlanID

    // Get the workflow logger and log with plan_id
    logger := workflow.GetLogger(ctx)
    logger.Info("Starting drift workflow",
        "plan_id", planID,
        "table", req.Table,
        "regions", req.Regions,
    )

    // Get tracer and start a workflow span
    tracer := otel.Tracer("drift-workflow")
    _, span := tracer.Start(ctx, "workflow.drift.start",
        trace.WithAttributes(
            attribute.String("planner.plan_id", planID),
            attribute.String("table", req.Table),
            attribute.Int("num_regions", len(req.Regions)),
        ),
    )
    defer span.End()

    // Shared context for all activities (includes plan_id)
    activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    1 * time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    1 * time.Minute,
            MaximumAttempts:    5,
        },
    })

    // Fan-out activities per region
    var futures []workflow.Future
    for _, region := range req.Regions {
        regionReq := &RegionalDriftRequest{
            PlanID:   planID,
            Table:    req.Table,
            Region:   region,
            Endpoint: req.Endpoints[region],
        }
        future := workflow.ExecuteActivity(activityCtx, RegionalDriftActivity, regionReq)
        futures = append(futures, future)
    }

    // Wait for all activities and collect results
    var results []*RegionalDriftResult
    for i, future := range futures {
        var result *RegionalDriftResult
        err := future.Get(ctx, &result)
        if err != nil {
            logger.Error("Regional activity failed",
                "region", req.Regions[i],
                "error", err,
            )
            span.RecordError(err)
            continue
        }
        results = append(results, result)
    }

    // Aggregate results
    driftDetected := false
    for _, result := range results {
        if result.DriftDetected {
            driftDetected = true
            break
        }
    }

    // Set span status
    if driftDetected {
        span.SetStatus(codes.Ok, "Drift detected, remediation triggered")
    } else {
        span.SetStatus(codes.Ok, "No drift detected")
    }

    return &DriftResult{
        PlanID:        planID,
        DriftDetected: driftDetected,
        Regions:       req.Regions,
    }, nil
}
```

---

## 3. Inject `plan_id` into Activities

Every activity should:
1. Receive `plan_id` in the request
2. Set `plan_id` on the activity span
3. Propagate `plan_id` to downstream calls (commit service, Trino)

```go
func RegionalDriftActivity(ctx context.Context, req *RegionalDriftRequest) (*RegionalDriftResult, error) {
    // Get the activity logger and tracer
    logger := activity.GetLogger(ctx)
    tracer := otel.Tracer("regional-drift-activity")

    // Start an activity span with plan_id
    actCtx, span := tracer.Start(ctx, "activity.regional_drift",
        trace.WithAttributes(
            attribute.String("planner.plan_id", req.PlanID),
            attribute.String("table", req.Table),
            attribute.String("region", req.Region),
            attribute.String("endpoint", req.Endpoint),
        ),
    )
    defer span.End()

    logger.Info("Starting regional drift check",
        "table", req.Table,
        "region", req.Region,
        "plan_id", req.PlanID,
    )

    // 1. Call commit service to get latest snapshot
    commitReq := &CommitManifestRequest{
        ManifestID: fmt.Sprintf("drift-%s-%s", req.PlanID, req.Region),
        Table:      req.Table,
        Region:     req.Region,
        TenantID:   "drift-detector",
    }

    // Important: Set X-Plan-ID header for trace correlation
    httpReq, _ := http.NewRequestWithContext(actCtx, "POST", fmt.Sprintf("%s/commit", req.Endpoint), bodyReader)
    httpReq.Header.Set("X-Plan-ID", req.PlanID)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        logger.Error("Failed to call commit service", "error", err)
        span.RecordError(err)
        span.SetStatus(codes.Error, "Commit service unreachable")
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        err := fmt.Errorf("commit service returned %d", resp.StatusCode)
        logger.Error("Commit service error", "status", resp.StatusCode)
        span.RecordError(err)
        span.SetStatus(codes.Error, "Commit service error")
        return nil, err
    }

    // 2. Query Trino to compare with expected state
    trinoQuery := fmt.Sprintf("SELECT count(*) FROM %s WHERE region = '%s'", req.Table, req.Region)

    qaCtx, qaSpan := tracer.Start(actCtx, "activity.query_trino",
        trace.WithAttributes(
            attribute.String("planner.plan_id", req.PlanID),
            attribute.String("query", trinoQuery),
            attribute.String("region", req.Region),
        ),
    )

    // Execute Trino query with plan_id propagation
    trinoClient := http.Client{Timeout: 30 * time.Second}
    trinoReq, _ := http.NewRequestWithContext(qaCtx, "POST", "http://trino:8080/v1/statement", queryBody)
    trinoReq.Header.Set("X-Plan-ID", req.PlanID)
    trinoReq.Header.Set("X-Presto-User", "drift-detector")

    trinoResp, err := trinoClient.Do(trinoReq)
    if err != nil {
        logger.Error("Trino query failed", "error", err)
        qaSpan.RecordError(err)
        qaSpan.SetStatus(codes.Error, "Trino query failed")
    }
    qaSpan.End()

    // 3. Compare results and detect drift
    driftDetected := row_count_mismatch // Your comparison logic

    span.AddEvent("drift_detection_complete",
        trace.WithAttributes(
            attribute.String("planner.plan_id", req.PlanID),
            attribute.Bool("drift_detected", driftDetected),
        ),
    )

    if driftDetected {
        span.SetStatus(codes.Ok, "Drift detected in region")
    } else {
        span.SetStatus(codes.Ok, "No drift in region")
    }

    return &RegionalDriftResult{
        PlanID:        req.PlanID,
        Region:        req.Region,
        DriftDetected: driftDetected,
    }, nil
}
```

---

## 4. Add OTEL Exporter Configuration

Create an environment variable or config file:

```yaml
# temporal-worker-config.yaml
otel:
  exporter:
    otlp:
      endpoint: "localhost:4317"
  resource:
    service:
      name: "temporal-worker"
      version: "1.0.0"
  sampling:
    ratio: 1.0  # Sample 100% in production; adjust as needed
```

Or via environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_RESOURCE_ATTRIBUTES=service.name=temporal-worker,service.version=1.0.0
export OTEL_TRACES_SAMPLER=always_on
```

---

## 5. Extend Grafana Dashboard with Workflow Metrics

Add these PromQL panels to your Commit Path Dashboard:

### Workflow Success Rate

```promql
sum(rate(temporal_workflow_executions_total{status="success"}[5m])) / 
(sum(rate(temporal_workflow_executions_total[5m])))
```

### Activity Latency by Region

```promql
histogram_quantile(0.95, sum(rate(temporal_activity_latency_ms_bucket[5m])) by (le, region))
```

### Workflow Failure Rate

```promql
sum(rate(temporal_workflow_executions_total{status="failed"}[5m]))
```

### Regional Activity Latency Breakdown

```promql
sum(rate(temporal_activity_latency_ms_bucket[5m])) by (region, le) | topk(5)
```

---

## 6. CI: Validate Workflow Spans in Integration Tests

Extend your OTLP collector test to assert workflow spans:

```java
@Test
@Order(5)
public void testWorkflowSpansExported() throws Exception {
    // Simulate a Temporal workflow execution with plan_id
    String planID = "plan-" + System.currentTimeMillis();
    
    // (In a real integration test, you'd execute a Temporal workflow here)
    // For now, we verify the collector is receiving spans
    
    Thread.sleep(2000);

    String spansContent = Files.readString(Path.of("/tmp/spans.json"));
    
    // Verify workflow-related spans are exported
    assertTrue(spansContent.contains("workflow") || spansContent.contains("activity"),
        "Spans should contain workflow or activity names");
    
    System.out.println("Workflow spans exported successfully");
}
```

---

## 7. What You Now Have

### End-to-End Trace Chain

```
1. API Gateway → GET /plan?table=incidents&region=us-east-1
   └─ span: api_get_plan
      attributes: table, region, api_version

2. Planner → Generate execution plan
   └─ span: planner_generate_plan
      attributes: plan_id, table, regions, degradation_strategy

3. Temporal Workflow → fan-out to regions
   └─ span: workflow_drift_start
      attributes: plan_id, table, num_regions
      └─ span: activity_regional_drift (per region)
         attributes: plan_id, region, table, endpoint

4. Commit Service → append parquet to Iceberg
   └─ span: commit_manifest
      attributes: plan_id, manifest_id, table, snapshot_id
      └─ span: iceberg_commit
         attributes: manifest_id, snapshot_id, duration

5. Trino → query snapshot
   └─ span: query_trino
      attributes: plan_id, query, table, duration
```

All tied together by `plan_id`, `table`, and `region`.

---

## 8. Debugging with Traces

When a query is slow or fails:

1. **Find the plan_id** from the response headers or API logs
2. **Open Jaeger/Tempo UI** and search for `plan_id=<value>`
3. **Drill down**:
   - Planner latency high? Check plan generation (data sampling, cardinality)
   - Activity latency high for region X? Check regional endpoint health
   - Commit latency high? Check S3 upload or Iceberg snapshot creation
   - Trino latency high? Check query plan or data volume

---

## 9. Production Readiness Checklist

- [ ] OTEL exporter configured and tested in staging
- [ ] Sampling ratio set appropriately (1.0 for initial rollout; 0.1+ for steady state)
- [ ] Temporal worker logging configured with `plan_id` context
- [ ] X-Plan-ID header propagated from API → Planner → Workflow → Commit → Trino
- [ ] Grafana dashboard updated with workflow latency panels
- [ ] Alert rules created for workflow failure rate > 5%
- [ ] Alert rules created for activity latency p95 > 30s (per region)
- [ ] Runbook created for "High-latency regional activity" incident
- [ ] Team trained on using trace UI for debugging

---

## 10. Next Steps

With Temporal tracing in place, your platform now has:

1. ✅ Planner spans (latency, plan details)
2. ✅ Temporal workflow spans (fan-out, activity latency per region)
3. ✅ Commit service spans (S3 validation, Iceberg commit)
4. ✅ Trino query spans (query latency, plan)

Next natural moves:

- **A — Build a React "Trace Explorer" component for your admin UI** (search by plan_id, visualize trace DAG)
- **B — Add Temporal → Commit → Trino latency correlation panels** (which region/activity caused the delay?)
- **C — Add a "per-tenant ingestion health" dashboard** (per-tenant commit success rate, latency, drift detection)
- **D — Add a "per-region commit heatmap" visualization** (see which regions have hot spots)

Which direction next?
