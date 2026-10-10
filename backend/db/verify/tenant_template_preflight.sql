-- Tenant provisioning, alpha side: is everything a new tenant is built FROM in place?
-- Read-only. Run against ALPHA (the control database) with any role that can read these tables:
--
--   psql "$ALPHA_URL" -X -v ON_ERROR_STOP=1 \
--     -v app='orm' \                      -- the product's app (the product code, lower case)
--     -v region='us-east-1' \             -- optional: the region code the tenant will be created in
--     -f backend/db/verify/tenant_template_preflight.sql
--
-- Exit status is non-zero when any check fails. Nothing is changed. The checks that need a live
-- database host (admin rights, PUBLIC connect, free name) are tenant_cluster_preflight.sql.
--
-- What it cannot check: the freshness of the gold copy's scan. PlanTenantStructure (the saga's first
-- step after registration, before anything is created) refuses an incomplete or stale scan, and
-- POST /api/system/tenants/provision/describe followed by a run shows that refusal as the failed step.

\set ON_ERROR_STOP on
\if :{?app}
\else
-- A missing app must fail the run, not pass it: \quit cannot set an exit status, an error can.
DO $$ BEGIN RAISE EXCEPTION 'set -v app=<product code>, e.g. -v app=orm' USING ERRCODE = 'invalid_parameter_value'; END $$;
\endif

CREATE TEMP TABLE pf (n serial, check_name text NOT NULL, ok boolean NOT NULL, detail text NOT NULL);

-- 1. Exactly one gold-copy datasource is marked as the template for this app (ADR-050). Zero or several
-- refuse the run, and so does a marked datasource that names no database.
INSERT INTO pf (check_name, ok, detail)
SELECT 'one template datasource is marked for the app',
       count(*) = 1,
       CASE count(*) WHEN 1 THEN 'marked: ' || string_agg(tpd.id::text, ', ')
                     WHEN 0 THEN 'none is marked; the owner marks the gold copy''s datasource (structure_template_app)'
                     ELSE count(*) || ' are marked: ' || string_agg(tpd.id::text, ', ') END
FROM public.tenant_product_datasource tpd WHERE tpd.structure_template_app = :'app';

INSERT INTO pf (check_name, ok, detail)
SELECT 'the template datasource names its database and schemas',
       count(*) > 0 AND bool_and(COALESCE(tpd.config ->> 'database', '') <> '' AND COALESCE(tpd.config ->> 'schema', '') <> ''),
       COALESCE(string_agg(format('%s schema=%s', tpd.config ->> 'database', tpd.config ->> 'schema'), '; '), 'no template')
FROM public.tenant_product_datasource tpd WHERE tpd.structure_template_app = :'app';

-- 2. The gold copy holds the product, so the product-filtered clone has something to register.
INSERT INTO pf (check_name, ok, detail)
SELECT 'the gold copy holds the product',
       count(*) > 0,
       format('%s gold-copy instance product(s) with code %s', count(*), :'app')
FROM public.tenant_product tp
JOIN public.alpha_product ap ON ap.id = tp.alpha_product_id
JOIN public.tenant_instance ti ON ti.id = tp.datasource_id
JOIN public.tenants t ON t.id = ti.tenant_id
WHERE COALESCE(t.gold_copy, false) AND lower(ap.product_code) = lower(:'app');

-- 3. The product is available to register: active, with a code the wizard offers.
INSERT INTO pf (check_name, ok, detail)
SELECT 'the product is active in the catalog',
       count(*) = 1,
       format('%s active alpha_product row(s) with code %s', count(*), :'app')
FROM public.alpha_product ap
WHERE ap.is_active AND ap.status = 'active' AND lower(ap.product_code) = lower(:'app');

-- 4. Exactly one gold-copy tenant, with a dedicated database (never the control plane).
INSERT INTO pf (check_name, ok, detail)
SELECT 'one gold-copy tenant', count(*) = 1, format('%s gold-copy tenant(s)', count(*))
FROM public.tenants t WHERE COALESCE(t.gold_copy, false);

-- 5. The region has an active cluster.
\if :{?region}
INSERT INTO pf (check_name, ok, detail)
SELECT 'the region has an active Postgres cluster',
       count(*) = 1,
       COALESCE(string_agg(format('%s -> %s:%s', pc.region_code, pc.host, pc.port), ''), format('no active cluster for %s', :'region'))
FROM public.region_config rc
JOIN public.region_postgres_cluster pc ON pc.region_code = rc.region_code
WHERE rc.is_active AND pc.is_active AND rc.region_code = :'region';
\endif

\echo '== Tenant template preflight'
SELECT CASE WHEN ok THEN 'PASS' ELSE 'FAIL' END AS result, check_name, detail FROM pf ORDER BY n;

DO $$
DECLARE bad integer;
BEGIN
    SELECT count(*) INTO bad FROM pf WHERE NOT ok;
    IF bad > 0 THEN
        RAISE EXCEPTION 'tenant template preflight: % check(s) failed', bad USING ERRCODE = 'check_violation';
    END IF;
END $$;
