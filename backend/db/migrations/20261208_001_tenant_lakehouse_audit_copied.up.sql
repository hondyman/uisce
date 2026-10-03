-- 20261208_001_tenant_lakehouse_audit_copied.up.sql
--
-- ADR-036. Status marker for the Iceberg copy of a tenant's lakehouse audit: the id of the last
-- audit entry known to have been copied. It is for operators and the System page ("copied through
-- #N") and is NEVER what decides what gets shipped: the copy resumes from the destination's own
-- max(id), because a counter kept here can drift from the data (a crash between the insert and the
-- update would duplicate rows on retry; the destination cannot drift from itself).
--
-- Forward-only and additive: 20261206_001 is already merged and may be applied, so it is not
-- edited.
ALTER TABLE public.tenant_lakehouse
    ADD COLUMN IF NOT EXISTS audit_copied_through_id BIGINT;

-- Like the retention values, it only ever moves forward.
CREATE OR REPLACE FUNCTION public.tenant_lakehouse_audit_copied_guard() RETURNS trigger AS $$
BEGIN
    IF OLD.audit_copied_through_id IS NOT NULL
       AND (NEW.audit_copied_through_id IS NULL OR NEW.audit_copied_through_id < OLD.audit_copied_through_id) THEN
        RAISE EXCEPTION 'tenant_lakehouse: audit_copied_through_id can only rise (% -> %)',
            OLD.audit_copied_through_id, NEW.audit_copied_through_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tenant_lakehouse_audit_copied_guard ON public.tenant_lakehouse;
CREATE TRIGGER trg_tenant_lakehouse_audit_copied_guard
BEFORE UPDATE ON public.tenant_lakehouse
FOR EACH ROW EXECUTE FUNCTION public.tenant_lakehouse_audit_copied_guard();

COMMENT ON COLUMN public.tenant_lakehouse.audit_copied_through_id IS
    'Last audit entry id known to be copied to the tenant''s Iceberg audit table. A status marker only; the copy resumes from the destination''s max(id) (ADR-036).';
