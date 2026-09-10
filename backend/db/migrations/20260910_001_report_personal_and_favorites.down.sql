-- Migration Down: 20260910_001_report_personal_and_favorites.down.sql
-- Note: Rolling back drops the `report_favorites` table, which will remove user favorite associations.

DROP TABLE IF EXISTS public.report_favorites;
DROP INDEX IF EXISTS public.uq_report_templates_tenant_lower_name;
DROP INDEX IF EXISTS public.idx_report_templates_personal;
DROP INDEX IF EXISTS public.idx_report_templates_creator;

ALTER TABLE public.report_templates 
    DROP COLUMN IF EXISTS is_personal,
    DROP COLUMN IF EXISTS created_by_id;
