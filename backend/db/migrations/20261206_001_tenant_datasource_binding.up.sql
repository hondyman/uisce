-- 20261206_001_tenant_datasource_binding.up.sql
--
-- ADR-029 / ADR-032. Data-plane registry additions, in two tables:
--
--   tenant_lakehouse           exactly ONE Iceberg warehouse per tenant (the
--                              tenant_id primary key is the guarantee).
--   tenant_datasource_binding  per-datasource warm/cache bindings, plus the
--                              datasource's namespace inside the tenant's one
--                              warehouse.
--
-- The Postgres tenant database is already named by
-- tenant_product_datasource.config (host/port/database/secret_path) and is
-- deliberately not duplicated here.
--
-- References and names only. No credential of any kind is stored in either
-- table; secrets stay in the secrets store (internal/dscreds).
--
-- Additive: two new tables, no change to any existing object.

-- ---------------------------------------------------------------------------
-- One warehouse per tenant.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.tenant_lakehouse (
    -- PRIMARY KEY on tenant_id is what makes it one-and-only-one.
    -- ON DELETE RESTRICT, not CASCADE: the warehouse holds WORM audit and must
    -- outlive the tenant row. Offboarding moves lifecycle_state forward; it does
    -- not delete this record (ADR-032).
    tenant_id             UUID PRIMARY KEY
        REFERENCES public.tenants(id) ON DELETE RESTRICT,

    -- Both names are derived from the tenant id and the database refuses any
    -- other value, so a warehouse cannot be bound to a tenant it is not named for.
    warehouse_name        TEXT NOT NULL UNIQUE
        CHECK (warehouse_name = 'ivy-t-' || replace(tenant_id::text, '-', '')),
    bucket                TEXT NOT NULL UNIQUE
        CHECK (bucket = 'ivy-t-' || replace(tenant_id::text, '-', '')),

    -- Lakekeeper's own id, recorded once the warehouse exists.
    lakekeeper_warehouse_id UUID,
    kms_key_id            TEXT,

    -- Default Object Lock retention for this tenant's audit bucket, in days. Set
    -- per tenant and required before the bucket is provisioned; there is no
    -- platform default. Compliance-mode retention cannot be shortened once objects
    -- are locked, so the guard trigger below refuses to lower or clear it.
    audit_retention_days  INTEGER
        CHECK (audit_retention_days IS NULL OR audit_retention_days > 0),

    lifecycle_state       TEXT NOT NULL DEFAULT 'provisioning'
        CHECK (lifecycle_state IN
            ('provisioning', 'active', 'suspended', 'offboarding', 'offboarded')),
    version               INTEGER NOT NULL DEFAULT 1,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unprovisioned rows have NULLs; the partial indexes keep them from colliding.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_lakehouse_lakekeeper_id
    ON public.tenant_lakehouse (lakekeeper_warehouse_id)
    WHERE lakekeeper_warehouse_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_lakehouse_kms_key
    ON public.tenant_lakehouse (kms_key_id)
    WHERE kms_key_id IS NOT NULL;

-- The registry must not be able to contradict the storage it describes. Once a
-- tenant's retention is set it can only go up, the key columns are immutable,
-- and a row that has a real warehouse behind it cannot be deleted out from under
-- it (the warehouse holds WORM audit and is only retired by offboarding).
CREATE OR REPLACE FUNCTION public.tenant_lakehouse_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.tenant_id <> OLD.tenant_id
           OR NEW.warehouse_name <> OLD.warehouse_name
           OR NEW.bucket <> OLD.bucket THEN
            RAISE EXCEPTION 'tenant_lakehouse: tenant_id, warehouse_name and bucket are immutable';
        END IF;
        IF OLD.audit_retention_days IS NOT NULL
           AND (NEW.audit_retention_days IS NULL OR NEW.audit_retention_days < OLD.audit_retention_days) THEN
            RAISE EXCEPTION 'tenant_lakehouse: audit retention can only be extended (% -> %)',
                OLD.audit_retention_days, NEW.audit_retention_days;
        END IF;
        RETURN NEW;
    END IF;

    -- DELETE: only a row that never got a warehouse, or one already offboarded.
    IF OLD.lakekeeper_warehouse_id IS NOT NULL AND OLD.lifecycle_state <> 'offboarded' THEN
        RAISE EXCEPTION 'tenant_lakehouse: cannot delete a provisioned warehouse record; offboard the tenant instead';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tenant_lakehouse_guard ON public.tenant_lakehouse;
CREATE TRIGGER trg_tenant_lakehouse_guard
BEFORE UPDATE OR DELETE ON public.tenant_lakehouse
FOR EACH ROW EXECUTE FUNCTION public.tenant_lakehouse_guard();

-- ---------------------------------------------------------------------------
-- Per-datasource bindings.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.tenant_datasource_binding (
    datasource_id             UUID PRIMARY KEY
        REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    tenant_id                 UUID NOT NULL
        REFERENCES public.tenants(id) ON DELETE CASCADE,

    -- Warm tier (ADR-033): one native database and role per tenant.
    starrocks_database        TEXT,
    starrocks_role            TEXT,
    starrocks_resource_group  TEXT,

    -- Cache (ADR-034): ACL user restricted to ~t:{<tenant>}:* .
    redis_acl_user            TEXT,
    redis_key_prefix          TEXT,

    -- Cold tier (ADR-032): the datasource's namespace inside the tenant's ONE
    -- warehouse. A namespace is a name, not a warehouse.
    lake_namespace            TEXT,

    lifecycle_state           TEXT NOT NULL DEFAULT 'provisioning'
        CHECK (lifecycle_state IN
            ('provisioning', 'active', 'suspended', 'offboarding', 'offboarded')),
    version                   INTEGER NOT NULL DEFAULT 1,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A physical resource must never be bound to two tenants.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tdb_starrocks_database
    ON public.tenant_datasource_binding (starrocks_database)
    WHERE starrocks_database IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tdb_starrocks_role
    ON public.tenant_datasource_binding (starrocks_role)
    WHERE starrocks_role IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tdb_redis_acl_user
    ON public.tenant_datasource_binding (redis_acl_user)
    WHERE redis_acl_user IS NOT NULL;
-- Namespaces only need to be unique within the tenant's warehouse.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tdb_tenant_lake_namespace
    ON public.tenant_datasource_binding (tenant_id, lake_namespace)
    WHERE lake_namespace IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tdb_tenant
    ON public.tenant_datasource_binding (tenant_id);

-- ---------------------------------------------------------------------------
-- Fail-closed tenant isolation, same shape as 20261016_001: with the session
-- GUC unset uisce_get_current_tenant() is NULL, so no row matches.
-- ---------------------------------------------------------------------------
ALTER TABLE public.tenant_lakehouse ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_lakehouse FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_lakehouse_isolation_policy ON public.tenant_lakehouse;
CREATE POLICY tenant_lakehouse_isolation_policy ON public.tenant_lakehouse
    FOR ALL
    USING (tenant_id = uisce_get_current_tenant())
    WITH CHECK (tenant_id = uisce_get_current_tenant());

ALTER TABLE public.tenant_datasource_binding ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.tenant_datasource_binding FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_datasource_binding_isolation_policy
    ON public.tenant_datasource_binding;
CREATE POLICY tenant_datasource_binding_isolation_policy
    ON public.tenant_datasource_binding
    FOR ALL
    USING (tenant_id = uisce_get_current_tenant())
    WITH CHECK (tenant_id = uisce_get_current_tenant());

COMMENT ON TABLE public.tenant_lakehouse IS
    'The one Iceberg warehouse of a tenant (ADR-032). Names are derived from tenant_id by CHECK. References only.';
COMMENT ON TABLE public.tenant_datasource_binding IS
    'Warm/cache bindings and lake namespace for a tenant datasource (ADR-029). References only; credentials live in the secrets store.';
