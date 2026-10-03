DROP INDEX IF EXISTS public.uq_tdb_pg_role;
ALTER TABLE public.tenant_datasource_binding
    DROP COLUMN IF EXISTS pg_credential_issued_at,
    DROP COLUMN IF EXISTS pg_role;
