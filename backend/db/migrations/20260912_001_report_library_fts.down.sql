-- Migration Down: 20260912_001_report_library_fts.down.sql
-- Description:
-- Drops GIN indexes and the generated search_vector column from public.report_templates.
-- Note: Extension pg_trgm is intentionally preserved as other tables or migrations may rely on it.

DROP INDEX IF EXISTS public.idx_report_templates_trgm_name;
DROP INDEX IF EXISTS public.idx_report_templates_search_vector;

ALTER TABLE public.report_templates
    DROP COLUMN IF EXISTS search_vector;
