-- Add missing columns for full compatibility with auth_handlers.go

ALTER TABLE public.app_user
    ADD COLUMN IF NOT EXISTS last_login TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Recreate view to include new columns
DROP VIEW IF EXISTS public.users;

CREATE VIEW public.users AS
SELECT 
    id,
    email,
    name,
    role,
    organization,
    permissions,
    is_core_admin,
    is_active,
    password_hash,
    tenant_id,
    username,
    display_name,
    created_at,
    updated_at,
    salt,
    last_login,
    status
FROM public.app_user;
