-- 20261216_001_platform_lakehouse.up.sql
--
-- ADR-045. The platform's own warehouse, ivy-control (ADR-032): exactly ONE row, because there is exactly one.
-- It belongs to no tenant, so there is no tenant_id and no row-level security: it holds no tenant data, only
-- the platform bucket's own provisioning state. Its job is the same as tenant_lakehouse's for a tenant:
--
--   credential_issued_at   the transactional record that lets provisioning tell "never issued" (safe to mint)
--                          from "issued but the secrets store cannot find it" (an outage or a loss, which must
--                          never silently mint a new credential and strand the warehouse on stale keys).
--   audit_retention_days   the Object Lock retention the bucket is to carry, an explicit decision with no default.
--                          Compliance-mode retention cannot be shortened, so it can only be raised.
--   retention_applied_days what the bucket was actually created with; never above the desired value; only rises.
--
-- Forward-only and additive. Nothing here touches a tenant table.
CREATE TABLE IF NOT EXISTS public.platform_lakehouse (
    -- The only legal value, as a primary key: a second warehouse cannot be recorded.
    name                    TEXT PRIMARY KEY CHECK (name = 'ivy-control'),
    bucket                  TEXT NOT NULL CHECK (bucket = 'ivy-control'),
    audit_retention_days    INTEGER NOT NULL CHECK (audit_retention_days > 0),
    retention_applied_days  INTEGER CHECK (retention_applied_days IS NULL OR
                                           (retention_applied_days > 0 AND retention_applied_days <= audit_retention_days)),
    kms_key_id              TEXT,
    lakekeeper_warehouse_id UUID,
    credential_issued_at    TIMESTAMPTZ,
    provisioned_at          TIMESTAMPTZ,
    version                 INTEGER NOT NULL DEFAULT 1,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Provisioned means all three exist together.
    CONSTRAINT platform_lakehouse_provisioned_chk CHECK (
        (provisioned_at IS NULL) = (lakekeeper_warehouse_id IS NULL)
        AND (provisioned_at IS NULL) = (kms_key_id IS NULL)
        AND (provisioned_at IS NULL) = (retention_applied_days IS NULL)
    )
);

CREATE OR REPLACE FUNCTION public.platform_lakehouse_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.lakekeeper_warehouse_id IS NOT NULL THEN
            RAISE EXCEPTION 'platform_lakehouse: cannot delete the record of a provisioned warehouse';
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.name <> OLD.name OR NEW.bucket <> OLD.bucket THEN
        RAISE EXCEPTION 'platform_lakehouse: name and bucket are immutable';
    END IF;
    IF NEW.audit_retention_days < OLD.audit_retention_days THEN
        RAISE EXCEPTION 'platform_lakehouse: audit retention can only be extended (% -> %)',
            OLD.audit_retention_days, NEW.audit_retention_days;
    END IF;
    IF OLD.retention_applied_days IS NOT NULL
       AND (NEW.retention_applied_days IS NULL OR NEW.retention_applied_days < OLD.retention_applied_days) THEN
        RAISE EXCEPTION 'platform_lakehouse: applied retention can only rise (% -> %)',
            OLD.retention_applied_days, NEW.retention_applied_days;
    END IF;
    IF OLD.lakekeeper_warehouse_id IS NOT NULL AND NEW.lakekeeper_warehouse_id IS DISTINCT FROM OLD.lakekeeper_warehouse_id THEN
        RAISE EXCEPTION 'platform_lakehouse: the warehouse id is set once';
    END IF;
    IF OLD.credential_issued_at IS NOT NULL AND NEW.credential_issued_at IS DISTINCT FROM OLD.credential_issued_at THEN
        RAISE EXCEPTION 'platform_lakehouse: credential_issued_at is set once';
    END IF;
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_platform_lakehouse_guard ON public.platform_lakehouse;
CREATE TRIGGER trg_platform_lakehouse_guard
BEFORE UPDATE OR DELETE ON public.platform_lakehouse
FOR EACH ROW EXECUTE FUNCTION public.platform_lakehouse_guard();

COMMENT ON TABLE public.platform_lakehouse IS
    'The platform warehouse ivy-control (ADR-032, ADR-045): one row, no tenant. Provisioning state only.';
