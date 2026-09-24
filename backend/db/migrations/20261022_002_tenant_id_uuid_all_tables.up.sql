-- 20261022_002_tenant_id_uuid_all_tables.up.sql
--
-- Converts every text/varchar tenant_id column to uuid (786 tables already use
-- uuid; this fixes the remaining base tables, catalog-driven).
--
-- Fail-closed by design; the whole migration aborts (and rolls back) if:
--   * any tenant_id value is not a valid UUID,
--   * any RLS policy expression that mentions tenant_id is not one of the two
--     recognised forms (see pg_temp.tid_rewrite),
--   * a dependent view cannot be recreated.
--
-- What it does, per affected table:
--   1. Captures and drops RLS policies that mention tenant_id.
--   2. Captures and drops views that depend on tenant_id (transitively).
--   3. Drops the column DEFAULT if any (sentinel defaults such as
--      'default-tenant' cannot be uuids; inserts must now supply a tenant).
--   4. ALTER COLUMN tenant_id TYPE uuid USING tenant_id::uuid.
--   5. Recreates views (definition, owner, grants, options, comment).
--   6. Recreates policies with the tenant comparison rewritten to
--        tenant_id = NULLIF(current_setting(<key>, true), '')::uuid
--      which keeps the same strictness: an unset or empty setting yields NULL,
--      so no rows match.
--
-- Originals (types, defaults, policies, views) are saved in
-- public.tenant_id_uuid_migration_backup for restoration.

CREATE FUNCTION pg_temp.tid_rewrite(e text) RETURNS text
LANGUAGE plpgsql AS $fn$
DECLARE
    r text := e;
    v text;
BEGIN
    IF e IS NULL THEN
        RETURN NULL;
    END IF;
    -- ((tenant_id)::text = current_setting('k'::text, true))
    r := regexp_replace(r, '\(tenant_id\)::text = current_setting\(([^)]*)\)',
                        'tenant_id = NULLIF(current_setting(\1), '''')::uuid', 'g');
    -- (tenant_id = current_setting('k'::text, true))  [text column]
    r := regexp_replace(r, '(^|[^A-Za-z0-9_.])tenant_id = current_setting\(([^)]*)\)',
                        '\1tenant_id = NULLIF(current_setting(\2), '''')::uuid', 'g');
    -- Anything mentioning tenant_id outside the accepted forms is unrecognised.
    v := regexp_replace(r, 'tenant_id = NULLIF\(current_setting\([^)]*\), ''''\)::uuid', '', 'g');
    v := regexp_replace(v, 'tenant_id IS NULL', '', 'g');
    IF v ~ 'tenant_id' THEN
        RAISE EXCEPTION 'tenant_id uuid migration: unrecognised policy expression: %', e;
    END IF;
    RETURN r;
END
$fn$;

DO $mig$
DECLARE
    r        record;
    bad      bigint;
    n_tables int := 0;
    n_pol    int := 0;
    n_views  int := 0;
    n_def    int := 0;
    ddl      text;
    vdef     text;
BEGIN
    CREATE TABLE IF NOT EXISTS public.tenant_id_uuid_migration_backup (
        id          bigserial PRIMARY KEY,
        kind        text  NOT NULL,   -- column | policy | view
        schema_name text  NOT NULL,
        object_name text  NOT NULL,
        detail      jsonb NOT NULL,
        created_at  timestamptz NOT NULL DEFAULT now()
    );

    CREATE TEMP TABLE _tid_t AS
    SELECT c.oid AS reloid, n.nspname AS s, c.relname AS t,
           format_type(a.atttypid, a.atttypmod) AS oldtype
    FROM pg_attribute a
    JOIN pg_class c ON c.oid = a.attrelid AND c.relkind = 'r'
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE a.attname = 'tenant_id' AND NOT a.attisdropped
      AND a.atttypid IN ('text'::regtype, 'varchar'::regtype)
      AND n.nspname NOT IN ('pg_catalog', 'information_schema')
      AND n.nspname NOT LIKE 'pg\_%' ESCAPE '\';

    -- 1. Pre-flight: every value must already be a valid UUID.
    FOR r IN SELECT * FROM _tid_t LOOP
        EXECUTE format(
            $q$SELECT count(*) FROM %I.%I WHERE tenant_id IS NOT NULL
               AND tenant_id !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'$q$,
            r.s, r.t) INTO bad;
        IF bad > 0 THEN
            RAISE EXCEPTION 'tenant_id uuid migration: %.% has % non-UUID tenant_id value(s)', r.s, r.t, bad;
        END IF;
    END LOOP;

    -- 2. Policies that mention tenant_id: capture, rewrite, validate.
    CREATE TEMP TABLE _tid_pol AS
    SELECT p.schemaname AS s, p.tablename AS t, p.policyname AS pol, p.permissive, p.roles, p.cmd,
           p.qual, p.with_check,
           pg_temp.tid_rewrite(p.qual)       AS new_qual,
           pg_temp.tid_rewrite(p.with_check) AS new_check
    FROM pg_policies p
    JOIN _tid_t t ON t.s = p.schemaname AND t.t = p.tablename
    WHERE coalesce(p.qual, '') ~ 'tenant_id' OR coalesce(p.with_check, '') ~ 'tenant_id';

    INSERT INTO public.tenant_id_uuid_migration_backup (kind, schema_name, object_name, detail)
    SELECT 'policy', s, t, jsonb_build_object('policy', pol, 'permissive', permissive, 'roles', roles,
                                             'cmd', cmd, 'qual', qual, 'with_check', with_check)
    FROM _tid_pol;

    FOR r IN SELECT * FROM _tid_pol LOOP
        EXECUTE format('DROP POLICY %I ON %I.%I', r.pol, r.s, r.t);
        n_pol := n_pol + 1;
    END LOOP;

    -- 3. Views that depend on tenant_id (transitively): capture, then drop deepest first.
    CREATE TEMP TABLE _tid_v AS
    WITH RECURSIVE dv AS (
        SELECT DISTINCT rw.ev_class AS void, 1 AS depth
        FROM _tid_t t
        JOIN pg_depend d ON d.refobjid = t.reloid AND d.classid = 'pg_rewrite'::regclass
        JOIN pg_attribute a ON a.attrelid = t.reloid AND a.attnum = d.refobjsubid AND a.attname = 'tenant_id'
        JOIN pg_rewrite rw ON rw.oid = d.objid AND rw.ev_class <> t.reloid
        UNION
        SELECT rw.ev_class, dv.depth + 1
        FROM dv
        JOIN pg_depend d ON d.refobjid = dv.void AND d.classid = 'pg_rewrite'::regclass
        JOIN pg_rewrite rw ON rw.oid = d.objid AND rw.ev_class <> dv.void
    )
    SELECT c.oid AS void, max(dv.depth) AS depth, n.nspname AS s, c.relname AS v, c.relkind,
           pg_get_viewdef(c.oid, true) AS def,
           pg_get_userbyid(c.relowner) AS owner,
           c.reloptions AS opts,
           obj_description(c.oid, 'pg_class') AS cmt,
           c.relispopulated AS populated,
           (SELECT coalesce(jsonb_agg(i.indexdef ORDER BY i.indexname), '[]'::jsonb)
            FROM pg_indexes i WHERE i.schemaname = n.nspname AND i.tablename = c.relname) AS idx,
           (SELECT coalesce(jsonb_agg(jsonb_build_object(
                       'grantee', CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,
                       'priv', a.privilege_type, 'grantable', a.is_grantable)), '[]'::jsonb)
            FROM aclexplode(c.relacl) a) AS grants
    FROM dv JOIN pg_class c ON c.oid = dv.void JOIN pg_namespace n ON n.oid = c.relnamespace
    GROUP BY c.oid, n.nspname, c.relname, c.relkind, c.relowner, c.reloptions, c.relacl, c.relispopulated;

    IF EXISTS (SELECT 1 FROM _tid_v WHERE relkind NOT IN ('v', 'm')) THEN
        RAISE EXCEPTION 'tenant_id uuid migration: a dependent relation is neither a view nor a materialized view';
    END IF;

    INSERT INTO public.tenant_id_uuid_migration_backup (kind, schema_name, object_name, detail)
    SELECT 'view', s, v, jsonb_build_object('relkind', relkind, 'definition', def, 'owner', owner,
                                            'options', opts, 'comment', cmt, 'grants', grants,
                                            'populated', populated, 'indexes', idx)
    FROM _tid_v;

    FOR r IN SELECT * FROM _tid_v ORDER BY depth DESC LOOP
        EXECUTE format('DROP %s %I.%I', CASE WHEN r.relkind = 'm' THEN 'MATERIALIZED VIEW' ELSE 'VIEW' END, r.s, r.v);
        n_views := n_views + 1;
    END LOOP;

    -- 4. Convert the columns (dropping sentinel defaults first).
    FOR r IN SELECT t.*, a.attnotnull,
                    pg_get_expr(ad.adbin, ad.adrelid) AS old_default
             FROM _tid_t t
             JOIN pg_attribute a ON a.attrelid = t.reloid AND a.attname = 'tenant_id'
             LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
    LOOP
        INSERT INTO public.tenant_id_uuid_migration_backup (kind, schema_name, object_name, detail)
        VALUES ('column', r.s, r.t, jsonb_build_object('old_type', r.oldtype, 'old_default', r.old_default,
                                                       'not_null', r.attnotnull));
        IF r.old_default IS NOT NULL THEN
            EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN tenant_id DROP DEFAULT', r.s, r.t);
            RAISE NOTICE 'tenant_id uuid migration: dropped default % on %.%', r.old_default, r.s, r.t;
            n_def := n_def + 1;
        END IF;
        EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN tenant_id TYPE uuid USING tenant_id::uuid', r.s, r.t);
        n_tables := n_tables + 1;
    END LOOP;

    -- 5. Recreate views, shallowest first.
    FOR r IN SELECT * FROM _tid_v ORDER BY depth ASC LOOP
        vdef := regexp_replace(r.def, ';\s*$', '');
        ddl := format('CREATE %s %I.%I %s AS %s%s',
                      CASE WHEN r.relkind = 'm' THEN 'MATERIALIZED VIEW' ELSE 'VIEW' END, r.s, r.v,
                      CASE WHEN r.opts IS NOT NULL THEN 'WITH (' || array_to_string(r.opts, ', ') || ')' ELSE '' END,
                      vdef,
                      CASE WHEN r.relkind = 'm' THEN CASE WHEN r.populated THEN ' WITH DATA' ELSE ' WITH NO DATA' END ELSE '' END);
        EXECUTE ddl;
        EXECUTE format('ALTER %s %I.%I OWNER TO %I',
                       CASE WHEN r.relkind = 'm' THEN 'MATERIALIZED VIEW' ELSE 'VIEW' END, r.s, r.v, r.owner);
        IF r.cmt IS NOT NULL THEN
            EXECUTE format('COMMENT ON %s %I.%I IS %L',
                           CASE WHEN r.relkind = 'm' THEN 'MATERIALIZED VIEW' ELSE 'VIEW' END, r.s, r.v, r.cmt);
        END IF;
        FOR vdef IN SELECT jsonb_array_elements_text(r.idx) LOOP
            EXECUTE vdef;
        END LOOP;
        EXECUTE format('REVOKE ALL ON %I.%I FROM PUBLIC', r.s, r.v);
        FOR vdef IN
            SELECT format('GRANT %s ON %I.%I TO %s%s', g->>'priv', r.s, r.v,
                          CASE WHEN g->>'grantee' = 'PUBLIC' THEN 'PUBLIC' ELSE quote_ident(g->>'grantee') END,
                          CASE WHEN (g->>'grantable')::boolean THEN ' WITH GRANT OPTION' ELSE '' END)
            FROM jsonb_array_elements(r.grants) g
        LOOP
            EXECUTE vdef;
        END LOOP;
    END LOOP;

    -- 6. Recreate policies with the uuid-safe comparison.
    FOR r IN SELECT * FROM _tid_pol LOOP
        EXECUTE format('CREATE POLICY %I ON %I.%I AS %s FOR %s TO %s%s%s',
                       r.pol, r.s, r.t, r.permissive, r.cmd,
                       (SELECT string_agg(CASE WHEN x = 'public' THEN 'public' ELSE quote_ident(x) END, ', ')
                        FROM unnest(r.roles) x),
                       CASE WHEN r.new_qual IS NOT NULL THEN ' USING (' || r.new_qual || ')' ELSE '' END,
                       CASE WHEN r.new_check IS NOT NULL THEN ' WITH CHECK (' || r.new_check || ')' ELSE '' END);
    END LOOP;

    -- The runner reuses one pooled connection, so do not leave temp tables behind.
    DROP TABLE IF EXISTS _tid_t;
    DROP TABLE IF EXISTS _tid_pol;
    DROP TABLE IF EXISTS _tid_v;

    RAISE NOTICE 'tenant_id uuid migration: % tables converted, % policies rewritten, % views recreated, % defaults dropped',
                 n_tables, n_pol, n_views, n_def;
END
$mig$;

DROP FUNCTION pg_temp.tid_rewrite(text);
