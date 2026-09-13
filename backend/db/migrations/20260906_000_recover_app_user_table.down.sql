-- CASCADE because public.users (20260906_001_extend_public_users_view.up.sql)
-- is a view defined as `FROM app_user` — dropping the table without cascade
-- would fail with a dependency error.
DROP TABLE IF EXISTS public.app_user CASCADE;
