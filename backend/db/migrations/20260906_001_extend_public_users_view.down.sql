-- Restore original public.users view (10-column form).
-- Captured from pg_get_viewdef before the up migration ran.
-- DO NOT use DROP VIEW — the 7 binary query sites expect the view to exist
-- and would error on rollback if it were dropped and needed to be re-created.
-- OR REPLACE recreates the view with the original column set.

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
    tenant_id
FROM app_user;
