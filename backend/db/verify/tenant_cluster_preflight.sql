-- Tenant provisioning, cluster side: can this Postgres cluster take a new tenant database?
-- Read-only. Run as the ADMINISTRATOR role the provisioning worker connects with (DB_USER), against
-- any database on the cluster that holds the tenant databases (the region's cluster), for example:
--
--   psql "postgresql://$DB_USER@100.84.50.65:5432/postgres" -X -v ON_ERROR_STOP=1 \
--     -v database='abc_orm' \              -- optional: the database the tenant will get
--     -v role_group='ivy_tenants' \        -- optional: the value of TENANT_DB_ROLE_GROUP, if one is used
--     -f backend/db/verify/tenant_cluster_preflight.sql
--
-- Exit status is non-zero when any check fails, so a pipeline can gate on it. Nothing is changed.
-- The fix for "PUBLIC can connect" is scripts/harden-tenant-cluster.sh (a dry run unless --apply).
-- The alpha side (template marker, product, region) is tenant_template_preflight.sql.

\set ON_ERROR_STOP on

CREATE TEMP TABLE pf (n serial, check_name text NOT NULL, ok boolean NOT NULL, detail text NOT NULL);

-- 1. The administrator can create databases and roles. The saga creates one database and one role per tenant.
INSERT INTO pf (check_name, ok, detail)
SELECT 'admin role can create databases and roles',
       r.rolsuper OR (r.rolcreatedb AND r.rolcreaterole),
       format('%s: superuser=%s createdb=%s createrole=%s', r.rolname, r.rolsuper, r.rolcreatedb, r.rolcreaterole)
FROM pg_roles r WHERE r.rolname = current_user;

-- 2. The isolation probe: a tenant's role must be able to connect to its own database only, so no other
-- database on the cluster may let PUBLIC connect (postgres and template1 included). Any listed here makes
-- provisioning fail at the probe, after the tenant's database and role already exist.
INSERT INTO pf (check_name, ok, detail)
SELECT 'no other database lets PUBLIC connect',
       count(*) = 0,
       CASE WHEN count(*) = 0 THEN 'every connectable database is closed to PUBLIC'
            ELSE 'open to PUBLIC: ' || string_agg(datname, ', ' ORDER BY datname) END
FROM pg_database d
WHERE d.datallowconn
  AND EXISTS (SELECT 1 FROM aclexplode(COALESCE(d.datacl, acldefault('d', d.datdba))) a
              WHERE a.grantee = 0 AND a.privilege_type = 'CONNECT');

-- 3. The role group, when the worker is configured with one (TENANT_DB_ROLE_GROUP): it must already exist,
-- because creating cluster roles is an administrator's decision and the saga will not do it.
\if :{?role_group}
INSERT INTO pf (check_name, ok, detail)
SELECT 'role group exists', EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'role_group'),
       format('TENANT_DB_ROLE_GROUP=%s', :'role_group');
\endif

-- 4. The database the tenant will get does not exist yet: "already exists" is another tenant's or an earlier run's.
\if :{?database}
INSERT INTO pf (check_name, ok, detail)
SELECT 'database name is free', NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'database'),
       format('%s', :'database');
INSERT INTO pf (check_name, ok, detail)
SELECT 'database and role names fit', length(:'database') <= 59 AND :'database' ~ '^[a-z][a-z0-9_]{0,58}$',
       format('%s (%s characters; the role is %s_app)', :'database', length(:'database'), :'database');
\endif

-- 5. pg_hba.conf admits tenant roles over TLS. Only a superuser (or pg_read_all_settings) can read the rules, so
-- this reports a NOTE and does not fail when it cannot see them.
DO $$
DECLARE n integer;
BEGIN
    SELECT count(*) INTO n FROM pg_hba_file_rules
     WHERE type = 'hostssl' AND auth_method IN ('scram-sha-256', 'md5', 'cert') AND error IS NULL;
    INSERT INTO pf (check_name, ok, detail)
    VALUES ('pg_hba.conf has a hostssl rule', n > 0, format('%s hostssl rule(s) with password or certificate auth', n));
EXCEPTION WHEN insufficient_privilege THEN
    INSERT INTO pf (check_name, ok, detail)
    VALUES ('pg_hba.conf has a hostssl rule', true, 'NOTE: not visible to this role; check it as a superuser');
END $$;

\echo '== Tenant cluster preflight'
SELECT CASE WHEN ok THEN 'PASS' ELSE 'FAIL' END AS result, check_name, detail FROM pf ORDER BY n;

-- Fail the run (non-zero exit under ON_ERROR_STOP) when any check failed.
DO $$
DECLARE bad integer;
BEGIN
    SELECT count(*) INTO bad FROM pf WHERE NOT ok;
    IF bad > 0 THEN
        RAISE EXCEPTION 'tenant cluster preflight: % check(s) failed', bad USING ERRCODE = 'check_violation';
    END IF;
END $$;
