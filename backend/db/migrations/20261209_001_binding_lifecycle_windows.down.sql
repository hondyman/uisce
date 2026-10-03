DROP INDEX IF EXISTS public.uq_tdb_redis_key_prefix;
ALTER TABLE public.tenant_datasource_binding
    DROP COLUMN IF EXISTS legal_hold,
    DROP COLUMN IF EXISTS warm_window_months,
    DROP COLUMN IF EXISTS hot_window_days,
    DROP COLUMN IF EXISTS pg_cluster;
