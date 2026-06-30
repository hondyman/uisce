-- Fix tenant_chart constraint for ON CONFLICT support
ALTER TABLE IF EXISTS public.tenant_chart DROP CONSTRAINT IF EXISTS tenant_chart_unique;
ALTER TABLE IF EXISTS public.tenant_chart ADD CONSTRAINT tenant_chart_unique UNIQUE (tenant_datasource_id, chart_name);
