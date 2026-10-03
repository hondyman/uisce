-- 20261210_001_binding_pg_credential.up.sql
--
-- ADR-030. Additive only: two columns and one index on tenant_datasource_binding.
--
-- pg_role                 the LOGIN role that owns access to the tenant's database.
-- pg_credential_issued_at set once, when the role's credential is first issued. It is the
--                         transactional record that lets provisioning tell "never issued" (safe to
--                         mint) from "issued but the secrets store cannot find it" (an outage or
--                         a loss, which must never silently mint a new credential). The secrets
--                         store alone cannot make that distinction (same reasoning as
--                         tenant_lakehouse.credential_issued_at).
--
-- No credential is stored here; the password lives in the secrets store at the datasource's
-- canonical path.

ALTER TABLE public.tenant_datasource_binding
    ADD COLUMN IF NOT EXISTS pg_role TEXT,
    ADD COLUMN IF NOT EXISTS pg_credential_issued_at TIMESTAMPTZ;

-- A role must never be bound to two datasources.
CREATE UNIQUE INDEX IF NOT EXISTS uq_tdb_pg_role
    ON public.tenant_datasource_binding (pg_role)
    WHERE pg_role IS NOT NULL;

COMMENT ON COLUMN public.tenant_datasource_binding.pg_role IS
    'LOGIN role for the tenant database (ADR-030). The password is in the secrets store, never here.';
COMMENT ON COLUMN public.tenant_datasource_binding.pg_credential_issued_at IS
    'Set once when the role credential is first issued; stops silent credential rotation.';
