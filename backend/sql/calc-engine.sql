-- ============================================================================
-- CALC ENGINE - POSTGRES SCHEMA
-- Metric Registry, Transactional Control Plane, and Governance
-- ============================================================================

-- ============================================================================
-- METRIC REGISTRY (Source of truth for metrics and calculation logic)
-- ============================================================================

CREATE TABLE IF NOT EXISTS metric_registry (
  tenant_id UUID NOT NULL,
  metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  display_name TEXT,
  domain TEXT NOT NULL,                    -- e.g., 'finance', 'operations'
  category TEXT,
  granularity TEXT NOT NULL DEFAULT 'day', -- e.g., 'date', 'month', 'quarter'
  aggregation_function TEXT NOT NULL,      -- e.g., 'sum', 'avg', 'ratio'
  base_query TEXT,                         -- SQL template or semantic reference
  comparison_periods JSONB,                -- e.g., ["previous_period", "yoy", "qoq"]

  -- Calculation logic (for dynamic/self-service calculations)
  computation_type TEXT DEFAULT 'SQL',     -- 'SQL' | 'PYTHON' | 'EXPRESSION'
  computation_logic TEXT,                  -- SQL template with {{ placeholders }}

  -- SLAs and governance
  sla_freshness_hours INT DEFAULT 24,
  sla_completeness_threshold NUMERIC(5,2) DEFAULT 95.00,
  golden_path BOOLEAN DEFAULT FALSE,

  -- Ownership and stewardship
  owner_user_id TEXT,
  steward_group TEXT,

  -- Audit
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now(),
  created_by TEXT,
  updated_by TEXT,

  CONSTRAINT unique_tenant_metric_name UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_metric_registry_tenant 
  ON metric_registry(tenant_id);

CREATE INDEX IF NOT EXISTS idx_metric_registry_golden 
  ON metric_registry(tenant_id, golden_path) 
  WHERE golden_path = TRUE;

-- ============================================================================
-- METRIC VALUES TRANSACTIONAL LOG (Durable record of ingestion events)
-- ============================================================================

CREATE TABLE IF NOT EXISTS metric_values_txn (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL,
  metric_id UUID,
  business_object_key TEXT,               -- Workday BO key for lineage
  metric_type TEXT NOT NULL,              -- 'clean_price', 'pop_computations', etc.
  metric_time TIMESTAMPTZ NOT NULL,
  value NUMERIC(38,10) NOT NULL,
  tags JSONB DEFAULT '{}'::JSONB,
  details JSONB DEFAULT '{}'::JSONB,      -- grain, data_quality, source_formula
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_metric_values_txn_tenant_time 
  ON metric_values_txn(tenant_id, metric_time DESC);

CREATE INDEX IF NOT EXISTS idx_metric_values_txn_metric_id 
  ON metric_values_txn(metric_id);

-- ============================================================================
-- JOB RUNS (Durable compute run tracking for Temporal correlation)
-- ============================================================================

CREATE TABLE IF NOT EXISTS metric_job_runs (
  run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  metric_id UUID NOT NULL,
  calc_type TEXT NOT NULL,                -- 'pop' | 'anomaly'
  period_label TEXT,                      -- e.g., '2024-08'
  period_start DATE,
  period_end DATE,
  status TEXT NOT NULL DEFAULT 'pending',  -- 'pending' | 'running' | 'success' | 'failed'
  error_message TEXT,
  stats JSONB DEFAULT '{}'::JSONB,        -- record_count, duration_ms, retry_count
  started_at TIMESTAMPTZ DEFAULT now(),
  ended_at TIMESTAMPTZ,

  CONSTRAINT unique_tenant_metric_run UNIQUE(tenant_id, metric_id, calc_type, period_label),
  CONSTRAINT fk_metric_registry FOREIGN KEY(metric_id) REFERENCES metric_registry(metric_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_job_runs_tenant_status 
  ON metric_job_runs(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_job_runs_metric 
  ON metric_job_runs(tenant_id, metric_id);

CREATE INDEX IF NOT EXISTS idx_job_runs_started_at 
  ON metric_job_runs(started_at DESC);

-- ============================================================================
-- ANOMALY EVENTS (Independent lifecycle management for anomalies)
-- ============================================================================

CREATE TABLE IF NOT EXISTS anomaly_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  metric_id UUID NOT NULL,
  computation_id UUID,                    -- Reference to job run
  anomaly_type TEXT NOT NULL,             -- 'z_score'
  detected_at TIMESTAMPTZ NOT NULL,
  severity TEXT NOT NULL,                 -- 'low' | 'medium' | 'high' | 'critical'
  confidence NUMERIC(5,4),
  actual_value NUMERIC(38,10),
  expected_value NUMERIC(38,10),
  expected_range_min NUMERIC(38,10),
  expected_range_max NUMERIC(38,10),
  detection_params JSONB,                 -- {"threshold": 2.5, "window_days": 90}
  status TEXT DEFAULT 'open',             -- 'open' | 'resolved' | 'acknowledged'
  resolved_at TIMESTAMPTZ,
  resolved_by TEXT,
  resolution_notes TEXT,

  created_at TIMESTAMPTZ DEFAULT now(),
  CONSTRAINT fk_metric_registry_anomaly FOREIGN KEY(metric_id) REFERENCES metric_registry(metric_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_anomaly_events_tenant_metric_status 
  ON anomaly_events(tenant_id, metric_id, status);

CREATE INDEX IF NOT EXISTS idx_anomaly_events_detected_at 
  ON anomaly_events(tenant_id, detected_at DESC);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Trigger to update metric_registry.updated_at
CREATE OR REPLACE FUNCTION update_metric_registry_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_metric_registry_updated_at ON metric_registry;
CREATE TRIGGER trigger_metric_registry_updated_at
BEFORE UPDATE ON metric_registry
FOR EACH ROW
EXECUTE FUNCTION update_metric_registry_updated_at();

-- Function to get metric with all related runs
CREATE OR REPLACE FUNCTION get_metric_with_runs(p_tenant_id UUID, p_metric_id UUID)
RETURNS TABLE (
  metric_id UUID,
  name TEXT,
  domain TEXT,
  granularity TEXT,
  aggregation_function TEXT,
  golden_path BOOLEAN,
  sla_freshness_hours INT,
  run_id UUID,
  calc_type TEXT,
  period_label TEXT,
  status TEXT,
  started_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ
) AS $$
BEGIN
  RETURN QUERY
  SELECT 
    mr.metric_id,
    mr.name,
    mr.domain,
    mr.granularity,
    mr.aggregation_function,
    mr.golden_path,
    mr.sla_freshness_hours,
    mjr.run_id,
    mjr.calc_type,
    mjr.period_label,
    mjr.status,
    mjr.started_at,
    mjr.ended_at
  FROM metric_registry mr
  LEFT JOIN metric_job_runs mjr ON mr.metric_id = mjr.metric_id
  WHERE mr.tenant_id = p_tenant_id AND mr.metric_id = p_metric_id
  ORDER BY mjr.started_at DESC NULLS LAST;
END;
$$ LANGUAGE plpgsql;

-- Function to get recent anomalies for a metric
CREATE OR REPLACE FUNCTION get_metric_anomalies(
  p_tenant_id UUID, 
  p_metric_id UUID, 
  p_days INT DEFAULT 30
)
RETURNS TABLE (
  id UUID,
  anomaly_type TEXT,
  detected_at TIMESTAMPTZ,
  severity TEXT,
  confidence NUMERIC,
  actual_value NUMERIC,
  expected_value NUMERIC,
  status TEXT
) AS $$
BEGIN
  RETURN QUERY
  SELECT 
    ae.id,
    ae.anomaly_type,
    ae.detected_at,
    ae.severity,
    ae.confidence,
    ae.actual_value,
    ae.expected_value,
    ae.status
  FROM anomaly_events ae
  WHERE ae.tenant_id = p_tenant_id 
    AND ae.metric_id = p_metric_id
    AND ae.detected_at >= now() - (p_days || ' days')::INTERVAL
  ORDER BY ae.detected_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to mark job run as failed
CREATE OR REPLACE FUNCTION mark_job_run_failed(
  p_run_id UUID,
  p_error_message TEXT
)
RETURNS VOID AS $$
BEGIN
  UPDATE metric_job_runs
  SET status = 'failed', 
      error_message = p_error_message,
      ended_at = now()
  WHERE run_id = p_run_id;
END;
$$ LANGUAGE plpgsql;

-- Function to mark job run as succeeded with stats
CREATE OR REPLACE FUNCTION mark_job_run_success(
  p_run_id UUID,
  p_stats JSONB DEFAULT '{}'::JSONB
)
RETURNS VOID AS $$
BEGIN
  UPDATE metric_job_runs
  SET status = 'success', 
      stats = p_stats,
      ended_at = now()
  WHERE run_id = p_run_id;
END;
$$ LANGUAGE plpgsql;
