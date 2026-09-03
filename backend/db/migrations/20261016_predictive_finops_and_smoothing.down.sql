-- Migration: 20261016_predictive_finops_and_smoothing.down.sql
-- Rollback for Predictive FinOps, Demand Forecasting & Prewarm Coordinator

DROP TABLE IF EXISTS finops.prewarm_execution_ledger CASCADE;
DROP TABLE IF EXISTS finops.workload_smoothing_policies CASCADE;
DROP TABLE IF EXISTS finops.compute_demand_forecasts CASCADE;

DROP INDEX IF EXISTS audit.idx_aqel_tenant_created;

ALTER TABLE audit.analytical_query_execution_logs
    DROP COLUMN IF EXISTS cpu_duration_ms,
    DROP COLUMN IF EXISTS attributed_cost_usd,
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS bo_id,
    DROP COLUMN IF EXISTS session_id;
