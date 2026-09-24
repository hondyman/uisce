-- Fix the orphan ORM backend used by 5 core business-object bindings
-- (account, broker, order, position, security).
--
-- Why: business_object_binding.backend_id = e102c0bf-e111-4f9d-81f9-2801ff2a3231
-- for these 5 bindings, but that id has no matching row in public.connections,
-- and its public.physical_backend row is a bare placeholder ("backend
-- e102c0bf-...", description "Registered by bo binding consolidation", no
-- tenant_id). Because there is no connection config, record queries
-- against it fall back to the app's default database (alpha), whose own
-- unrelated "orm" schema (a different, OMS-style set of tables) has no
-- "security" table -- hence "relation \"orm.security\" does not exist".
--
-- The tables these BOs are actually bound to (orm.account, orm.broker,
-- orm.order, orm.position, orm.security) live in the crims database,
-- reachable through the existing, correctly-configured "CRIMS ORM
-- Database" connection/backend (441f62c9-aad1-481d-9aab-62943fa11cd3).
--
-- We cannot just repoint these bindings' backend_id to 441f62c9 (the
-- Security BO already has a second binding on that backend, for
-- /mdm/security_golden_record, and (tenant_id, bo_id, backend_id) is
-- unique) -- so instead we give the orphan backend id its own connection
-- config, cloned from the CRIMS ORM Database connection (same host, port,
-- database, schema, credentials), and fix its physical_backend metadata
-- (name, tenant_id) to match. This is a strict copy done entirely in SQL;
-- no credential value is ever displayed.
--
-- Usage:
--   psql "$DATABASE_URL" -v mode=plan  -f fix_orphan_orm_backend.sql
--   psql "$DATABASE_URL" -v mode=apply -f fix_orphan_orm_backend.sql
\if :{?mode}
\else
  \set mode 'plan'
\endif

\set ON_ERROR_STOP on

\echo '== mode:' :mode

BEGIN;

SELECT (:'mode' = 'apply') AS is_apply \gset

DO $$
BEGIN
  IF (SELECT count(*) FROM public.connections WHERE id = '441f62c9-aad1-481d-9aab-62943fa11cd3') = 0 THEN
    RAISE EXCEPTION 'ABORT: source connection 441f62c9-aad1-481d-9aab-62943fa11cd3 (CRIMS ORM Database) not found';
  END IF;
  IF (SELECT count(*) FROM public.physical_backend WHERE backend_id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231') = 0 THEN
    RAISE EXCEPTION 'ABORT: orphan backend e102c0bf-e111-4f9d-81f9-2801ff2a3231 not found (nothing to fix)';
  END IF;
  IF (SELECT count(*) FROM public.business_object_binding WHERE backend_id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231') <> 5 THEN
    RAISE EXCEPTION 'ABORT: expected exactly 5 bindings on the orphan backend, found a different count -- re-check before proceeding';
  END IF;
END $$;

\echo '-- affected bindings (before) --'
SELECT b.bo_binding_id, bo.bo_key, b.backend_id
FROM public.business_object_binding b
JOIN public.business_objects bo ON bo.id = b.bo_id
WHERE b.backend_id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231'
ORDER BY bo.bo_key;

-- Clone the CRIMS ORM Database connection under the orphan backend's id,
-- if it doesn't already have one.
INSERT INTO public.connections
  (id, tenant_id, name, type, host, port, database, schema, username, password,
   metadata, is_active, core_id, secret_path)
SELECT
  'e102c0bf-e111-4f9d-81f9-2801ff2a3231',
  tenant_id, 'CRIMS ORM Database (bo binding consolidation)', type, host, port, database, schema, username, password,
  metadata, is_active, core_id, secret_path
FROM public.connections
WHERE id = '441f62c9-aad1-481d-9aab-62943fa11cd3'
  AND NOT EXISTS (SELECT 1 FROM public.connections WHERE id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231');

-- Fix the physical_backend metadata to match (name, tenant_id) so it's no
-- longer a bare placeholder.
UPDATE public.physical_backend
SET backend_name = 'CRIMS ORM Database (bo binding consolidation)',
    tenant_id = (SELECT tenant_id FROM public.connections WHERE id = '441f62c9-aad1-481d-9aab-62943fa11cd3')
WHERE backend_id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231';

\echo '-- physical_backend after fix --'
SELECT backend_id, backend_name, tenant_id, dialect_name, storage_tier
FROM public.physical_backend
WHERE backend_id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231';

\echo '-- connections after fix (no credential columns selected) --'
SELECT id, name, type, host, port, database, schema, is_active
FROM public.connections
WHERE id = 'e102c0bf-e111-4f9d-81f9-2801ff2a3231';

\if :is_apply
  \echo '== APPLY: committing'
  COMMIT;
\else
  \echo '== PLAN: rolling back (no changes made)'
  ROLLBACK;
\endif
