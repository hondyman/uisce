-- Extend public.users view to include columns the binary queries.
-- The view was created in the database (not via migrations) and is being adopted
-- into the managed migration chain here. Provenance: definition captured
-- 2026-09-06 via pg_get_viewdef, source: live DB, not migrations.
--
-- The underlying app_user table has 23 columns. The original view selected 10.
-- The binary makes 7 distinct queries against public.users, all with explicit
-- column lists (position-safe for OR REPLACE). Four columns are missing:
--   role, organization, permissions, is_core_admin
--
-- app_user column order: id, email, display_name, created_at, is_active, name,
--   role, username, organization, permissions, is_core_admin, ...
-- New columns appended after the original 10 to preserve SELECT * callers.

CREATE OR REPLACE VIEW public.users AS
SELECT
    id,
    username,
    email,
    COALESCE(name, display_name::character varying) AS name,
    first_name,
    last_name,
    COALESCE(status, 'active'::text) AS status,
    is_active,
    created_at,
    tenant_id,
    role,
    organization,
    COALESCE(permissions, '[]'::jsonb) AS permissions,
    COALESCE(is_core_admin, false) AS is_core_admin
FROM app_user;
