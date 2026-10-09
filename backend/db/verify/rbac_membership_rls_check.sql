-- RBAC membership under row-level security: run as the APPLICATION's database role.
--
-- Why this exists: app_user and user_tenant are row-level-security forced. The RBAC
-- handlers read them through the application's pool, with no tenant set on the session
-- unless a transaction sets it. This script shows what the application role actually sees,
-- so the membership check (backend/internal/api/bp_rbac_role_scope.go, authorizeUser) and the
-- user lists (listUsers, listAssignableUsers) can be judged against real policy, not assumed.
--
-- Run against staging with the app's connection string (NOT as a superuser or the table owner,
-- which bypass or see differently):
--
--   psql "$STAGING_APP_URL" -X -v ON_ERROR_STOP=1 \
--     -v tenant='<tenant uuid>' \
--     -v home_member='<user id whose home tenant is <tenant>>' \
--     -v mapped_member='<user id mapped to <tenant> in user_tenant, with no home tenant>' \
--     -v other_user='<user id of a different tenant>' \
--     -v unassigned='<user id with no home tenant and no mapping>' \
--     -f backend/db/verify/rbac_membership_rls_check.sql
--
-- Expected results (a mismatch means the membership check is broken for that case):
--   home_member   -> member = t
--   mapped_member -> member = t   (if f: user_tenant is invisible to the app role; see the policy section)
--   other_user    -> member = f
--   unassigned    -> unassigned_check = t, and listed by the picker query
--
-- The first sections are facts about the session; read them before the membership rows.

\echo '== 1. Who am I, and do I bypass RLS?'
SELECT current_user AS role,
       (SELECT rolsuper FROM pg_roles WHERE rolname = current_user)    AS superuser,
       (SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user) AS bypass_rls;

\echo '== 2. Are RLS flags on, and forced?'
SELECT relname, relrowsecurity AS rls_enabled, relforcerowsecurity AS rls_forced
FROM pg_class
WHERE oid IN ('public.app_user'::regclass, 'public.user_tenant'::regclass);

\echo '== 3. Which policies exist, and what do they allow?'
SELECT polrelid::regclass AS table_name, polname, pg_get_expr(polqual, polrelid) AS using_expr
FROM pg_policy
WHERE polrelid IN ('public.app_user'::regclass, 'public.user_tenant'::regclass);

\echo '== 4. The handlers set no tenant. Inside a transaction that sets it, as the app would:'
BEGIN;
SELECT set_config('uisce.current_tenant', :'tenant', true);
SELECT 'app_user rows visible'    AS what, count(*) AS n FROM app_user;
SELECT 'user_tenant rows visible' AS what, count(*) AS n FROM user_tenant;

\echo '== 5. Membership, as authorizeUser decides it:'
SELECT 'home_member'   AS case, :'home_member'   AS user_id,
       EXISTS (SELECT 1 FROM user_tenant WHERE user_id = :'home_member'   AND tenant_id = :'tenant'::uuid)
    OR EXISTS (SELECT 1 FROM app_user    WHERE id = :'home_member'   AND tenant_id = :'tenant'::uuid) AS member;
SELECT 'mapped_member' AS case, :'mapped_member' AS user_id,
       EXISTS (SELECT 1 FROM user_tenant WHERE user_id = :'mapped_member' AND tenant_id = :'tenant'::uuid)
    OR EXISTS (SELECT 1 FROM app_user    WHERE id = :'mapped_member' AND tenant_id = :'tenant'::uuid) AS member;
SELECT 'other_user'    AS case, :'other_user'    AS user_id,
       EXISTS (SELECT 1 FROM user_tenant WHERE user_id = :'other_user'    AND tenant_id = :'tenant'::uuid)
    OR EXISTS (SELECT 1 FROM app_user    WHERE id = :'other_user'    AND tenant_id = :'tenant'::uuid) AS member;

\echo '== 6. Unassigned, as updateUserTenant and the picker decide it:'
SELECT 'unassigned' AS case, :'unassigned' AS user_id,
       EXISTS (SELECT 1 FROM app_user u WHERE u.id = :'unassigned' AND u.tenant_id IS NULL
               AND NOT EXISTS (SELECT 1 FROM user_tenant ut WHERE ut.user_id = u.id)) AS unassigned_check;
COMMIT;

\echo '== 7. The same checks with NO tenant set (what an unwrapped handler sees):'
SELECT 'app_user rows visible, no tenant set'    AS what, count(*) AS n FROM app_user;
SELECT 'user_tenant rows visible, no tenant set' AS what, count(*) AS n FROM user_tenant;
