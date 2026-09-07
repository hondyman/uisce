-- Migration: Add first_name and last_name to public.users view for RBAC compatibility
-- Issue: bp_rbac_users_handler.go queries public.users expecting first_name and last_name columns
-- but the view (created by fix_auth_schema_part2.sql) is based on app_user which doesn't have these columns

-- Add missing columns to app_user table
ALTER TABLE public.app_user
    ADD COLUMN IF NOT EXISTS first_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS last_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Recreate public.users view to include all expected columns
-- This view is used by bp_rbac_users_handler.go and other RBAC code
DROP VIEW IF EXISTS public.users;

CREATE OR REPLACE VIEW public.users AS
SELECT
    id,
    email,
    name,
    COALESCE(first_name, SPLIT_PART(name, ' ', 1)) AS first_name,
    COALESCE(last_name, CASE WHEN POSITION(' ' IN name) > 0 THEN SUBSTR(name, POSITION(' ' IN name) + 1) ELSE '' END) AS last_name,
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

-- +migrate Down

DROP VIEW IF EXISTS public.users;

ALTER TABLE public.app_user
    DROP COLUMN IF EXISTS first_name,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS updated_at;
