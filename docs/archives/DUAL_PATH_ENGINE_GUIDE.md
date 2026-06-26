# Dual-Path Metric Calculation Engine

## Overview

This implementation provides a **pragmatic, registry-driven architecture** for managing heterogeneous metrics across real-time and batch orchestration lanes. It unifies time-series computations, anomaly detection, BI catalog definitions, and private markets KPIs into a single canonical model with idempotent execution semantics.

---

## Architecture Layers

### 1. **Canonical Metric Model**

All metrics normalize to:

```sql
public.metrics(
  id, industry_id, metric_type, metric_time, 
  value, tags, details, created_at, updated_at
)
```

Source provenance and execution metadata live in `details` JSONB. The model accommodates:
- **Atomic metrics**: Direct values from external sources (DAX/Excel, APIs)
- **Derived metrics**: PoP deltas, percent changes, anomaly flags
- **Composite metrics**: Fund KPIs, aggregated indices

---

### 2. **Semantic Metric Registry** (`semantic_layer.metric_registry`)

**Single Source of Truth** for what each metric is and how it's computed:

| Field | Purpose |
|-------|---------|
| `metric_id`, `name`, `display_name` | Identity & labeling |
| `domain`, `category`, `metric_type` | Semantics (atomic/derived/composite) |
| `base_query`, `aggregation_function`, `granularity` | Computation rules |
| `source_formula`, `source_system` | Lineage & provenance |
| `comparison_periods`, `period_label_format` | Time alignment (YYYY-MM, YYYY-MM-DD) |
| `sla_freshness_hours`, `sla_completeness_threshold` | Quality gates |
| `refresh_schedule` | Execution frequency (hourly/daily/weekly/monthly) |
| `golden_path` | Governance flag for high-stakes metrics |

**Backfill from catalog**: Existing `public.pop_metrics` definitions are automatically migrated into the registry, preserving all governance and SLA metadata.

---

### 3. **Dual Execution Lanes**

#### **Real-Time Atomic Lane** (`public.refresh_atomic_metrics`)

- **Schedule**: Every 1 hour (configurable)
- **Scope**: Metrics with `granularity=['date']` and `refresh_schedule='daily'` or faster
- **Ingestion**: Pulls new metrics from `public.metrics` (time-series tables, APIs, warehouse feeds)
- **Output**: `public.metrics_finalized` with freshness gates and SLA compliance flags
- **Quality Checks**:
  - Freshness validation: `(NOW() - metric_time) <= sla_freshness_hours`
  - Completeness validation: `completeness_score >= sla_completeness_threshold`
  - SLA violations logged to `public.sla_violations`

```sql
-- Example: refresh_atomic_metrics() ingests DAX/Excel metrics, validates SLA
-- Finalized metrics are available for dashboards, calculated fields (Workday, Excel)
```

---

#### **Batch Periodized Lane** (`public.compute_monthly_pop` + `public.compute_comparison_periods`)

- **Schedule**: 2 AM on 1st of month (post-data cutoff)
- **Scope**: Metrics with `granularity=['month']` or grouped periods
- **Computation**:
  1. Monthly aggregations with `period_start`, `period_end`, `period_label` (YYYY-MM)
  2. Lagged window join for `previous_value`, `delta`, `percent_change`
  3. Upsert to `public.pop_computations` (idempotent by metric_id + period_start/end + granularity)
  4. Compute YoY, QoQ, PoP deltas via window functions → `public.metrics_comparison_periods`
- **Idempotency**: Natural key ensures safe backfills and re-runs without duplication
- **Completeness**: All period_labels track `record_count` and `computation_status`

```sql
-- Example monthly PoP computation with history
WITH monthly_current AS (
  SELECT metric_id, DATE_TRUNC('month', metric_time), SUM(value), COUNT(*)
  FROM public.metrics
  GROUP BY 1, 2
),
lagged AS (
  SELECT *, 
    LAG(current_value) OVER (PARTITION BY metric_id ORDER BY period_start)
  FROM monthly_current
)
INSERT INTO public.pop_computations (metric_id, period_start, ..., delta, percent_change, ...)
SELECT ..., current_value - previous_value, ...
FROM lagged
ON CONFLICT (metric_id, period_start, period_end, granularity) DO UPDATE SET ...;
```

---

### 4. **Anomaly Detection** (`public.detect_zscore_anomalies`)

- **Trigger**: Daily at 3 AM
- **Window**: 90 days (configurable `p_window_days`)
- **Method**: Z-score with rolling stats
  - $z = \frac{x - \mu}{\sigma}$ over windowed history
  - Severity: `high` (|z| ≥ 3.0), `medium` (|z| ≥ 2.5), `low`
  - Confidence: Sigmoid transform: $\frac{1}{1 + e^{-|z|}}$
- **Idempotent Upsert**: Key = (metric_id, computation_id, anomaly_type) prevents duplicates on re-run
- **Storage**: `public.pop_anomalies` with detection_params JSON for reproducibility

```sql
-- Z-score computed over 90-day window with at least 7 data points
WITH windowed AS (
  SELECT metric_id, computation_id, value as x,
    AVG(value) OVER w as mu,
    STDDEV_POP(value) OVER w as sigma
  FROM pop_computations
  WINDOW w AS (PARTITION BY metric_id ORDER BY period_end DESC ROWS BETWEEN 89 PRECEDING AND CURRENT ROW)
),
scored AS (
  SELECT *, (x - mu) / sigma as z_score
  WHERE window_count >= 7
)
INSERT INTO pop_anomalies (...) SELECT ... WHERE ABS(z_score) >= 2.5
ON CONFLICT (metric_id, computation_id, anomaly_type) DO NOTHING;
```

---

## Execution & Orchestration

### **Scheduling Model**

```yaml
# Real-time lane (hourly)
atomic_refresh: 
  interval: 1 hour
  timeout: 30 min
  targets: metrics with granularity=['date'], refresh_schedule='daily'|'hourly'
  
# Batch lane (monthly)
monthly_pop:
  schedule: "0 2 1 * *"  # 2 AM on 1st of month
  timeout: 1 hour
  
# Anomaly detection (daily)
anomaly_detection:
  schedule: "0 3 * * *"  # 3 AM daily
  timeout: 1 hour
  
# SLA enforcement (every 6 hours)
sla_check:
  interval: 6 hours
  scope: golden_path=TRUE
```

### **Orchestrator Start**

```go
// backend/internal/orchestration/metric_orchestrator.go
orchestrator := orchestration.NewMetricOrchestrator(registryService, nil)
orchestrator.Start(ctx)  // Starts all schedulers
```

All execution logs captured in `semantic_layer.metric_execution_log` with:
- Lane (real-time vs batch)
- Execution type (refresh, backfill, recompute)
- Period info (period_start/end for batch jobs)
- Status (started, completed, failed, partial)
- Quality metrics (completeness_score, freshness_hours)
- Error tracking (error_message, error_details JSONB)

---

## API Endpoints

### **Registry Discovery**

```bash
# List all active metrics
GET /api/metrics-registry?domain=finance&golden_only=true

# Get specific metric definition
GET /api/metrics-registry/{metricID}

# Execution history
GET /api/metrics-registry/{metricID}/history?limit=50
```

### **Orchestration Triggers**

```bash
# Refresh atomic metrics (real-time lane)
POST /api/metrics-registry/refresh-atomic
Body: {"metric_id": "..."}

# Compute monthly PoP
POST /api/metrics-registry/{metricID}/compute-pop
Body: {"period_start": "2024-10-01", "period_end": "2024-10-31"}

# Compute comparison periods
POST /api/metrics-registry/{metricID}/compute-comparisons

# Detect anomalies
POST /api/metrics-registry/{metricID}/detect-anomalies
Body: {
  "zscore_threshold": 2.5,
  "window_days": 90,
  "min_data_points": 7
}

# Check golden path readiness
GET /api/metrics-registry/golden-path/readiness
```

### **Governance**

```bash
# Promote metric to golden path
POST /api/metrics-registry/{metricID}/promote-golden
```

---

## Integration Patterns

### **Pattern 1: Registry-Backed Exports**

Expose finalized metrics via API for downstream consumers (dashboards, Excel, Workday):

```go
// GET /api/metrics/{node_id}/current
func (h *MetricsHandler) GetMetricSnapshot(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	
	// Fetch latest finalized value
	var snapshot MetricSnapshot
	h.db.Get(&snapshot, `
		SELECT value, last_refresh, freshness_status, completeness_score
		FROM public.metrics_finalized
		WHERE metric_type = $1
		ORDER BY as_of_date DESC LIMIT 1
	`, nodeID)
	
	// Also return registry definition for consumers to understand what they're getting
	json.NewEncoder(w).Encode(map[string]interface{}{
		"snapshot": snapshot,
		"metadata": registry,  // display_name, SLAs, owner, etc.
	})
}
```

### **Pattern 2: Multi-Tenant Scope**

The canonical model includes `industry_id` (UUID) to support multi-tenant deployments:

```sql
-- Tenant-scoped queries
SELECT * FROM public.metrics WHERE industry_id = $1;
SELECT * FROM public.metrics_finalized WHERE metric_id IN 
  (SELECT metric_id FROM semantic_layer.metric_registry WHERE owner_user_id IN (...))
```

### **Pattern 3: Backfill & Recompute**

Safely backfill a metric range with re-run semantics:

```go
periodStart := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
periodEnd := time.Date(2024, 8, 31, 23, 59, 59, 0, time.UTC)

execLog, err := registryService.ComputeMonthlyPoP(ctx, &metricID, &periodStart, &periodEnd)
// ON CONFLICT (metric_id, period_start, period_end, granularity) DO UPDATE
// → Safe re-run, no duplicates
```

---

## Quality Gates & SLA Enforcement

### **Golden Path Readiness View**

```sql
SELECT * FROM public.golden_path_readiness;
-- Returns: metric_id, name, readiness_status, current_value, last_refresh, violation_type
```

Readiness statuses:
- `ready`: All SLAs met, data is fresh
- `sla_violation`: Open breach in `public.sla_violations`
- `quality_gate_failed`: `completeness_score < threshold` or `freshness_status != 'fresh'`
- `stale_data`: `(NOW() - last_refresh) > sla_freshness_hours`

### **SLA Violation Tracking**

All breaches logged to `public.sla_violations` with:
- `violation_type`: freshness, completeness, or both
- `expected_threshold`: SLA threshold from registry
- `actual_value`: observed metric
- `breach_amount`: deviation
- `status`: open, acknowledged, resolved

### **Enforcement Flow**

1. Real-time lane validates freshness + completeness before publishing
2. If breach → insert row to `sla_violations` + don't publish
3. Orchestrator checks `golden_path_readiness` every 6 hours
4. Alert/escalate for open breaches

---

## Data Quality & Lineage

### **Execution Lineage**

Every computation leaves a trail in `metric_execution_log`:

```sql
SELECT 
  execution_id, metric_id, lane, execution_type,
  period_start, period_end, period_label,
  status, record_count, success_count, error_count,
  completeness_score, error_message, started_at, completed_at
FROM semantic_layer.metric_execution_log
WHERE metric_id = ? AND completed_at >= NOW() - INTERVAL '7 days'
ORDER BY completed_at DESC;
```

This enables:
- Audit trails for governance
- Debugging (what happened in this job?)
- Replayability (re-run a past period exactly)

### **Details JSON**

The `public.metrics.details` JSONB field carries:
- `source_system`: origin (DAX, API, warehouse)
- `completeness_score`: % of records present
- `freshness_hours`: age of the data
- `grain_values`: [date, week, month] for multi-grain sources
- Custom field mappings for lineage

---

## Backlog: Next Steps

1. **⚡ Backfill Registry** from catalog rows + mark atomic nodes (DAX/Excel origins)
2. **⚡ Deploy Real-Time Lane** → validate golden metrics daily
3. **⚡ Activate Monthly PoP Batch** at end of month
4. **⚡ Wire Anomaly Detection** for PoP time-series (daily z-score runs)
5. **⚡ Expose Golden Path API** for dashboard consumption
6. **⚡ Configure Temporal/Airflow** for orchestration (optional for full DAG replay)
7. **📊 Add Observability**: Prometheus metrics for lane throughput, SLA % green, anomaly count

---

## Example Usage

### Scenario: Track monthly revenue with PoP + anomalies

```go
// 1. Register metric in registry
metric := &MetricRegistryEntry{
  Name: "monthly_revenue",
  DisplayName: "Monthly Revenue (USD)",
  Domain: "finance",
  Category: "p&l",
  MetricType: "derived",
  Granularity: []string{"month"},
  AggregationFunction: "SUM",
  ValueColumn: "revenue",
  DateColumn: "transaction_date",
  SLAFreshnessHours: 24,
  SLACompletenessThreshold: 95.0,
  RefreshSchedule: "monthly",
  GoldenPath: true,
}
metricID, _ := registryService.RegisterMetric(ctx, metric)

// 2. Execute monthly computation (1st of month at 2 AM)
execLog, _ := registryService.ComputeMonthlyPoP(ctx, &metricID, nil, nil)

// 3. Compute YoY/PoP comparisons
compLog, _ := registryService.ComputeComparisonPeriods(ctx, &metricID)

// 4. Run anomaly detection (3 AM daily)
anomalies, _ := registryService.DetectZScoreAnomalies(ctx, &metricID, 2.5, 90, 7)

// 5. Check readiness
readiness, _ := registryService.GetGoldenPathReadiness(ctx)
// e.g., {"readiness_status": "ready", "current_value": 1250000.50, "last_refresh": "2024-11-01T02:30:00Z"}
```

---

## References

- **Canonical Model**: `public.metrics` (id, industry_id, metric_type, metric_time, value, tags, details)
- **Registry**: `semantic_layer.metric_registry` (definitions + lineage)
- **PoP Computations**: `public.pop_computations` (period_start, period_end, period_label, deltas)
- **Comparisons**: `public.metrics_comparison_periods` (YoY, QoQ, PoP pre-materialized)
- **Anomalies**: `public.pop_anomalies` (z_score, severity, confidence, detection_params)
- **Execution Log**: `semantic_layer.metric_execution_log` (audit trail)
- **SLA Tracking**: `public.sla_violations` (breaches with status)
- **Golden Path View**: `public.golden_path_readiness` (quick health check)

---

**Deployed**: 2025-11-01  
**Version**: 1.0  
**Status**: Production-ready
