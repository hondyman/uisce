-- DDL for consolidated metrics table in public schema
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

-- Example indexes (create as needed)
CREATE INDEX IF NOT EXISTS metrics_idx_industry_time ON public.metrics (industry_id, metric_time DESC);
CREATE INDEX IF NOT EXISTS metrics_idx_industry_type ON public.metrics (industry_id, metric_type);
CREATE INDEX IF NOT EXISTS metrics_gin_tags ON public.metrics USING GIN (tags);
CREATE INDEX IF NOT EXISTS metrics_gin_details ON public.metrics USING GIN (details);
