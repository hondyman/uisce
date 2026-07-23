-- Ensure required user columns exist (older schemas may be missing them).
ALTER TABLE public.app_user
    ADD COLUMN IF NOT EXISTS role VARCHAR(50),
    ADD COLUMN IF NOT EXISTS username VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS organization VARCHAR(255),
    ADD COLUMN IF NOT EXISTS permissions JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS is_core_admin BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Make sure permissions column has a non-null value.
UPDATE public.app_user SET permissions = COALESCE(permissions, '[]'::jsonb);

-- Ensure a core admin account exists for local development login flows.
-- Ensure a core admin account exists for local development login flows.
-- Use an idempotent upsert into public.app_user, then reference the user row by username
-- for subsequent inserts so we avoid fragile CTE usage which some runners mishandle.

INSERT INTO public.app_user (
    id,
    username,
    email,
    name,
    role,
    organization,
    permissions,
    is_core_admin,
    is_active,
    created_at,
    updated_at
)
VALUES (
    gen_random_uuid()::text,
    'admin@example.com',
    'admin@example.com',
    'SemLayer Admin',
    'steward',
    'SemLayer Platform',
    '[]'::jsonb || jsonb_build_array('read','write','admin','core_admin'),
    true,
    true,
    NOW(),
    NOW()
)
ON CONFLICT (username) DO UPDATE
SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    role = EXCLUDED.role,
    organization = EXCLUDED.organization,
    permissions = EXCLUDED.permissions,
    is_core_admin = true,
    is_active = true,
    updated_at = NOW();





-- +goose Down
WITH target AS (
    SELECT id FROM public.app_user WHERE username = 'admin@example.com'
)
DELETE FROM private_markets_user_auth WHERE user_id IN (SELECT id FROM target);

DELETE FROM public.app_user WHERE username = 'admin@example.com';
