-- Fix Auth Schema to match Backend Code

-- 1. Update app_user with missing columns expected by auth_handlers.go
ALTER TABLE public.app_user
    ADD COLUMN IF NOT EXISTS name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS role VARCHAR(50),
    ADD COLUMN IF NOT EXISTS username VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS organization VARCHAR(255),
    ADD COLUMN IF NOT EXISTS permissions JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS is_core_admin BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS password_hash TEXT,
    ADD COLUMN IF NOT EXISTS salt TEXT,
    ADD COLUMN IF NOT EXISTS tenant_id TEXT; -- Referenced in auth_handlers query

-- Backfill name from display_name if exists
UPDATE public.app_user SET name = display_name WHERE name IS NULL;

-- 2. Create private_markets_sessions table (as expected by auth_handlers.go)
CREATE TABLE IF NOT EXISTS public.private_markets_sessions (
    user_id TEXT NOT NULL, -- References app_user(id) but usually no FK for flexibility in sessions
    session_token TEXT PRIMARY KEY,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON public.private_markets_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh ON public.private_markets_sessions(refresh_token);

-- 3. Create public.users VIEW to support existing code referencing it
-- This avoids modifying the Go code significantly if we prefer to keep "public.users" in query strings
CREATE OR REPLACE VIEW public.users AS
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
    salt
FROM public.app_user;

-- 4. Seed Admin User
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
    created_at
)
VALUES (
    '36f45238-bac6-4b06-a495-6155c43df552', -- Fixed ID for consistency
    'admin@example.com',
    'admin@example.com',
    'SemLayer Admin',
    'global_admin',
    'uisce',
    '["read","write","admin","global_admin"]'::jsonb,
    true,
    true,
    -- bcrypt hash for "password123"
    '$2a$10$ZyLGQ5MY8mjhILLIuQIIcuIhtFuh2sUlNUzCKr4sAkMSoz69cB5bC', 
    NULL,
    NOW()
)
ON CONFLICT (email) DO UPDATE
SET
    password_hash = '$2a$10$ZyLGQ5MY8mjhILLIuQIIcuIhtFuh2sUlNUzCKr4sAkMSoz69cB5bC', -- password123
    is_core_admin = true,
    role = 'global_admin';

-- NOTE: The hash above is a placeholder. I will use a real hash in the actual run.
