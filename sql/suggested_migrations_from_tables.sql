-- Suggested migration SQL: move metric-like tables into public.metrics
-- Review and adjust column names and UUIDs before executing.
-- Generated: 2025-11-01

-- 1) DDL for target table (same as sql/create_public_metrics.sql)
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE IF NOT EXISTS public.metrics (
  id bigserial PRIMARY KEY,
  industry_id uuid NOT NULL,
  metric_type text NOT NULL,
  metric_time timestamptz NOT NULL,
  value double precision,
  tags jsonb DEFAULT '{}'::jsonb,
  details jsonb DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS metrics_idx_industry_time ON public.metrics (industry_id, metric_time DESC);
CREATE INDEX IF NOT EXISTS metrics_idx_industry_type ON public.metrics (industry_id, metric_type);
CREATE INDEX IF NOT EXISTS metrics_gin_tags ON public.metrics USING GIN (tags);
CREATE INDEX IF NOT EXISTS metrics_gin_details ON public.metrics USING GIN (details);

-- 2) Per-table INSERTs (generated from introspected schema, Nov 1 2025)
-- Actual column names verified and safe expressions used for numeric coercion

-- public.performance_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-111111111111
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-111111111111'::uuid AS industry_id,
  m.metric_name::text AS metric_type,
  COALESCE(m.collected_at, now()) AS metric_time,
  m.metric_value::double precision AS value,
  COALESCE(m.labels, '{}'::jsonb) AS tags,
  to_jsonb(m) - ARRAY['id','collected_at'] AS details,
  now() AS created_at,
  now() AS updated_at
FROM public.performance_metrics m;

-- public.pop_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-222222222222
-- Metric metadata table; stores metric definitions, not values. Store all fields in details.
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-222222222222'::uuid AS industry_id,
  m.metric_type AS metric_type,
  COALESCE(m.updated_at, m.created_at, now()) AS metric_time,
  NULL::double precision AS value,
  '{}'::jsonb AS tags,
  to_jsonb(m) - ARRAY['id','created_at','updated_at'] AS details,
  COALESCE(m.created_at, now()) AS created_at,
  COALESCE(m.updated_at, now()) AS updated_at
FROM public.pop_metrics m;

-- public.pop_computations -> f2a3b1c4-7a6d-4b2e-9c3d-333333333333
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-333333333333'::uuid AS industry_id,
  COALESCE(m.period_label, m.granularity, 'pop_computations') AS metric_type,
  COALESCE(m.period_end::timestamptz, m.last_updated, now()) AS metric_time,
  COALESCE(m.current_value::double precision, 0.0)::double precision AS value,
  '{}'::jsonb AS tags,
  to_jsonb(m) - ARRAY['id','period_start','period_end','last_updated'] AS details,
  now() AS created_at,
  now() AS updated_at
FROM public.pop_computations m;

-- public.pop_anomalies -> f2a3b1c4-7a6d-4b2e-9c3d-444444444444
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-444444444444'::uuid AS industry_id,
  m.anomaly_type AS metric_type,
  COALESCE(m.detected_at, now()) AS metric_time,
  COALESCE(m.confidence::double precision, NULL) AS value,
  jsonb_build_object('severity', m.severity) AS tags,
  to_jsonb(m) - ARRAY['id','detected_at'] AS details,
  now() AS created_at,
  now() AS updated_at
FROM public.pop_anomalies m;

-- public.integration_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-555555555555
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-555555555555'::uuid AS industry_id,
  'integration_metrics' AS metric_type,
  m.timestamp AS metric_time,
  m.requests_count::double precision AS value,
  '{}'::jsonb AS tags,
  to_jsonb(m) - ARRAY['id','timestamp','requests_count','tenant_id','integration_id'] AS details,
  now() AS created_at,
  now() AS updated_at
FROM public.integration_metrics m;

-- public.api_metrics_current_month -> f2a3b1c4-7a6d-4b2e-9c3d-666666666666
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-666666666666'::uuid AS industry_id,
  'api_metrics_current_month' AS metric_type,
  m.timestamp AS metric_time,
  COALESCE(m.response_time::double precision, NULL) AS value,
  jsonb_build_object('status_code', m.status_code) AS tags,
  to_jsonb(m) - ARRAY['id','timestamp','response_time','tenant_id','status_code'] AS details,
  now() AS created_at,
  now() AS updated_at
FROM public.api_metrics_current_month m;

-- public.api_metrics_next_month -> f2a3b1c4-7a6d-4b2e-9c3d-777777777777
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-777777777777'::uuid AS industry_id,
  'api_metrics_next_month' AS metric_type,
  m.timestamp AS metric_time,
  COALESCE(m.response_time::double precision, NULL) AS value,
  jsonb_build_object('status_code', m.status_code) AS tags,
  to_jsonb(m) - ARRAY['id','timestamp','response_time','tenant_id','status_code'] AS details,
  now() AS created_at,
  now() AS updated_at
FROM public.api_metrics_next_month m;

-- semantic_layer.preaggregated_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-888888888888
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-888888888888'::uuid AS industry_id,
  m.name::text AS metric_type,
  m.last_refresh AS metric_time,
  m.value::double precision AS value,
  COALESCE(m.grain, '{}'::jsonb) AS tags,
  to_jsonb(m) - ARRAY['id','name','value','last_refresh','grain','created_at','updated_at'] AS details,
  COALESCE(m.created_at, now()) AS created_at,
  COALESCE(m.updated_at, now()) AS updated_at
FROM semantic_layer.preaggregated_metrics m;

-- public.private_markets_fund_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-999999999999
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-999999999999'::uuid AS industry_id,
  'private_markets_fund_metrics' AS metric_type,
  m.as_of_date::timestamptz AS metric_time,
  COALESCE(m.tvpi::double precision, NULL) AS value,
  jsonb_build_object('fund_id', m.fund_id) AS tags,
  to_jsonb(m) - ARRAY['id','as_of_date','tvpi','fund_id','created_at','updated_at'] AS details,
  COALESCE(m.created_at, now()) AS created_at,
  COALESCE(m.updated_at, now()) AS updated_at
FROM public.private_markets_fund_metrics m;

-- public.prepared_statement_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-aaaaaaaaaaaa
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-aaaaaaaaaaaa'::uuid AS industry_id,
  'prepared_statement_metrics' AS metric_type,
  COALESCE(m.last_executed, now()) AS metric_time,
  COALESCE(m.execution_count::double precision, 0.0) AS value,
  '{}'::jsonb AS tags,
  to_jsonb(m) - ARRAY['id','last_executed','execution_count','created_at'] AS details,
  COALESCE(m.created_at, now()) AS created_at,
  now() AS updated_at
FROM public.prepared_statement_metrics m;

-- public.bp_branch_metrics -> f2a3b1c4-7a6d-4b2e-9c3d-bbbbbbbbbbbb
INSERT INTO public.metrics (industry_id, metric_type, metric_time, value, tags, details, created_at, updated_at)
SELECT
  'f2a3b1c4-7a6d-4b2e-9c3d-bbbbbbbbbbbb'::uuid AS industry_id,
  'bp_branch_metrics' AS metric_type,
  COALESCE(m.created_at::timestamptz, now()) AS metric_time,
  COALESCE(m.total_executions::double precision, 0.0) AS value,
  jsonb_build_object('branch_label', m.branch_label, 'completion_rate', m.completion_rate) AS tags,
  to_jsonb(m) - ARRAY['id','tenant_id','step_id','branch_label','total_executions','created_at','updated_at'] AS details,
  COALESCE(m.created_at::timestamptz, now()) AS created_at,
  COALESCE(m.updated_at::timestamptz, now()) AS updated_at
FROM public.bp_branch_metrics m;

-- End of suggested migrations. Review fully before running. Back up your DB first (pg_dump).
