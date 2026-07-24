-- Seed Test User (test@example.com) for Frontend Connectivity Check

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
    password_hash,
    tenant_id,
    created_at,
    updated_at,
    last_login,
    status
)
VALUES (
    '36f45238-bac6-4b06-a495-6155c43df551', -- Different UUID from admin
    'test@example.com',
    'test@example.com',
    'Test User',
    'user',
    'Default Organization',
    '["read"]'::jsonb,
    false,
    true,
    -- bcrypt hash for "password123" ($2a$10$ZyLGQ5MY8mjhILLIuQIIcuIhtFuh2sUlNUzCKr4sAkMSoz69cB5bC)
    '$2a$10$ZyLGQ5MY8mjhILLIuQIIcuIhtFuh2sUlNUzCKr4sAkMSoz69cB5bC',
    NULL,
    NOW(),
    NOW(),
    NULL,
    'active'
)
ON CONFLICT (email) DO UPDATE
SET
    password_hash = '$2a$10$ZyLGQ5MY8mjhILLIuQIIcuIhtFuh2sUlNUzCKr4sAkMSoz69cB5bC',
    status = 'active',
    is_active = true;
