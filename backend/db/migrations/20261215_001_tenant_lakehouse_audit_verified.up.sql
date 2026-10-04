-- 20261215_001_tenant_lakehouse_audit_verified.up.sql
--
-- ADR-044. The latest outcome of verifying a tenant's Iceberg audit copy against alpha. The tiering
-- job (ADR-035) reads it before detaching or dropping anything the copy is meant to replace.
--
-- Unlike audit_copied_through_id this is NOT monotonic: a later run that finds a problem must take
-- back an earlier pass, so each run replaces the whole outcome in one statement. Safe to rely on
-- only when audit_verify_finding_kind IS NULL, audit_verified_through_id covers what is needed, and
-- audit_verified_at is recent. A finding stores the kind and the entry id and never a value.
--
-- Forward-only and additive: earlier migrations are not edited.
ALTER TABLE public.tenant_lakehouse
    ADD COLUMN IF NOT EXISTS audit_verified_through_id BIGINT,
    ADD COLUMN IF NOT EXISTS audit_verified_at         TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS audit_verify_finding_kind TEXT,
    ADD COLUMN IF NOT EXISTS audit_verify_finding_id   BIGINT;

ALTER TABLE public.tenant_lakehouse DROP CONSTRAINT IF EXISTS tenant_lakehouse_audit_verify_chk;
ALTER TABLE public.tenant_lakehouse ADD CONSTRAINT tenant_lakehouse_audit_verify_chk CHECK (
    (audit_verified_at IS NULL) = (audit_verified_through_id IS NULL)
    AND (audit_verified_through_id IS NULL OR audit_verified_through_id >= 0)
    AND (audit_verify_finding_kind IS NULL) = (audit_verify_finding_id IS NULL)
    AND (audit_verify_finding_kind IS NULL OR audit_verified_at IS NOT NULL)
    AND (audit_verify_finding_kind IS NULL OR audit_verify_finding_kind IN
         ('alpha_chain_broken', 'missing_in_copy', 'extra_in_copy', 'differs', 'chain_break'))
);

COMMENT ON COLUMN public.tenant_lakehouse.audit_verified_through_id IS
    'Highest audit id the last verification proved identical in the Iceberg copy. Entries after it are pending, not verified (ADR-044).';
COMMENT ON COLUMN public.tenant_lakehouse.audit_verify_finding_kind IS
    'What the last verification found wrong, or NULL when it found nothing. Never holds a value from the audit payload (ADR-044).';
