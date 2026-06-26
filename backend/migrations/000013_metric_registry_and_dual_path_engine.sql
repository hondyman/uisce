-- =====================================================
-- Migration: Metric Registry & Dual-Path Calculation Engine
-- Date: 2025-11-01
-- Purpose: Implement canonical metric model, registry,
--          and orchestration for real-time + batch lanes
-- =====================================================

-- =====================================================
-- 1. METRIC REGISTRY (Single Source of Truth)
-- =====================================================

CREATE SCHEMA IF NOT EXISTS semantic_layer;

DROP TABLE IF EXISTS semantic_layer.metric_registry CASCADE;

CREATE TABLE IF NOT EXISTS semantic_layer.metric_registry (
  metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  
  -- Identity & semantics
  name TEXT UNIQUE NOT NULL,
  display_name TEXT NOT NULL,
  description TEXT,
  domain TEXT NOT NULL,
  category TEXT NOT NULL,
  metric_type TEXT NOT NULL CHECK (metric_type IN ('atomic', 'derived', 'composite')),
  
  -- Computation rules
  base_query TEXT,
  aggregation_function TEXT,
  granularity TEXT[] DEFAULT ARRAY['date'],
  value_column TEXT,
  date_column TEXT,
  
  -- Source lineage
  source_formula TEXT,  -- 'DAX/Excel formula', or upstream metric IDs
  source_system TEXT,   -- 'Workday', 'Excel', 'API', 'warehouse'
  
  -- Time alignment & periods
  comparison_periods JSONB DEFAULT '{"previous_period": false, "yoy": false, "qoq": false}'::JSONB,
  period_label_format TEXT DEFAULT 'YYYY-MM',  -- For PoP labels
  
  -- SLA & quality gates
  sla_freshness_hours INT DEFAULT 24,
  sla_completeness_threshold NUMERIC(5,2) DEFAULT 95.00,
  refresh_schedule TEXT DEFAULT 'daily',  -- 'hourly', 'daily', 'weekly', 'monthly', 'real-time'
  
  -- Governance
  owner_user_id TEXT,
  steward_group TEXT,
  golden_path BOOLEAN DEFAULT FALSE,
  status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'deprecated')),
  
  version INT DEFAULT 1,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  created_by TEXT,
  updated_by TEXT
);

CREATE INDEX idx_registry_domain ON semantic_layer.metric_registry(domain, category);
CREATE INDEX idx_registry_golden ON semantic_layer.metric_registry(golden_path) WHERE golden_path;
CREATE INDEX idx_registry_source ON semantic_layer.metric_registry(source_system);
CREATE INDEX idx_registry_status ON semantic_layer.metric_registry(status);

-- =====================================================
-- 2. EXECUTION METADATA & LINEAGE
-- =====================================================

CREATE TABLE IF NOT EXISTS semantic_layer.metric_execution_log (
  execution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  
  metric_id UUID NOT NULL REFERENCES semantic_layer.metric_registry(metric_id) ON DELETE CASCADE,
  lane TEXT NOT NULL CHECK (lane IN ('real-time', 'batch')),
  execution_type TEXT NOT NULL CHECK (execution_type IN ('refresh', 'backfill', 'recompute')),
  
  -- Period information for batch jobs
  period_start DATE,
  period_end DATE,
  period_label TEXT,
  
  -- Execution status
  status TEXT NOT NULL DEFAULT 'started' CHECK (status IN ('started', 'completed', 'failed', 'partial')),
  record_count INT,
  success_count INT,
  error_count INT,
  
  -- Quality metrics
  completeness_score NUMERIC(5,2),
  freshness_hours NUMERIC(10,2),
  
  -- Error tracking
  error_message TEXT,
  error_details JSONB,
  
  started_at TIMESTAMPTZ DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  duration_ms INT
);

CREATE INDEX idx_exec_log_metric ON semantic_layer.metric_execution_log(metric_id, completed_at DESC);
CREATE INDEX idx_exec_log_lane ON semantic_layer.metric_execution_log(lane, status);
CREATE INDEX idx_exec_log_period ON semantic_layer.metric_execution_log(period_start, period_end);

-- =====================================================
-- 3. REAL-TIME LANE: FINALIZED ATOMIC METRICS
-- =====================================================

CREATE TABLE IF NOT EXISTS public.metrics_finalized (
  id BIGSERIAL PRIMARY KEY,
  
  metric_id UUID NOT NULL REFERENCES semantic_layer.metric_registry(metric_id) ON DELETE RESTRICT,
  metric_name TEXT NOT NULL,
  as_of_date DATE NOT NULL,
  
  -- Value & metadata
  value DOUBLE PRECISION,
  previous_value DOUBLE PRECISION,
  
  -- Quality gates
  freshness_status TEXT DEFAULT 'unknown' CHECK (freshness_status IN ('fresh', 'stale', 'unknown')),
  meets_sla BOOLEAN DEFAULT FALSE,
  completeness_score NUMERIC(5,2),
  
  -- Lineage
  source_system TEXT,
  source_record_count INT,
  
  last_refresh TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_metrics_finalized_metric_date 
  ON public.metrics_finalized(metric_id, as_of_date);
CREATE INDEX idx_metrics_finalized_freshness 
  ON public.metrics_finalized(freshness_status, meets_sla);
CREATE INDEX idx_metrics_finalized_date 
  ON public.metrics_finalized(as_of_date DESC);

-- =====================================================
-- 4. BATCH LANE: COMPARISON PERIODS (YoY, QoQ, PoP)
-- =====================================================

CREATE TABLE IF NOT EXISTS public.metrics_comparison_periods (
  id BIGSERIAL PRIMARY KEY,
  
  metric_id UUID NOT NULL REFERENCES semantic_layer.metric_registry(metric_id) ON DELETE CASCADE,
  period_label TEXT NOT NULL,
  
  -- Current & previous
  current_value DOUBLE PRECISION,
  previous_period_value DOUBLE PRECISION,
  yoy_value DOUBLE PRECISION,
  qoq_value DOUBLE PRECISION,
  
  -- Deltas
  previous_period_delta DOUBLE PRECISION,
  previous_period_percent_change NUMERIC(10,4),
  yoy_delta DOUBLE PRECISION,
  yoy_percent_change NUMERIC(10,4),
  qoq_delta DOUBLE PRECISION,
  qoq_percent_change NUMERIC(10,4),
  
  -- Metadata
  record_count INT,
  computation_status TEXT DEFAULT 'success' CHECK (computation_status IN ('success', 'error', 'partial')),
  
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_comparison_periods_metric_label 
  ON public.metrics_comparison_periods(metric_id, period_label);
CREATE INDEX idx_comparison_periods_metric 
  ON public.metrics_comparison_periods(metric_id, period_label DESC);

-- =====================================================
-- 5. SLA VIOLATIONS & QUALITY TRACKING
-- =====================================================

CREATE TABLE IF NOT EXISTS public.sla_violations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  
  metric_id UUID NOT NULL REFERENCES semantic_layer.metric_registry(metric_id) ON DELETE CASCADE,
  violation_type TEXT NOT NULL CHECK (violation_type IN ('freshness', 'completeness', 'both')),
  
  -- Threshold details
  expected_threshold NUMERIC,
  actual_value NUMERIC,
  breach_amount NUMERIC,
  
  details JSONB,
  status TEXT DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'resolved')),
  
  detected_at TIMESTAMPTZ DEFAULT NOW(),
  acknowledged_at TIMESTAMPTZ,
  resolved_at TIMESTAMPTZ,
  
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sla_violations_metric ON public.sla_violations(metric_id, detected_at DESC);
CREATE INDEX idx_sla_violations_status ON public.sla_violations(status) WHERE status = 'open';

-- =====================================================
-- 6. HELPER VIEWS
-- =====================================================

-- View: Registry with latest execution stats
CREATE OR REPLACE VIEW semantic_layer.metric_registry_with_stats AS
SELECT
  r.*,
  el.status as last_execution_status,
  el.completed_at as last_execution_time,
  el.completeness_score as last_completeness,
  el.error_count as last_error_count,
  CASE 
    WHEN el.completed_at IS NULL THEN 'never_executed'
    WHEN (NOW() - el.completed_at) <= INTERVAL '1 day' THEN 'fresh'
    WHEN (NOW() - el.completed_at) <= INTERVAL '7 days' THEN 'stale'
    ELSE 'very_stale'
  END as execution_freshness
FROM semantic_layer.metric_registry r
LEFT JOIN semantic_layer.metric_execution_log el ON r.metric_id = el.metric_id
  AND el.execution_id = (
    SELECT execution_id FROM semantic_layer.metric_execution_log
    WHERE metric_id = r.metric_id
    ORDER BY completed_at DESC NULLS LAST
    LIMIT 1
  );

-- View: Golden path metrics readiness
CREATE OR REPLACE VIEW public.golden_path_readiness AS
SELECT
  r.metric_id,
  r.name,
  r.display_name,
  r.domain,
  CASE
    WHEN r.golden_path = FALSE THEN 'not_golden'
    WHEN sv.id IS NOT NULL THEN 'sla_violation'
    WHEN mf.meets_sla = FALSE THEN 'quality_gate_failed'
    WHEN (NOW() - mf.last_refresh) > INTERVAL '24 hours' THEN 'stale_data'
    ELSE 'ready'
  END as readiness_status,
  mf.value as current_value,
  mf.as_of_date as last_data_date,
  mf.last_refresh,
  sv.violation_type,
  sv.status as violation_status
FROM semantic_layer.metric_registry r
LEFT JOIN public.metrics_finalized mf ON r.metric_id = mf.metric_id
  AND mf.id = (
    SELECT id FROM public.metrics_finalized
    WHERE metric_id = r.metric_id
    ORDER BY as_of_date DESC LIMIT 1
  )
LEFT JOIN public.sla_violations sv ON r.metric_id = sv.metric_id
  AND sv.status = 'open'
WHERE r.golden_path = TRUE;

-- =====================================================
-- 7. UPDATE TRIGGERS
-- =====================================================

CREATE OR REPLACE FUNCTION update_metric_registry_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_metric_registry_update
BEFORE UPDATE ON semantic_layer.metric_registry
FOR EACH ROW
EXECUTE FUNCTION update_metric_registry_timestamp();

-- =====================================================
-- 8. SAMPLE DATA: Backfill registry from catalog
-- =====================================================

-- Migrate existing pop_metrics into registry
INSERT INTO semantic_layer.metric_registry (
  name, display_name, description, domain, category, metric_type,
  base_query, aggregation_function, granularity, value_column, date_column,
  source_system, comparison_periods, period_label_format,
  sla_freshness_hours, sla_completeness_threshold, refresh_schedule,
  owner_user_id, steward_group, golden_path, created_by
)
SELECT
  m.name,
  m.display_name,
  m.description,
  m.domain,
  m.category,
  'derived',  -- pop_metrics are derived
  m.base_query,
  m.aggregation_function,
  ARRAY['month'],
  m.value_column,
  m.date_column,
  m.data_source,
  m.comparison_periods,
  'YYYY-MM',
  m.sla_freshness_hours,
  m.sla_completeness_threshold,
  'monthly',
  m.owner_user_id,
  m.steward_group,
  m.golden_path,
  m.created_by
FROM public.pop_metrics m
WHERE NOT EXISTS (
  SELECT 1 FROM semantic_layer.metric_registry
  WHERE name = m.name
)
ON CONFLICT (name) DO NOTHING;

-- =====================================================
-- 9. GRANTS (if multi-tenant)
-- =====================================================

GRANT USAGE ON SCHEMA semantic_layer TO PUBLIC;
GRANT SELECT ON semantic_layer.metric_registry TO PUBLIC;
GRANT SELECT ON semantic_layer.metric_registry_with_stats TO PUBLIC;
GRANT SELECT ON public.golden_path_readiness TO PUBLIC;

GRANT INSERT, UPDATE ON semantic_layer.metric_execution_log TO PUBLIC;
GRANT INSERT, UPDATE ON public.metrics_finalized TO PUBLIC;
GRANT INSERT, UPDATE ON public.metrics_comparison_periods TO PUBLIC;
GRANT INSERT ON public.sla_violations TO PUBLIC;
