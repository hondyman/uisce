-- ============================================================================
-- Migration 016: SSO Integration
-- ============================================================================
-- Purpose: Support enterprise SSO (SAML/OIDC)
-- Deploy: psql $DB_URL -f docs/migrations/016_sso.sql
-- ============================================================================

-- Create SSO providers table
CREATE TABLE IF NOT EXISTS public.sso_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Provider info
    provider_type VARCHAR(50) NOT NULL CHECK (provider_type IN ('saml', 'oidc', 'oauth')),
    provider_name VARCHAR(100) NOT NULL, -- Okta, Azure AD, OneLogin, etc.
    
    -- SAML configuration
    saml_entity_id VARCHAR(500),
    saml_sso_url VARCHAR(500),
    saml_certificate TEXT,
    saml_private_key TEXT,
    
    -- OIDC configuration
    oidc_issuer VARCHAR(500),
    oidc_client_id VARCHAR(255),
    oidc_client_secret VARCHAR(255),
    oidc_redirect_uri VARCHAR(500),
    oidc_scopes TEXT[],
    
    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    is_primary BOOLEAN DEFAULT FALSE,
    auto_provision_users BOOLEAN DEFAULT TRUE,
    default_user_role VARCHAR(50) DEFAULT 'user',
    
    -- Metadata
    metadata_url VARCHAR(500), -- For SAML metadata discovery
    last_synced_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, provider_type, provider_name)
);

-- Create SSO sessions table
CREATE TABLE IF NOT EXISTS public.sso_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES public.users(id) ON DELETE CASCADE,
    sso_provider_id UUID REFERENCES public.sso_providers(id) ON DELETE SET NULL,
    
    -- Session info
    session_id VARCHAR(255) NOT NULL UNIQUE,
    idp_user_id VARCHAR(255), -- User ID from IdP
    idp_email VARCHAR(255),
    idp_attributes JSONB,
    
    -- SAML specific
    saml_assertion_id VARCHAR(255),
    saml_name_id VARCHAR(255),
    
    -- OIDC specific
    oidc_id_token TEXT,
    oidc_access_token TEXT,
    oidc_refresh_token TEXT,
    oidc_token_expires_at TIMESTAMPTZ,
    
    -- Session state
    authenticated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create team invitations table (for SSO auto-provisioning)
CREATE TABLE IF NOT EXISTS public.team_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    team_id UUID REFERENCES public.teams(id) ON DELETE CASCADE,
    
    -- Invitation info
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'member',
    invited_by UUID REFERENCES public.users(id),
    
    -- SSO specific
    sso_provider_id UUID REFERENCES public.sso_providers(id),
    idp_user_id VARCHAR(255),
    
    -- State
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired', 'cancelled')),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, email)
);

-- Indexes
CREATE INDEX idx_sso_providers_tenant ON public.sso_providers(tenant_id, is_active);
CREATE INDEX idx_sso_sessions_session ON public.sso_sessions(session_id);
CREATE INDEX idx_sso_sessions_user ON public.sso_sessions(user_id, expires_at);
CREATE INDEX idx_sso_sessions_idp ON public.sso_sessions(idp_user_id, sso_provider_id);
CREATE INDEX idx_team_invitations_email ON public.team_invitations(email, status);

-- Enable RLS
ALTER TABLE public.sso_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.sso_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.team_invitations ENABLE ROW LEVEL SECURITY;

CREATE POLICY sso_providers_tenant_isolation 
ON public.sso_providers
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY sso_sessions_tenant_isolation 
ON public.sso_sessions
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY team_invitations_tenant_isolation 
ON public.team_invitations
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- View for SSO configuration
CREATE OR REPLACE VIEW public.sso_config AS
SELECT 
    sp.id,
    sp.tenant_id,
    sp.provider_type,
    sp.provider_name,
    sp.is_active,
    sp.is_primary,
    sp.auto_provision_users,
    sp.default_user_role,
    CASE 
        WHEN sp.provider_type = 'saml' THEN sp.saml_sso_url
        WHEN sp.provider_type = 'oidc' THEN sp.oidc_issuer
    END as issuer_url,
    sp.created_at
FROM public.sso_providers sp
WHERE sp.is_active = TRUE;

-- Function to validate SSO session
CREATE OR REPLACE FUNCTION public.validate_sso_session(p_session_id VARCHAR)
RETURNS TABLE (
    valid BOOLEAN,
    user_id UUID,
    tenant_id UUID,
    expires_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ss.expires_at > NOW() as valid,
        ss.user_id,
        ss.tenant_id,
        ss.expires_at
    FROM public.sso_sessions ss
    WHERE ss.session_id = p_session_id;
END;
$$ LANGUAGE plpgsql;

-- Comment columns
COMMENT ON COLUMN public.sso_providers.auto_provision_users IS 'Automatically create user accounts on first SSO login';
COMMENT ON COLUMN public.sso_sessions.idp_attributes IS 'Additional attributes from IdP (groups, roles, etc.)';
