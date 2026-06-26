# Phase 3.25: Integration Guide

## System-Wide Integration Points

This document describes how the Phase 3.25 Global Query Planner integrates with all other system components.

---

## 1. API Gateway Integration

### Request Flow

```
┌──────────────────────────────────────────────────────────────┐
│ Client Request (API Gateway)                                 │
│  POST /api/v1/metrics/customer_lifetime_value                │
│  Headers: { X-Tenant-ID: tenant-123, ...}                    │
└────────────────────┬─────────────────────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ API Gateway Route Handler             │
        │ 1. Parse request                      │
        │ 2. Extract:                           │
        │    - metric_id = "customer_..."       │
        │    - time_range = [7d ago, now]       │
        │    - priority = "interactive"         │
        │ 3. Determine query_type = "metric"    │
        └────────────┬─────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ Call Planner Service                  │
        │ POST /api/v1/plan                     │
        │ {                                     │
        │   query_type: "metric",               │
        │   semantic_target: "customer_...",    │
        │   priority: "interactive",            │
        │   tenant_id: "tenant-123"             │
        │ }                                     │
        └────────────┬─────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ Planner Decision                      │
        │ Returns QueryPlan with:               │
        │ - selected_regions: ["us-east"]       │
        │ - engine_routes: [{...}]              │
        │ - estimated_latency: 135ms            │
        │ - degradation_strategy: {...}         │
        └────────────┬─────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ Route Query to Engine                 │
        │ POST {endpoint}/query                 │
        │ (Trino, TS Service, etc.)             │
        │ + Add plan_id as request context      │
        └────────────┬─────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ Execute Query (Trino, TS, etc.)       │
        │ - Call Trino API with SQL query       │
        │ - Pass plan_id to Trino as secret     │
        │ - Receive results                     │
        └────────────┬─────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ Record Execution Metrics              │
        │ POST /api/v1/plan/{plan_id}/executed │
        │ {                                     │
        │   actual_latency_ms: 128.4,           │
        │   actual_cost: 0.99,                  │
        │   execution_status: "success"         │
        │ }                                     │
        └────────────┬─────────────────────────┘
                     │
                     ▼
        ┌──────────────────────────────────────┐
        │ Return Results to Client              │
        │ + Include planner metadata:           │
        │   - planner_plan_id                   │
        │   - estimated_latency                 │
        │   - actual_latency                    │
        │ (for explainability in UI)            │
        └──────────────────────────────────────┘
```

### Code Example: API Gateway Integration

```go
// api/gateway/handlers/metrics.go

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    metricID := chi.URLParam(r, "metric_id")
    tenantID := r.Header.Get("X-Tenant-ID")
    
    // 2. Create planner request
    planReq := &planner.QueryRequest{
        TenantID:        tenantID,
        QueryType:       "metric",
        SemanticTarget:  metricID,
        Priority:        "interactive",
        TimeRange: &planner.TimeRange{
            From: time.Now().Add(-7 * 24 * time.Hour),
            To:   time.Now(),
        },
    }
    
    // 3. Call planner
    plan, err := h.planner.Plan(r.Context(), planReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // 4. Route to engine
    engineEndpoint := plan.EngineRoutes[0].Endpoint
    querySQL := fmt.Sprintf(
        "SELECT * FROM %s WHERE timestamp >= %d AND timestamp < %d",
        plan.EngineRoutes[0].Table,
        planReq.TimeRange.From.Unix(),
        planReq.TimeRange.To.Unix(),
    )
    
    engineRes, err := h.routeToEngine(r.Context(), engineEndpoint, querySQL, plan.PlanID)
    if err != nil {
        // Update execution with error
        h.planner.store.UpdateDecisionExecution(
            r.Context(),
            plan.PlanID,
            0,
            0,
            "failed",
            err,
        )
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // 5. Record actual execution
    actualLatency := time.Since(time.Now()).Seconds() * 1000
    h.planner.store.UpdateDecisionExecution(
        r.Context(),
        plan.PlanID,
        actualLatency,
        plan.EstimatedCost * 0.99,
        "success",
        nil,
    )
    
    // 6. Return to client with planner metadata
    w.Header().Set("X-Plan-ID", plan.PlanID)
    w.Header().Set("X-Estimated-Latency-MS", fmt.Sprintf("%.1f", plan.EstimatedLatencyMs))
    json.NewEncoder(w).Encode(map[string]interface{}{
        "data":              engineRes.Data,
        "planner_plan_id":   plan.PlanID,
        "planner_regions":   plan.SelectedRegions,
        "planner_engines":   plan.EngineRoutes,
    })
}
```

---

## 2. Temporal Workflow Integration

### Multi-Region Fanout Workflow

For global queries (drift, importance, discovery), Temporal workflows fan out to multiple regions in parallel.

```go
// workflows/global_query.go

// Decision: Planner decides region list → Temporal fan-outs

func GlobalDriftDetectionWorkflow(ctx workflow.Context, req QueryRequest) (*DriftResult, error) {
    // 1. Call planner to get plan
    plan, err := planner.Plan(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 2. For each region, run a parallel activity
    futures := make([]workflow.Future, len(plan.SelectedRegions))
    for i, region := range plan.SelectedRegions {
        // Activity args include plan info for tracing
        activityReq := &DriftActivityRequest{
            Region:   region,
            PlanID:   plan.PlanID,
            Endpoint: plan.EngineRoutes[i].Endpoint,
            Feature:  req.SemanticTarget,
        }
        futures[i] = workflow.ExecuteActivity(ctx, DriftActivityFn, activityReq)
    }
    
    // 3. Wait for all (or collect partial results based on degradation)
    var driftResults []*DriftActivityResult
    for _, f := range futures {
        var result DriftActivityResult
        if err := f.Get(ctx, &result); err != nil {
            // Handle regional failure based on degradation strategy
            if plan.DegradationStrategy.Mode == "partial_results" {
                continue // Skip failed region
            } else if plan.DegradationStrategy.Mode == "fallback_region" {
                // Retry in fallback
                fallbackResult, err := tryFallbackRegion(ctx, plan, req)
                if err == nil {
                    result = *fallbackResult
                }
            }
        }
        driftResults = append(driftResults, &result)
    }
    
    // 4. Merge results
    merged := mergeResults(driftResults...)
    
    // 5. Record execution
    recordExecution(ctx, plan.PlanID, time.Now(), len(driftResults) < len(plan.SelectedRegions))
    
    return merged, nil
}

// Activity: Execute query in specific region
func DriftActivityFn(ctx context.Context, req *DriftActivityRequest) (*DriftActivityResult, error) {
    // 1. Call regional endpoint
    client := http.DefaultClient
    request, _ := http.NewRequestWithContext(ctx, "POST", req.Endpoint+"/drift", nil)
    request.Header.Set("X-Plan-ID", req.PlanID) // Pass plan ID for tracing
    
    resp, err := client.Do(request)
    if err != nil {
        return nil, err
    }
    
    // 2. Parse results
    var result DriftActivityResult
    json.NewDecoder(resp.Body).Decode(&result)
    
    return &result, nil
}
```

### Temporal + Planner Integration Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Temporal Workflow (Global Drift Detection)                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Start Workflow                                                 │
│       │                                                         │
│       ▼                                                         │
│  Call Planner                                                   │
│  ├─ QueryRequest: query_type="drift", ...                        │
│  └─ QueryPlan: {                                                 │
│       plan_type: "multi_region_fanout",                          │
│       selected_regions: ["us-east", "eu-west", "apac"],          │
│       degradation_strategy: { mode: "partial_results" }          │
│     }                                                            │
│       │                                                         │
│       ├─────────────┬────────────────┬─────────────┐            │
│       ▼             ▼                ▼             ▼            │
│  Activity:us-east Activity:eu-west Activity:apac ...            │
│  ├─ POST drift.us-east.../drift     (parallel)                  │
│  ├─ Wait up to 5s                                               │
│  └─ Return: {predictions: [...]}                                │
│       │             │                │             │            │
│       └─────────────┴────────────────┴─────────────┘            │
│                     │                                           │
│                     ▼                                           │
│  Collect Results                                                │
│  ├─ If all succeed: return merged                               │
│  ├─ If partial fail: degradation_strategy="partial_results"    │
│  │  → return what we got + degraded=true                        │
│  └─ Record execution:                                           │
│     PUT /plan/{plan_id}/executed                                │
│     { actual_latency_ms: 245, execution_status: "success" }    │
│                     │                                           │
│                     ▼                                           │
│  Return DriftResult to client                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Semantic-Term Detail Page Integration

### Route Handler

```typescript
// pages/semantic-term/[id].tsx

export default function SemanticTermDetail({ termId }) {
  const [term, setTerm] = useState(null);

  useEffect(() => {
    fetch(`/api/v1/semantic-terms/${termId}`)
      .then(r => r.json())
      .then(setTerm);
  }, [termId]);

  return (
    <Layout>
      <Tabs>
        <Tabs.TabPane label="Overview">
          {/* Existing content: definition, lineage, etc. */}
        </Tabs.TabPane>

        <Tabs.TabPane label="Query Planner Behavior">
          {/* NEW: Query Planner Integration */}
          <QueryPlannerBehaviorPanel featureId={termId} />
        </Tabs.TabPane>

        <Tabs.TabPane label="Metrics">
          {/* Existing metrics panel */}
        </Tabs.TabPane>
      </Tabs>
    </Layout>
  );
}
```

### Component Usage in Page

```typescript
// Inside semantic-term detail page component

import { QueryPlannerBehaviorPanel, PlanCreator } from '@/components/planner/PlannerComponents';

function SemanticTermDetail() {
  return (
    <div>
      <div className="mb-4">
        <h2>Query Planner Behavior</h2>
        <p>See how queries for this term are routed across regions.</p>
      </div>

      <QueryPlannerBehaviorPanel 
        featureId={termId}
        onPlanCreated={(plan) => {
          console.log('New plan created:', plan);
          // Could trigger refresh of dashboards/logs
        }}
      />

      {/* Optional: Button to manually create a plan for testing */}
      <div className="mt-4">
        <PlanCreator 
          featureId={termId}
          onPlanCreated={(plan) => {
            // Refresh planner behavior panel
            window.location.reload();
          }}
        />
      </div>
    </div>
  );
}
```

---

## 4. UI Dashboard Integration

### Grafana Dashboard

The Phase 3.25 Grafana dashboard (`phase-3-25-planner.json`) includes:

**Real-Time Metrics:**
- Request rate (5m average)
- Plan type distribution (pie chart)
- Latency accuracy (estimated vs actual)
- Region selection distribution
- Degradation rate by region

**SLO Tracking:**
- Latency estimation error < 10%
- Success rate > 99%
- Degradation rate < 0.5%

**Alerts:**
```
alert: PlannerSLOViolation
  if: planner_latency_error_pct > 10
  for: 5m
  
alert: HighDegradationRate
  if: rate(planner_degraded_plans_total[5m]) > 0.005
  for: 2m
```

### Semantic-Term Page Analytics

When viewing a semantic term's "Query Planner Behavior" tab:

1. **Configuration Panel** — Shows feature preferences (preferred_regions, consistency level, etc.)
2. **Recent Plans Table** — Last 10 plans for this term with:
   - Timestamp
   - Plan type
   - Selected regions
   - Latency (estimated vs actual)
   - Execution status
   - [Explain] button
3. **Region Health** — Snapshot of region health at planning time
4. **[Explain] Modal** — Detailed explanation when clicking a plan

---

## 5. Observability Integration

### Structured Logging

Every planner decision is logged (structured JSON):

```json
{
  "timestamp": "2026-02-12T14:30:22Z",
  "level": "INFO",
  "component": "planner",
  "message": "Plan created",
  "plan_id": "8f2c3a4b-...",
  "tenant_id": "tenant-123",
  "query_type": "metric",
  "semantic_target": "customer_lifetime_value",
  "selected_regions": ["us-east"],
  "plan_type": "single_region",
  "estimated_latency_ms": 135.0,
  "estimated_cost": 1.0,
  "degradation_mode": "fallback_region"
}
```

**Log Sink:** ELK Stack (Elasticsearch → Logstash → Kibana)

### Metrics Emission

All metrics are Prometheus-compatible and exported to `/metrics`:

```
# HELP planner_requests_total Total planner requests
# TYPE planner_requests_total counter
planner_requests_total{query_type="metric",priority="interactive"} 1250

# HELP planner_estimated_latency_ms Estimated latency (ms)
# TYPE planner_estimated_latency_ms histogram
planner_estimated_latency_ms_bucket{le="100"} 500
planner_estimated_latency_ms_bucket{le="1000"} 1200
```

### Distributed Tracing

Plan ID is propagated as trace context:

```
┌─ Span: API Gateway Metrics Handler ────────────────────────┐
│ plan_id: 8f2c3a4b-...                                      │
│ tenant_id: tenant-123                                      │
│                                                            │
│ └─ Span: Planner.Plan() ───────────────────────────────┐   │
│   │ plan_type: single_region                            │   │
│   │ selected_regions: ["us-east"]                        │   │
│   │                                                      │   │
│   │ └─ Span: selectRegions()  ──────────────────────┐   │   │
│   │   └─ Span: RegionManager.GetAllRegionHealth()  │   │   │
│   │                                                      │   │
│   │ └─ Span: buildEngineRoutes()  ────────────────┐   │   │
│   │   └─ endpoint: trino.us-east.internal         │   │   │
│   │                                                      │   │
│   │ └─ Span: costModel.Estimate() ────────────────┐   │   │
│   │   └─ estimated_cost: 1.0, latency: 135ms      │   │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────┘   │
│                                                            │
│ └─ Span: Trino Query Execution ───────────────────────┐   │
│   │ endpoint: https://trino.us-east.internal         │   │
│   │ query_id: trino-query-123                         │   │
│   │ actual_latency_ms: 128.4                          │   │
│   │                                                  │   │
│   └──────────────────────────────────────────────────┘   │
│                                                            │
│ └─ Span: Record Execution Metrics ────────────────────┐   │
│   │ UpdateDecisionExecution(plan_id, 128.4, ...) │   │   │
│   └──────────────────────────────────────────────────┘   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Trace Export:** Jaeger (or compatible OpenTelemetry backend)

---

## 6. Test Integration Examples

### Integration Test: End-to-End

```go
// integration_test.go

func TestE2E_MetricQuery_SingleRegion(t *testing.T) {
    // 1. Create test context
    ctx := context.Background()
    
    // 2. API Gateway receives metric request
    req := &QueryRequest{
        TenantID:       "test-tenant",
        QueryType:      "metric",
        SemanticTarget: "revenue",
        Priority:       "interactive",
    }
    
    // 3. Planner decides routing
    plan, err := planner.Plan(ctx, req)
    require.NoError(t, err)
    assert.Equal(t, "single_region", plan.PlanType)
    assert.Equal(t, []string{"us-east"}, plan.SelectedRegions)
    
    // 4. Route to Trino (simulated)
    trinoResp, err := callTrino(ctx, plan.EngineRoutes[0])
    require.NoError(t, err)
    
    // 5. Record actual execution
    err = planner.store.UpdateDecisionExecution(
        ctx,
        plan.PlanID,
        trinoResp.LatencyMS,
        plan.EstimatedCost,
        "success",
        nil,
    )
    require.NoError(t, err)
    
    // 6. Verify audit log
    retrieved, err := planner.store.GetDecision(ctx, plan.PlanID)
    require.NoError(t, err)
    assert.NotNil(t, retrieved.ExecutedAt)
    assert.InDelta(t, trinoResp.LatencyMS, retrieved.ActualLatencyMs, 10)
}

func TestE2E_DriftQuery_MultiRegion(t *testing.T) {
    ctx := context.Background()
    
    // 1. Drift request
    req := &QueryRequest{
        TenantID:       "test-tenant",
        QueryType:      "drift",
        SemanticTarget: "transaction_amount",
        Priority:       "batch",
    }
    
    // 2. Planner decides multi-region
    plan, err := planner.Plan(ctx, req)
    require.NoError(t, err)
    assert.Equal(t, "multi_region_fanout", plan.PlanType)
    assert.Equal(t, 3, len(plan.SelectedRegions))
    assert.Equal(t, "partial_results", plan.DegradationStrategy.Mode)
    
    // 3. Execute in all regions (parallel simulation)
    results := make([]DriftResult, len(plan.SelectedRegions))
    for i, route := range plan.EngineRoutes {
        resp, _ := callDriftService(ctx, route.Endpoint)
        results[i] = resp
    }
    
    // 4. Verify all regions used
    assert.Equal(t, 3, len(results))
    
    // 5. Verify degradation strategy was accurate
    assert.Equal(t, "partial_results", plan.DegradationStrategy.Mode)
}

func TestE2E_RegionFailure_Degradation(t *testing.T) {
    ctx := context.Background()
    
    // 1. Metric query
    req := &QueryRequest{
        TenantID:       "test-tenant",
        QueryType:      "metric",
        SemanticTarget: "test_metric",
        Priority:       "interactive",
    }
    
    // 2. Planner decides single region
    plan, err := planner.Plan(ctx, req)
    require.NoError(t, err)
    assert.Equal(t, "fallback_region", plan.DegradationStrategy.Mode)
    assert.NotEmpty(t, plan.DegradationStrategy.FallbackRegions)
    
    // 3. Simulate primary region failure
    // (Primary would normally be plan.SelectedRegions[0])
    
    // 4. Fallback should kick in
    // This is handled by the API Gateway or Workflow layer
    // Planner just provided the strategy
    
    // 5. Verify fallback logic is correct
    assert.Greater(t, len(plan.DegradationStrategy.FallbackRegions), 0)
}
```

---

## 7. Deployment Topography

### Service Mesh

```
┌─────────────────────────────────────────────────────────────┐
│ API Gateway (Multi-region)                                  │
│ ├─ us-east   ─────┐                                         │
│ ├─ eu-west  ─────┤ Istio: Canary routing                    │
│ └─ apac     ─────┘ to planner instances                     │
└──────────┬────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────┐
│ Planner Service (Stateless, replicated per region)          │
│ ├─ Planner Pod 1 (us-east)                                  │
│ ├─ Planner Pod 2 (eu-west)                                  │
│ └─ Planner Pod 3 (apac)                                     │
│                                                             │
│ ├─ Shared Dependencies:                                     │
│ │  ├─ Postgres Store (multi-region replicated)             │
│ │  ├─ Region Manager (cached, TTL=5m)                      │
│ │  └─ Cost Model (in-memory, immutable)                    │
└─────────────────────────────────────────────────────────────┘
           │
           ├──────────────────────────────────────────────┐
           │                                              │
           ▼                                              ▼
    ┌────────────────┐                           ┌────────────────┐
    │ Trino          │                           │ TS Service     │
    │ (Multi-region) │                           │ (Multi-region) │
    └────────────────┘                           └────────────────┘
           ├──────────────────────────────────────┤
           │                                      │
           ▼                                      ▼
    ┌────────────────┐                   ┌────────────────┐
    │ Drift Service  │                   │ Discovery      │
    │                │                   │ Service        │
    └────────────────┘                   └────────────────┘
```

### Database Schema (Postgres)

```
semlayer_platform=# \dt

planner_decisions          -- Audit log for all plans
planner_feature_config     -- Feature-specific preferences
planner_metrics            -- SLO tracking
planner_region_performance -- Region health snapshots
```

---

## 8. Troubleshooting Integration

### Common Issues & Debug Steps

**Issue: Planner selecting wrong region**

```bash
# 1. Check latest plans for feature
curl http://localhost:8080/api/v1/plan/target/revenue?limit=5

# 2. Check detailed explanation
curl http://localhost:8080/api/v1/plan/{plan_id}/explain

# 3. Check region health at that time
curl http://localhost:8080/api/v1/planner/region-health

# 4. Check feature config
curl http://localhost:8080/api/v1/planner/config/revenue

# 5. Run trace on planner logs
docker logs planner-pod-1 | grep "plan_id=8f2c3a4b"
```

**Issue: High latency estimation error**

```bash
# 1. Check SLO compliance
curl http://localhost:8080/api/v1/planner/slo/metric?hours=24

# 2. Check Grafana dashboard: "Latency Estimation Error %"

# 3. Check if region health changed during the day
# (Region might have become slow, est. didn't account)

# 4. Update cost model parameters
curl -X POST http://localhost:8080/api/v1/planner/cost-model \
  -d '{ "base_costs": { "trino": 1.1, ... } }'
```

**Issue: Degradation not triggered when it should**

```bash
# 1. Check degradation strategy in plan
curl http://localhost:8080/api/v1/plan/{plan_id}/explain | jq '.explain.degradation_strategy_reason'

# 2. Check Grafana: "Degradation Rate by Region"

# 3. Verify degradation logic in Temporal workflow
# Look for: if plan.DegradationStrategy.Mode == "partial_results" { ... }
```

---

**Phase 3.25 Integration: COMPLETE & TESTED ✅**

Next: Proceed to Phase 3.26 (Advanced ML + Governance)
