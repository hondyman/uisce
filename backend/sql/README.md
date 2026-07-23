# Phase 3.21: Advanced Feature Engineering - Schema & SQL Documentation

This directory contains the complete PostgreSQL schema, migrations, and utilities for Phase 3.21 feature engineering infrastructure.

## Files Overview

### Core Schema & Setup

- **`phase_3_21_schema.sql`** (1,100+ lines)
  - 10 core tables with comprehensive indexes
  - Helper functions for lineage, health checks, ancestor retrieval
  - Row-level triggers for timestamp management
  - Views for active features, pending approvals, failing tests, active drifts
  - Materialized views for SLO tracking and top features
  - Complete permissions and GRANT statements
  - Production-ready constraints and documentation

- **`init_schema.sh`** (Bash script)
  - One-command schema initialization
  - Verifies PostgreSQL connectivity
  - Prints post-init summary with table/view/index counts
  - Handles all environment variables (PGHOST, PGPORT, PGUSER, PGDATABASE)

### Migrations

- **`migrations/001_phase_3_21_initial_schema.sql`**
  - Initial migration for schema versioning
  - Tracks schema version and application timestamps
  - Reference implementation for future migrations

### Sample Data

- **`sample_data.sql`** (500+ lines)
  - 5 realistic blockchain-domain features
  - Watermarks, drift metrics, quality checks
  - SHAP importance scores from real ML domain
  - Change log with governance events
  - Test cases, lineage, and computation logs
  - Full end-to-end sample dataset for testing

## Schema Architecture

### 10 Core Tables

```
feature_catalog           — Master registry of all features
├─ feature_watermarks    — Incremental materialization tracking
├─ feature_drift_metrics — Distribution shift detection results
├─ feature_quality_checks — Data quality assertions
├─ feature_importance    — SHAP/importance scores and trends
├─ feature_change_log    — Governance audit trail
├─ feature_test_cases    — Unit/integration tests per feature
├─ feature_lineage       — Upstream/downstream dependencies
├─ feature_computations  — Job execution metrics and SLOs
└─ schema_migrations     — Migration version tracking
```

### Key Design Principles

1. **Region & Tenant Aware**: All tables include `tenant_id` and `region` columns for multi-tenant deployments
2. **JSONB Flexibility**: `feature_catalog.properties` stores flexible metadata (owner, feature_type, aggregations, drift config, test cases)
3. **Audit Trail**: Complete changelog with approvals, deployments, rollbacks
4. **Performance First**: Strategic indexes on query patterns (tenant/region, timestamps, status)
5. **Partitioning Ready**: `feature_drift_metrics` designed for range partitioning by timestamp
6. **Observability**: Materialized views for SLO tracking, computation cost analysis, feature rankings

## Getting Started

### 1. Initialize Schema

```bash
cd backend/sql
chmod +x init_schema.sh
PGHOST=localhost PGPORT=5432 PGUSER=postgres PGPASSWORD=secret PGDATABASE=semlayer ./init_schema.sh
```

### 2. Load Sample Data (Optional, for testing)

```bash
psql -h localhost -U postgres -d semlayer -f sample_data.sql
```

### 3. Verify Installation

```bash
# Check tables
psql -h localhost -U postgres -d semlayer -c "
  SELECT tablename FROM pg_tables 
  WHERE schemaname = 'public' AND tablename LIKE 'feature%'
  ORDER BY tablename;"

# Check views
psql -h localhost -U postgres -d semlayer -c "
  SELECT viewname FROM pg_views 
  WHERE schemaname = 'public' AND viewname LIKE 'feature%' OR viewname LIKE '%_active'
  ORDER BY viewname;"

# Check sample data (if loaded)
psql -h localhost -U postgres -d semlayer -c "
  SELECT COUNT(*) as features FROM feature_catalog;
  SELECT COUNT(*) as drifts FROM feature_drift_metrics;
  SELECT COUNT(*) as importance FROM feature_importance;"
```

## Table Descriptions

### feature_catalog
Master registry of all features with flexible JSONB metadata.

**Key Fields:**
- `feature_id` (PK): e.g., `feature:orders.revenue_v1`
- `properties` (JSONB): Contains feature_type, expression, aggregation, window, materialization_policy, drift_config, test_cases
- `is_core`: Boolean flag for SLA-critical features
- `owner`: Responsible team/person
- `version`: Semantic versioning
- `deprecated_at`: Soft-delete support

**Indexes:**
- owner, namespace, is_core, tenant/region, created_at, properties (GIN on feature_type), deprecated status, lifecycle

**Views:**
- `feature_catalog_active`: Non-deprecated features only

---

### feature_watermarks
Tracks last processed timestamp per feature for incremental materialization.

**Key Fields:**
- `feature_id` (PK, FK → feature_catalog)
- `last_processed`: Watermark timestamp
- `materialization_lag_seconds`: SLO tracking
- `watermark_age_seconds`: Generated column = time since last process

**Purpose:** Enables exactly-once semantics in incremental pipelines

---

### feature_drift_metrics
Distribution shift detection results (KS test, PSI, Chi-square, classifier-based).

**Key Fields:**
- `drift_id` (PK, UUID)
- `feature_id` (FK → feature_catalog)
- `method`: ks | psi | chi2 | classifier
- `score`: Statistical test result [0,1] or [0,∞)
- `is_drifted`: Boolean flag (score > threshold)
- `baseline_window_*`: Historical reference window
- `eval_window_*`: Recent evaluation window
- `alert_sent`: Alerting status

**Indexes:**
- feature_id + timestamp (for recent drifts)
- is_drifted + timestamp (for active incidents)
- alert_sent (for alerting automation)

**Materialized Views:**
- `active_drifts`: Recent drifts with recency rank

**Partitioning Strategy (Manual):**
- Range partition by `recorded_at` (e.g., monthly) for rolling-window performance

---

### feature_quality_checks
Data quality assertions (null rate, cardinality, type expectations, schema drift).

**Key Fields:**
- `check_id` (PK, UUID)
- `feature_id` (FK)
- `check_name`: e.g., "null_rate_too_high"
- `check_type`: null_rate | cardinality | type | range | custom
- `threshold_type`: max | min | range | exact
- `passed`: Boolean result
- `observed_value`: Actual metric

**Triggers Materialization Failures:** If feature quality checks fail, materialization is blocked.

---

### feature_importance
SHAP values, permutation importance, and stability trends.

**Key Fields:**
- `importance_id` (PK, UUID)
- `feature_id` (FK)
- `model_id`: Cross-reference to model registry
- `mean_abs_shap`: Aggregated SHAP magnitude [0,∞)
- `shap_values[]`: Raw sampled SHAP values for distribution analysis
- `permutation_importance`: Drop-column importance
- `stability_score`: 1 - variance(importance over time) [0,1]
- `importance_trend`: Linear trend over recent runs
- `importance_percentile`: Feature rank [0,100]

**Materialized Views:**
- `top_features_by_model`: Nightly top K features per model

---

### feature_change_log
Immutable governance audit trail: approvals, deployments, rollbacks.

**Key Fields:**
- `change_id` (PK, UUID)
- `change_type`: created | updated | approved | deployed | deprecated | rollback
- `pr_url`: GitHub PR link
- `requested_by`: User or automation
- `approved_by`: Approver
- `approval_status`: PENDING | APPROVED | REJECTED
- `deployed_at`: Deployment timestamp
- `old_properties` / `new_properties`: Before/after snapshots

**Views:**
- `pending_approvals`: Features awaiting approval

---

### feature_test_cases
Unit/integration test definitions per feature.

**Key Fields:**
- `test_id` (PK, UUID)
- `feature_id` (FK)
- `test_name`: Human-readable name
- `test_type`: unit | integration | regression | property
- `sql_assertion`: SQL query returning bool or numeric value
- `expected_result`: Expected outcome (JSONB)
- `tolerance`: For numeric comparisons
- `critical`: Boolean (must pass pre-deployment)
- `last_run_*`: Execution history

**Views:**
- `failing_tests`: Recent test failures

---

### feature_lineage
Upstream (source tables, features) and downstream (models, dashboards) dependencies.

**Key Fields:**
- `lineage_id` (PK, UUID)
- `source_feature_id`: Upstream feature (nullable)
- `source_table`: Schema.table if upstream is external
- `target_feature_id` (FK): Downstream feature
- `lineage_type`: feature_to_feature | table_to_feature | feature_to_model
- `contains_pii`: Sensitive data flag

**Recursive View:**
- `feature_lineage_ancestors`: All upstream dependencies (recursive)

---

### feature_computations
Execution logs for materialization, drift, importance jobs.

**Key Fields:**
- `computation_id` (PK, UUID)
- `feature_id` (FK)
- `job_type`: materialization | drift | importance
- `job_id`: Spark/Temporal run ID
- `status`: RUNNING | SUCCESS | FAILED | CANCELLED
- `started_at`, `completed_at`: Duration tracking
- `compute_cost_usd`: Resource cost
- `rows_processed`, `bytes_written`: Data volume
- `success_rate`: For batch jobs [0,1]

**Materialized Views:**
- `computation_slos`: SLO metrics (success rate, p95 latency, avg cost) by feature and job type

---

## Helper Functions

### get_feature_health(feature_id TEXT)
Returns comprehensive health report: materialization lag, drift count, quality failures, test failures.

```sql
SELECT * FROM get_feature_health('feature:orders.revenue_v1');
```

### get_feature_ancestors(feature_id_input TEXT)
Recursively retrieves all upstream feature dependencies.

```sql
SELECT * FROM get_feature_ancestors('feature:orders.revenue_v1');
```

## Triggers

### update_feature_catalog_timestamp
Automatically updates `updated_at` on any catalog change.

### update_test_cases_timestamp
Automatically updates `updated_at` on test case modifications.

## Performance Tuning

### Index Strategy
- **Hot paths:** feature_id + timestamp (drift, importance, computations)
- **Filtering paths:** tenant_id + region + timestamp (multi-tenant fan-out)
- **Status paths:** Partial indexes on status columns (active drifts, failed tests, pending approvals)
- **JSONB paths:** GIN indexes on properties (feature_type, lifecycle)

### Partitioning Strategy (Recommended)
Apply range partitions for `feature_drift_metrics` and `feature_computations`:

```sql
-- Create monthly partitions for drift_metrics
CREATE TABLE feature_drift_metrics_202602 PARTITION OF feature_drift_metrics
FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE feature_drift_metrics_202603 PARTITION OF feature_drift_metrics
FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

### Query Optimization Tips
- Always filter by `tenant_id` and `region` for multi-tenant systems
- Use `recorded_at DESC` for recent-first queries
- Leverage materialized views for aggregations (`top_features_by_model`, `active_drifts`, `computation_slos`)
- Update materialized views on Temporal cron schedules (e.g., nightly for `top_features_by_model`)

## Integration Points

### With Temporal Workflows (Phase 3.21 Package A)
- FeatureMaterializationWorkflow reads/updates `feature_watermarks`
- DriftDetectionWorkflow inserts to `feature_drift_metrics`
- FeatureImportanceWorkflow inserts to `feature_importance`
- FeatureDiscoveryWorkflow inserts draft features to `feature_catalog`

### With Go Feature API (Upcoming)
- RESTful endpoints for feature CRUD
- Approval workflows updating `feature_change_log`
- Health checks querying `get_feature_health()`

### With Python Drift/Importance Services (Phase 3.21 Packages B, C)
- Write drift results to `feature_drift_metrics`
- Write importance scores to `feature_importance`
- Log computation metadata to `feature_computations`

## Monitoring & SLOs

### Key Metrics to Monitor
- `materialization_lag_seconds` (feature_watermarks) — Target: <1h for hourly features
- `is_drifted` count in `active_drifts` — Track active incidents
- `quality_check_failures` — Catch data regressions early
- `importance_trend` stability — Flag degraded features
- `success_rate` from `computation_slos` — Target: >99%
- `compute_cost_usd` — Budget tracking

### Recommended Alerts
- Feature freshness > SLA (e.g., >2x expected lag)
- Drift metric exceeds threshold for core features
- >10 active drifts across catalog
- Test failure count > threshold
- Materialization cost spike (>2x rolling average)

## Migration & Rollout

### Phase 1: Schema Initialization (Week 0)
```bash
./init_schema.sh
psql -f sample_data.sql  # Optional, for dev/staging
```

### Phase 2: Service Integration (Weeks 1–2)
- Temporal workflows start writing to schema
- Drift detection service writes `feature_drift_metrics`
- Importance pipeline writes `feature_importance`

### Phase 3: Feature Adoption (Weeks 3–4)
- Top 500 features populated via discovery pipeline
- Materialized views updated nightly
- Alerts configured and validated
- CI/CD gating integrated

## Troubleshooting

### High Query Latency
- Check index usage: `EXPLAIN ANALYZE` on slow queries
- Apply recommended partitioning strategy
- Update materialized views more frequently (adjust Temporal cron)

### Missing Data
- Verify Temporal workflows are running
- Check feature_computations for job failures
- Validate service connectivity to PostgreSQL

### Drift Alerts Spamming
- Tune `drift_config.threshold` in feature_catalog properties
- Increase `baseline_window` for stability
- Filter alerts by `is_core = true` if needed

## Next Steps

- Create Grafana dashboards querying these tables (Phase 3.21 Package F)
- Implement drift detection service (Phase 3.21 Package B)
- Build importance pipeline (Phase 3.21 Package C)
- Add CI/CD tests (Phase 3.21 Package G)

---

**Total Schema Size:** 10 tables, 35+ indexes, 10 views, 2 materialized views, 2 helper functions, 3 triggers.  
**Production Ready:** Yes. All constraints, permissions, and indexes optimized for enterprise use.
