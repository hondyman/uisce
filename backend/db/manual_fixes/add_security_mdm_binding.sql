-- Add a second (non-default) binding to the "Security" business object
-- (c2000000-0000-4000-8000-000000000003) pointing at the MDM golden-record
-- table (mdm.security_golden_record), alongside its existing default ORM
-- binding (/orm/security).
--
-- Why: the security MDM tables (Part I of the "Where the Security Master
-- Went" DDL) were seeded as their own standalone business objects
-- (security_golden_record, security_asset_class, ...), so the MDM golden
-- record was never wired up as a second source on the main Security BO the
-- way ORM security is. This script adds that second binding only -- it does
-- NOT add field_bindings, because security_golden_record's actual field
-- values live inside a jsonb "golden_attributes" column rather than in
-- fixed columns matching the Security BO's field names (Cusip, Isin,
-- Ticker, SecName, ...). Only security_id and sec_typ_cd map cleanly to
-- real columns; mapping the rest needs the golden_attributes key schema,
-- which is a separate follow-up.
--
-- Usage:
--   psql "$DATABASE_URL" -v mode=plan  -f add_security_mdm_binding.sql
--   psql "$DATABASE_URL" -v mode=apply -f add_security_mdm_binding.sql
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
  IF (SELECT count(*) FROM public.business_objects WHERE id = 'c2000000-0000-4000-8000-000000000003') = 0 THEN
    RAISE EXCEPTION 'ABORT: Security BO not found';
  END IF;
  IF (SELECT count(*) FROM public.catalog_node WHERE qualified_path = '/mdm/security_golden_record') = 0 THEN
    RAISE EXCEPTION 'ABORT: driving catalog_node /mdm/security_golden_record not found';
  END IF;
  IF (SELECT count(*) FROM public.physical_backend WHERE backend_id = '441f62c9-aad1-481d-9aab-62943fa11cd3') = 0 THEN
    RAISE EXCEPTION 'ABORT: backend 441f62c9-aad1-481d-9aab-62943fa11cd3 not found';
  END IF;
  IF EXISTS (
    SELECT 1 FROM public.business_object_binding
    WHERE bo_id = 'c2000000-0000-4000-8000-000000000003'
      AND driving_node_id = (SELECT id FROM public.catalog_node WHERE qualified_path = '/mdm/security_golden_record')
  ) THEN
    RAISE EXCEPTION 'ABORT: a binding from Security BO to /mdm/security_golden_record already exists';
  END IF;
END $$;

INSERT INTO public.business_object_binding
  (bo_binding_id, tenant_id, bo_id, backend_id, driving_node_id,
   binding_name, is_default, is_active, is_core, temporal_mode, temporal_type)
SELECT
  gen_random_uuid(),
  bo.tenant_id,
  bo.id,
  '441f62c9-aad1-481d-9aab-62943fa11cd3',
  cn.id,
  'Security MDM Golden Record Binding',
  false,   -- non-default: ORM binding stays the default source
  true,
  true,
  'NONE',
  'NONE'
FROM public.business_objects bo
CROSS JOIN public.catalog_node cn
WHERE bo.id = 'c2000000-0000-4000-8000-000000000003'
  AND cn.qualified_path = '/mdm/security_golden_record';

\echo '-- bindings on the Security BO after this change --'
SELECT b.bo_binding_id, b.binding_name, b.is_default, b.is_active, cn.qualified_path
FROM public.business_object_binding b
JOIN public.catalog_node cn ON cn.id = b.driving_node_id
WHERE b.bo_id = 'c2000000-0000-4000-8000-000000000003'
ORDER BY b.is_default DESC;

\if :is_apply
  \echo '== APPLY: committing'
  COMMIT;
\else
  \echo '== PLAN: rolling back (no changes made)'
  ROLLBACK;
\endif
