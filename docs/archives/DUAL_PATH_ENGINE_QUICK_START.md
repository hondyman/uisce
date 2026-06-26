# Dual-Path Metric Engine: Quick Start

## 1. Deploy Schema Migrations

```bash
# Run migrations in order
cd backend/migrations

# Run migration 13 (metric registry schema)
psql -f 000013_metric_registry_and_dual_path_engine.sql

# Run migration 14 (execution procedures)
psql -f 000014_dual_path_execution_procedures.sql
```

This creates:
- `semantic_layer.metric_registry` — registry of all metrics
- `public.metrics_finalized` — real-time atomic metrics
- `public.metrics_comparison_periods` — YoY/QoQ pre-computed
- `public.sla_violations` — breach tracking
- Execution views and stored procedures

---

## 2. Backfill Registry from Existing Catalog

The migration automatically migrates `pop_metrics` into the registry. Verify:

```sql
SELECT COUNT(*) FROM semantic_layer.metric_registry;
-- Should return count of migrated pop_metrics

SELECT name, display_name, domain, golden_path, refresh_schedule
FROM semantic_layer.metric_registry
ORDER BY domain, category
LIMIT 10;
```

---

## 3. Register Additional Atomic Metrics

If you have DAX/Excel sources or API-fed metrics not yet in the registry:

```sql
INSERT INTO semantic_layer.metric_registry (
  name, display_name, domain, category, metric_type,
  source_formula, source_system, granularity, 
  sla_freshness_hours, sla_completeness_threshold, refresh_schedule,
  golden_path
) VALUES 
  ('clean_price', 'Clean Price', 'finance', 'pricing', 'atomic',
   'DAX formula', 'Bloomberg', ARRAY['date'], 24, 95.0, 'daily', FALSE),
  ('net_interest_margin', 'Net Interest Margin', 'finance', 'risk',
   'atomic', 'Excel formula', 'DataLake', ARRAY['date'], 24, 95.0, 'daily', TRUE);
```

---

## 4. Initialize Orchestrator in Your Server

In your main entry point (e.g., `cmd/server/main.go`):

```go
import (
	"github.com/hondyman/semlayer/backend/internal/orchestration"
	"github.com/hondyman/semlayer/backend/internal/services"
)

func main() {
	// ... database setup ...
	
	// Create registry service
	registryService := services.NewMetricRegistryService(db)
	
	// Create orchestrator with default config (or customize)
	orchConfig := orchestration.DefaultConfig()
	// Optionally customize:
	// orchConfig.AtomicRefreshInterval = 30 * time.Minute
	// orchConfig.DefaultZScoreThreshold = 3.0
	
	orchestrator := orchestration.NewMetricOrchestrator(registryService, orchConfig)
	
	// Start all schedulers (real-time, batch, anomaly, SLA checks)
	orchestrator.Start(context.Background())
	defer orchestrator.Stop()
	
	// ... rest of server ...
}
```

---

## 5. Register HTTP Routes

In your router setup:

```go
import "github.com/hondyman/semlayer/backend/internal/handlers"

func setupRoutes(db *sqlx.DB, r chi.Router) {
	registryService := services.NewMetricRegistryService(db)
	handler := handlers.NewMetricRegistryHandler(registryService)
	
	// Register all metrics-registry routes
	handler.RegisterRoutes(r)
	
	// Available endpoints:
	// GET  /api/metrics-registry
	// GET  /api/metrics-registry/{metricID}
	// GET  /api/metrics-registry/{metricID}/history
	// POST /api/metrics-registry/refresh-atomic
	// POST /api/metrics-registry/{metricID}/compute-pop
	// POST /api/metrics-registry/{metricID}/compute-comparisons
	// POST /api/metrics-registry/{metricID}/detect-anomalies
	// POST /api/metrics-registry/{metricID}/promote-golden
	// GET  /api/metrics-registry/golden-path/readiness
}
```

---

## 6. Test Real-Time Atomic Lane

```bash
# Manually trigger atomic refresh
curl -X POST http://localhost:8080/api/metrics-registry/refresh-atomic \
  -H "Content-Type: application/json" \
  -d '{}'

# Response:
# {
#   "status": "queued",
#   "execution_id": "...",
#   "logs": [...]
# }

# Check execution history
curl http://localhost:8080/api/metrics-registry/{metricID}/history?limit=10
```

---

## 7. Test Batch PoP & Comparison Computation

```bash
# Trigger monthly PoP computation (e.g., for October 2024)
curl -X POST http://localhost:8080/api/metrics-registry/{metricID}/compute-pop \
  -H "Content-Type: application/json" \
  -d '{
    "period_start": "2024-10-01",
    "period_end": "2024-10-31"
  }'

# Compute comparison periods
curl -X POST http://localhost:8080/api/metrics-registry/{metricID}/compute-comparisons \
  -H "Content-Type: application/json" \
  -d '{}'

# Query results
psql -c "
  SELECT period_label, current_value, previous_period_value, 
         previous_period_percent_change, yoy_percent_change
  FROM public.metrics_comparison_periods
  WHERE metric_id = '...'
  ORDER BY period_label DESC
  LIMIT 12;
"
```

---

## 8. Test Anomaly Detection

```bash
# Trigger z-score anomaly detection
curl -X POST http://localhost:8080/api/metrics-registry/{metricID}/detect-anomalies \
  -H "Content-Type: application/json" \
  -d '{
    "zscore_threshold": 2.5,
    "window_days": 90,
    "min_data_points": 7
  }'

# Query detected anomalies
psql -c "
  SELECT id, metric_id, anomaly_type, severity, confidence, z_score, detected_at
  FROM public.pop_anomalies
  WHERE status = 'open'
  ORDER BY detected_at DESC
  LIMIT 20;
"
```

---

## 9. Monitor Golden Path Readiness

```bash
# Check status of all golden path metrics
curl http://localhost:8080/api/metrics-registry/golden-path/readiness

# Response:
# {
#   "count": 5,
#   "readiness": [
#     {
#       "metric_id": "...",
#       "name": "revenue",
#       "readiness_status": "ready",
#       "current_value": 1250000.50,
#       "last_refresh": "2024-11-01T02:30:00Z"
#     },
#     ...
#   ]
# }
```

---

## 10. Query Execution Logs

```sql
-- Recent execution activity across all lanes
SELECT 
  el.execution_id, el.metric_id, mr.name, el.lane, el.execution_type,
  el.status, el.completed_at, el.completeness_score, el.error_message
FROM semantic_layer.metric_execution_log el
JOIN semantic_layer.metric_registry mr ON el.metric_id = mr.metric_id
WHERE el.completed_at >= NOW() - INTERVAL '24 hours'
ORDER BY el.completed_at DESC;

-- Check for SLA violations
SELECT 
  sv.metric_id, mr.name, sv.violation_type, 
  sv.expected_threshold, sv.actual_value, sv.status
FROM public.sla_violations sv
JOIN semantic_layer.metric_registry mr ON sv.metric_id = mr.metric_id
WHERE sv.status = 'open';
```

---

## 11. Promote a Metric to Golden Path

```bash
curl -X POST http://localhost:8080/api/metrics-registry/{metricID}/promote-golden

# Response: {"status": "promoted_to_golden_path", "metric_id": "..."}
```

Once golden, the metric is:
- Subject to SLA enforcement (every 6 hours)
- Included in golden path readiness dashboard
- Prioritized for anomaly detection

---

## 12. Manual Job Execution (Backfill)

For ad-hoc backfills, directly call orchestrator methods:

```go
// In your admin/ops handler
func (h *OpsHandler) BackfillMetricPoP(w http.ResponseWriter, r *http.Request) {
	metricID, _ := uuid.Parse(r.URL.Query().Get("metric_id"))
	start := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 8, 31, 23, 59, 59, 0, time.UTC)
	
	log, err := h.registryService.ComputeMonthlyPoP(r.Context(), &metricID, &start, &end)
	// ON CONFLICT ... DO UPDATE ensures idempotent re-run
	
	json.NewEncoder(w).Encode(log)
}
```

---

## 13. View Registry Metadata

```bash
# List all metrics in a domain with their definitions
curl http://localhost:8080/api/metrics-registry?domain=finance

# Get full definition of a metric
curl http://localhost:8080/api/metrics-registry/{metricID}

# Returns:
# {
#   "metric_id": "...",
#   "name": "revenue",
#   "display_name": "Monthly Revenue",
#   "domain": "finance",
#   "category": "p&l",
#   "metric_type": "derived",
#   "granularity": ["month"],
#   "aggregation_function": "SUM",
#   "sla_freshness_hours": 24,
#   "sla_completeness_threshold": 95.0,
#   "refresh_schedule": "monthly",
#   "golden_path": true,
#   "status": "active"
# }
```

---

## 14. Common Scenarios

### Scenario A: New metric from DAX source
```sql
-- Register
INSERT INTO semantic_layer.metric_registry (...) VALUES 
  ('trading_pnl', 'Trading P&L', 'finance', 'pnl', 'atomic',
   'DAX: SUM([PlnAmount])', 'Excel', ARRAY['date'], 24, 98.0, 'daily', TRUE);

-- Will be automatically ingested by atomic_refresh lane (hourly)
```

### Scenario B: Backfill Q3 2024 PoP
```bash
curl -X POST /api/metrics-registry/{metricID}/compute-pop \
  -d '{
    "period_start": "2024-07-01",
    "period_end": "2024-09-30"
  }'

# Three upserts: July, August, September (idempotent by period_start/end)
```

### Scenario C: Alert on anomaly spike
```sql
-- Query for HIGH severity anomalies in last 24h
SELECT COUNT(*) as critical_anomalies
FROM public.pop_anomalies
WHERE detected_at >= NOW() - INTERVAL '24 hours'
  AND severity = 'high';

-- Trigger webhook or Slack notification if count > threshold
```

---

## Troubleshooting

### Orchestrator not executing jobs?
- Check logs: `orchestrator.GetStatus()`
- Verify schedulers started: `SELECT * FROM semantic_layer.metric_execution_log LIMIT 1`
- Ensure database connections are healthy

### Metrics not finalizing?
- Check SLA violations: `SELECT * FROM public.sla_violations WHERE status='open'`
- Verify data is fresh: `SELECT MAX(metric_time) FROM public.metrics`
- Confirm registry entry exists: `SELECT * FROM semantic_layer.metric_registry WHERE name = '...'`

### Anomaly detection not running?
- Check z-score computation: `SELECT * FROM public.pop_anomalies ORDER BY detected_at DESC LIMIT 1`
- Verify min_data_points met: `SELECT COUNT(*) FROM public.pop_computations WHERE metric_id = $1`
- Check threshold: adjust `p_zscore_threshold` if all scores are low

---

## Summary

| Component | Purpose | Trigger |
|-----------|---------|---------|
| `refresh_atomic_metrics()` | Ingest & validate daily metrics | Every 1 hour |
| `compute_monthly_pop()` | Calculate deltas, percent_change | 1st of month, 2 AM |
| `compute_comparison_periods()` | YoY, QoQ, PoP pre-compute | After PoP computation |
| `detect_zscore_anomalies()` | Flag statistical outliers | Daily at 3 AM |
| `golden_path_readiness` | SLA compliance dashboard | Every 6 hours |

All traces preserved in `metric_execution_log` for audit & replay.

---

**Next**: See `DUAL_PATH_ENGINE_GUIDE.md` for full architecture details.
