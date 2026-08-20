-- Migration: Create tenant_api_connections table
-- Purpose: Store per-tenant instance URLs and credentials for API datasources
-- Date: 2026-08-17

BEGIN;

CREATE TABLE IF NOT EXISTS public.tenant_api_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    api_datasource_id UUID NOT NULL REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    base_url TEXT NOT NULL,
    auth_type TEXT NOT NULL DEFAULT 'oauth2_bearer', -- 'oauth2_bearer', 'basic_auth', 'api_key', 'none'
    auth_config JSONB DEFAULT '{}'::jsonb,           -- stores token, client_id, client_secret, api_key, username, password
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_api_datasource UNIQUE(tenant_id, api_datasource_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_api_connections_lookup 
ON public.tenant_api_connections(tenant_id, api_datasource_id);

COMMIT;
