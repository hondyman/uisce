-- 20261206_002_tenant_lakehouse_audit (down)
--
-- Drops the lakehouse configuration audit trail. This is DESTRUCTIVE to the
-- audit record itself: this table is the system of record for who changed a
-- tenant's retention. Export it first.
DROP TABLE IF EXISTS public.tenant_lakehouse_audit;
DROP FUNCTION IF EXISTS public.tenant_lakehouse_audit_verify(UUID);
DROP FUNCTION IF EXISTS public.tenant_lakehouse_audit_before_insert();
DROP FUNCTION IF EXISTS public.tenant_lakehouse_audit_append_only();
DROP FUNCTION IF EXISTS public.tenant_lakehouse_audit_hash(TEXT, UUID, TIMESTAMPTZ, TEXT, TEXT, JSONB, JSONB);
