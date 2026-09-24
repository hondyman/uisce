-- 20261018_002_report_templates_spine_columns.down.sql
DROP INDEX IF EXISTS public.idx_report_templates_is_core;
ALTER TABLE public.report_templates DROP COLUMN IF EXISTS is_core;
ALTER TABLE public.report_templates DROP COLUMN IF EXISTS primary_business_object_id;
ALTER TABLE public.report_templates DROP COLUMN IF EXISTS grouping;
ALTER TABLE public.report_templates DROP COLUMN IF EXISTS presentation_events;
ALTER TABLE public.report_templates DROP COLUMN IF EXISTS parameters;
ALTER TABLE public.report_templates DROP COLUMN IF EXISTS bands;
