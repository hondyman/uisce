-- 20261209_001_binding_lifecycle_windows.up.sql
--
-- ADR-038. Additive only: new columns and one new index on the table created by
-- 20261206_001. Nothing existing is edited, dropped or retyped.
--
-- Cold retention is deliberately NOT here; it is tenant_lakehouse.audit_retention_days.

ALTER TABLE public.tenant_datasource_binding
    ADD COLUMN IF NOT EXISTS pg_cluster         TEXT,
    ADD COLUMN IF NOT EXISTS hot_window_days    INTEGER NOT NULL DEFAULT 90
        CHECK (hot_window_days > 0),
    ADD COLUMN IF NOT EXISTS warm_window_months INTEGER NOT NULL DEFAULT 13
        CHECK (warm_window_months > 0),
    ADD COLUMN IF NOT EXISTS legal_hold         BOOLEAN NOT NULL DEFAULT false;

-- A key prefix must never be bound to two tenants (the other physical names already have this).
CREATE UNIQUE INDEX IF NOT EXISTS uq_tdb_redis_key_prefix
    ON public.tenant_datasource_binding (redis_key_prefix)
    WHERE redis_key_prefix IS NOT NULL;

COMMENT ON COLUMN public.tenant_datasource_binding.pg_cluster IS
    'NULL = the shared Postgres cluster; otherwise the dedicated cluster name.';
COMMENT ON COLUMN public.tenant_datasource_binding.legal_hold IS
    'Suspends partition drops (ADR-035). Never alters Object Lock.';
