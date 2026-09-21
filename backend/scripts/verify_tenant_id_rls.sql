-- verify_tenant_id_rls.sql
--
-- Behavioural RLS check for the tenant_id text->uuid conversion (migration
-- 20261022_002). Definition review cannot catch the failure mode that matters:
-- a policy that errors or mis-compares at runtime silently filters rows out.
--
-- For every converted table that has a tenant_id RLS policy, as a non-superuser
-- NOBYPASSRLS role (created inside this transaction; nothing persists), using that
-- table's own policy setting key:
--   READ   A. key = tenant A       -> sees exactly A's rows, no other tenant's
--          B. key = random UUID    -> sees 0 tenant-owned rows (filtering is active, not vacuous)
--          C. key = ''             -> sees 0 tenant-owned rows and does NOT error (fail-closed)
--          (NULL-tenant "global" rows are exempt: some policies expose them on purpose)
--   WRITE  D. key = A, INSERT a row claiming tenant B -> must be rejected by RLS (42501)
--          E. key = A, INSERT a row for tenant A      -> control, must succeed
--
-- Tables with no rows get a SYNTHETIC row (dummy values by column type) so their
-- policy has something to filter; if one cannot be built the table is reported as
-- skipped with the reason. Everything runs in one transaction that ends in ROLLBACK.
-- Side effect that does NOT roll back: sequences advance (harmless gaps).
--
-- Run:  psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f backend/scripts/verify_tenant_id_rls.sql
-- Needs a superuser (or CREATEROLE + ownership) connection.

-- A pager waiting on a keypress would hold the transaction open (and the role
-- uncommitted); never allow it.
\pset pager off

-- Unique per run, so a stuck earlier session can never block this one.
SELECT 'tid_rls_probe_' || pg_backend_pid() AS probe \gset

BEGIN;

CREATE ROLE :"probe" NOLOGIN NOSUPERUSER NOBYPASSRLS;
SELECT set_config('tid.probe', :'probe', true);

CREATE TEMP TABLE _rls_results (
    tbl text, setting_key text, source text,
    rows_total bigint, rows_for_a bigint, seen_a bigint, seen_other_tenants bigint,
    seen_random bigint, seen_empty bigint,
    write_cross_tenant text, write_control text,
    err text, note text, read_pass boolean, write_pass boolean
);

DO $t$
DECLARE
    r        record;
    probe    text := current_setting('tid.probe');
    a        uuid;
    b        uuid := gen_random_uuid();
    tot bigint; tot_a bigint;
    v_a bigint; v_other bigint; v_rand bigint; v_empty bigint;
    v_err text; v_note text; v_src text;
    w_cross text; w_ctrl text;
    has_id boolean;
    cols text; vals text;
    ovr_b jsonb; ovr_a jsonb;
BEGIN
    FOR r IN
        SELECT DISTINCT b2.schema_name AS s, b2.object_name AS t,
               (regexp_match(b2.detail->>'qual', 'current_setting\(''([^'']+)'''))[1] AS k
        FROM public.tenant_id_uuid_migration_backup b2
        WHERE b2.kind = 'policy'
          AND b2.detail->>'qual' ~ 'tenant_id'
          AND b2.detail->>'qual' ~ 'current_setting'
          AND b2.detail->>'cmd' IN ('ALL', 'SELECT')
        ORDER BY 1, 2
    LOOP
        RESET ROLE;
        PERFORM set_config('uisce.current_tenant', '', true);
        PERFORM set_config('app.tenant_id', '', true);
        v_err := NULL; v_note := NULL; v_src := 'data';
        v_a := NULL; v_other := NULL; v_rand := NULL; v_empty := NULL;
        w_cross := NULL; w_ctrl := NULL;

        EXECUTE format('SELECT tenant_id FROM %I.%I WHERE tenant_id IS NOT NULL LIMIT 1', r.s, r.t) INTO a;

        IF a IS NULL THEN
            -- Empty table: try a synthetic row (NOT NULL, no-default columns get dummy values by type).
            a := gen_random_uuid();
            SELECT string_agg(quote_ident(c.column_name), ',' ORDER BY c.ordinal_position),
                   string_agg(CASE
                       WHEN c.column_name = 'tenant_id' THEN quote_literal(a::text) || '::uuid'
                       WHEN c.data_type = 'uuid' THEN 'gen_random_uuid()'
                       WHEN c.data_type IN ('text', 'character varying', 'character') THEN quote_literal('probe')
                       WHEN c.data_type IN ('integer', 'bigint', 'smallint', 'numeric', 'real', 'double precision') THEN '0'
                       WHEN c.data_type = 'boolean' THEN 'false'
                       WHEN c.data_type LIKE 'timestamp%' THEN 'now()'
                       WHEN c.data_type = 'date' THEN 'current_date'
                       WHEN c.data_type IN ('jsonb', 'json') THEN quote_literal('{}') || '::' || c.data_type
                       WHEN c.data_type = 'ARRAY' THEN quote_literal('{}') || '::' || quote_ident(c.udt_name)
                       ELSE 'NULL' END, ',' ORDER BY c.ordinal_position)
              INTO cols, vals
            FROM information_schema.columns c
            WHERE c.table_schema = r.s AND c.table_name = r.t
              AND (c.column_name = 'tenant_id'
                   OR (c.is_nullable = 'NO' AND c.column_default IS NULL
                       AND c.is_generated = 'NEVER' AND c.is_identity = 'NO'));
            BEGIN
                EXECUTE format('INSERT INTO %I.%I (%s) VALUES (%s)', r.s, r.t, cols, vals);
                v_src := 'synthetic';
            EXCEPTION WHEN OTHERS THEN
                INSERT INTO _rls_results (tbl, setting_key, source, note)
                VALUES (r.s || '.' || r.t, r.k, 'skipped', 'synthetic row impossible: ' || left(SQLERRM, 90));
                CONTINUE;
            END;
        END IF;

        EXECUTE format('SELECT count(*) FROM %I.%I', r.s, r.t) INTO tot;
        EXECUTE format('SELECT count(*) FROM %I.%I WHERE tenant_id = %L', r.s, r.t, a) INTO tot_a;

        SELECT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_schema = r.s AND table_name = r.t AND column_name = 'id' AND data_type = 'uuid')
          INTO has_id;
        ovr_b := jsonb_build_object('tenant_id', b);
        ovr_a := jsonb_build_object('tenant_id', a);

        EXECUTE format('GRANT USAGE ON SCHEMA %I TO %I', r.s, probe);
        EXECUTE format('GRANT SELECT, INSERT ON %I.%I TO %I', r.s, r.t, probe);

        EXECUTE format('SET LOCAL ROLE %I', probe);
        BEGIN
            -- READ
            PERFORM set_config(r.k, a::text, true);
            EXECUTE format('SELECT count(*) FILTER (WHERE tenant_id = %L), count(*) FILTER (WHERE tenant_id <> %L) FROM %I.%I',
                           a, a, r.s, r.t) INTO v_a, v_other;
            -- Rows with a NULL tenant are "global" by design on some tables (e.g. persona_definitions:
            -- policy tenant_id IS NULL OR ...), so they are excluded from the must-be-0 assertions.
            PERFORM set_config(r.k, b::text, true);
            EXECUTE format('SELECT count(*) FILTER (WHERE tenant_id IS NOT NULL) FROM %I.%I', r.s, r.t) INTO v_rand;
            PERFORM set_config(r.k, '', true);
            EXECUTE format('SELECT count(*) FILTER (WHERE tenant_id IS NOT NULL) FROM %I.%I', r.s, r.t) INTO v_empty;

            -- WRITE: key = A. D: claim tenant B (must be rejected). E: claim tenant A (control).
            PERFORM set_config(r.k, a::text, true);
            BEGIN
                EXECUTE format('INSERT INTO %I.%I SELECT (jsonb_populate_record(NULL::%I.%I, to_jsonb(x) || %L::jsonb || %L::jsonb)).* FROM %I.%I x WHERE x.tenant_id = %L LIMIT 1',
                               r.s, r.t, r.s, r.t, ovr_b,
                               CASE WHEN has_id THEN jsonb_build_object('id', gen_random_uuid()) ELSE '{}'::jsonb END,
                               r.s, r.t, a);
                w_cross := 'ALLOWED';
            EXCEPTION
                WHEN insufficient_privilege THEN w_cross := 'blocked';
                WHEN OTHERS THEN w_cross := 'inconclusive:' || SQLSTATE;
            END;
            BEGIN
                EXECUTE format('INSERT INTO %I.%I SELECT (jsonb_populate_record(NULL::%I.%I, to_jsonb(x) || %L::jsonb || %L::jsonb)).* FROM %I.%I x WHERE x.tenant_id = %L LIMIT 1',
                               r.s, r.t, r.s, r.t, ovr_a,
                               CASE WHEN has_id THEN jsonb_build_object('id', gen_random_uuid()) ELSE '{}'::jsonb END,
                               r.s, r.t, a);
                w_ctrl := 'ok';
            EXCEPTION WHEN OTHERS THEN w_ctrl := 'failed:' || SQLSTATE;
            END;
        EXCEPTION WHEN OTHERS THEN
            v_err := SQLERRM;
        END;
        RESET ROLE;

        INSERT INTO _rls_results VALUES (
            r.s || '.' || r.t, r.k, v_src, tot, tot_a, v_a, v_other, v_rand, v_empty, w_cross, w_ctrl, v_err, v_note,
            v_err IS NULL AND v_a = tot_a AND v_other = 0 AND v_rand = 0 AND v_empty = 0,
            w_cross = 'blocked' AND w_ctrl = 'ok');
    END LOOP;
END
$t$;

\echo
\echo '=== SUMMARY ==='
SELECT count(*) FILTER (WHERE source <> 'skipped')                              AS tables_probed,
       count(*) FILTER (WHERE source = 'data')                                  AS with_real_rows,
       count(*) FILTER (WHERE source = 'synthetic')                             AS with_synthetic_row,
       count(*) FILTER (WHERE source = 'skipped')                               AS skipped,
       count(*) FILTER (WHERE read_pass)                                        AS read_passed,
       count(*) FILTER (WHERE source <> 'skipped' AND NOT read_pass)            AS READ_FAILED,
       count(*) FILTER (WHERE write_cross_tenant = 'blocked')                   AS write_blocked,
       count(*) FILTER (WHERE write_cross_tenant = 'ALLOWED')                   AS CROSS_TENANT_WRITE_ALLOWED,
       count(*) FILTER (WHERE write_cross_tenant LIKE 'inconclusive%')          AS write_inconclusive,
       count(*) FILTER (WHERE write_control <> 'ok' AND source <> 'skipped')    AS control_insert_failed
FROM _rls_results;

\echo
\echo '=== READ FAILURES or CROSS-TENANT WRITES ALLOWED (empty means none) ==='
SELECT tbl, setting_key, source, rows_for_a, seen_a, seen_other_tenants, seen_random, seen_empty, write_cross_tenant, err
FROM _rls_results
WHERE source <> 'skipped' AND (NOT read_pass OR write_cross_tenant = 'ALLOWED')
ORDER BY tbl;

\echo
\echo '=== WRITE INCONCLUSIVE (SQLSTATE shown; not a failure, the insert failed before RLS could decide) ==='
SELECT write_cross_tenant AS outcome, count(*) AS tables, string_agg(tbl, ', ' ORDER BY tbl) FILTER (WHERE true) AS example_tables
FROM (SELECT * FROM _rls_results WHERE write_cross_tenant LIKE 'inconclusive%') x GROUP BY 1 ORDER BY 2 DESC;

\echo
\echo '=== SKIPPED, by reason ==='
SELECT left(note, 60) AS reason, count(*) AS tables FROM _rls_results WHERE source = 'skipped' GROUP BY 1 ORDER BY 2 DESC LIMIT 8;

\echo
\echo '=== SAMPLE OF FULL PASSES ==='
SELECT tbl, setting_key, source, seen_a, seen_random, seen_empty, write_cross_tenant, write_control
FROM _rls_results WHERE read_pass AND write_pass ORDER BY source, tbl LIMIT 8;

ROLLBACK;

\echo
\echo 'Rolled back. Leftover probe roles (expect 0):'
SELECT count(*) AS leftover_probe_roles FROM pg_roles WHERE rolname LIKE 'tid\_rls\_probe\_%' ESCAPE '\' AND rolname = :'probe';
