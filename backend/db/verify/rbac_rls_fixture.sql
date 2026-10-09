-- Fixture for the RBAC membership test under row-level security (internal/api/bp_rbac_rls_test.go).
--
-- Run as the OWNER of a scratch database (not against staging, and not as the restricted
-- role). It loads four users across two tenants and turns on forced row-level security on
-- the two tables, with the same tenant policy shape app_user uses, then grants read to the
-- restricted role the test connects as.
--
--   createdb uisce_rbac_rls
--   psql "postgresql://postgres@localhost/uisce_rbac_rls" -f db/verify/rbac_rls_fixture.sql
--
-- Then run the test as the restricted role:
--   UISCE_RBAC_RLS_DSN="postgres://app_user_test:...@localhost/uisce_rbac_rls?sslmode=disable" \
--     go test ./internal/api/ -run TestRBACMembership_RestrictedRole -v
--
-- The restricted role here is app_user_test: a non-superuser without BYPASSRLS, the same
-- kind of role the application's database login must be for this to mean anything.

DROP TABLE IF EXISTS user_tenant;
DROP VIEW IF EXISTS users;
DROP TABLE IF EXISTS app_user;

CREATE TABLE app_user (
    id        text PRIMARY KEY,
    tenant_id uuid,
    username  text,
    email     text,
    name      text,
    is_active boolean NOT NULL DEFAULT true
);
CREATE TABLE user_tenant (
    user_id   text NOT NULL,
    tenant_id uuid NOT NULL,
    access_role varchar(50) DEFAULT 'viewer',
    PRIMARY KEY (user_id, tenant_id)
);
CREATE VIEW users AS
    SELECT id, username, email, name, NULL::text AS first_name, NULL::text AS last_name,
           'active'::text AS status, is_active, now() AS created_at, tenant_id
    FROM app_user;

-- Rows are loaded before row-level security is forced, so the owner can insert them.
INSERT INTO app_user (id, tenant_id, username, email, name) VALUES
    ('home-member',   '11111111-1111-4111-8111-111111111111', 'ann', 'ann@example.com', 'Ann'),
    ('mapped-member', NULL,                                   'dee', 'dee@example.com', 'Dee'),
    ('other-tenant',  '22222222-2222-4222-8222-222222222222', 'bob', 'bob@example.com', 'Bob'),
    ('unassigned',    NULL,                                   'cy',  'cy@example.com',  'Cy');
INSERT INTO user_tenant (user_id, tenant_id) VALUES
    ('mapped-member', '11111111-1111-4111-8111-111111111111'),
    ('other-tenant',  '22222222-2222-4222-8222-222222222222');

ALTER TABLE app_user ENABLE ROW LEVEL SECURITY;
ALTER TABLE app_user FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON app_user
    USING ((tenant_id)::text = current_setting('uisce.current_tenant', true));

-- user_tenant is forced with NO policy, as far as the migrations in this repo show.
ALTER TABLE user_tenant ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_tenant FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user_test') THEN
        GRANT USAGE ON SCHEMA public TO app_user_test;
        GRANT SELECT ON app_user, user_tenant, users TO app_user_test;
    END IF;
END $$;
