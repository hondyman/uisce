-- Migration: 20261019_001_workspace_layout_preferences.up.sql
-- Purpose: Persist roaming multi-tenant user workspace layout profiles in PostgreSQL.
-- Enables seamless cross-device layout restoration and travel mode transitions.

CREATE TABLE IF NOT EXISTS public.user_workspace_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    profile_name VARCHAR(128) NOT NULL DEFAULT 'default',
    schema_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    layout_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_workspace_layout UNIQUE (tenant_id, user_id, profile_name)
);

CREATE INDEX IF NOT EXISTS idx_user_workspace_layouts_lookup 
    ON public.user_workspace_layouts (tenant_id, user_id);
