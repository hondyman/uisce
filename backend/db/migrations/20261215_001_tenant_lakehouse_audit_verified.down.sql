-- 20261215_001_tenant_lakehouse_audit_verified (down)
--
-- Drops the recorded verification outcome only. The Iceberg copy and alpha's audit are untouched.
ALTER TABLE public.tenant_lakehouse DROP CONSTRAINT IF EXISTS tenant_lakehouse_audit_verify_chk;
ALTER TABLE public.tenant_lakehouse
    DROP COLUMN IF EXISTS audit_verify_finding_id,
    DROP COLUMN IF EXISTS audit_verify_finding_kind,
    DROP COLUMN IF EXISTS audit_verified_at,
    DROP COLUMN IF EXISTS audit_verified_through_id;
