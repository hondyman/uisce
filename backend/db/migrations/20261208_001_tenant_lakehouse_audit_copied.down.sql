-- 20261208_001_tenant_lakehouse_audit_copied (down)
--
-- Drops the status marker only. The Iceberg audit copy itself is untouched, and so is alpha's audit
-- table, which remains the system of record.
DROP TRIGGER IF EXISTS trg_tenant_lakehouse_audit_copied_guard ON public.tenant_lakehouse;
DROP FUNCTION IF EXISTS public.tenant_lakehouse_audit_copied_guard();
ALTER TABLE public.tenant_lakehouse DROP COLUMN IF EXISTS audit_copied_through_id;
