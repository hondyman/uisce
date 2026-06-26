-- ==========================================================================
-- Migration 010: Teams and Workspaces
-- Description: Multi-tenant team management and workspace isolation
-- ==========================================================================

-- Teams table
CREATE TABLE IF NOT EXISTS public.teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Team Members table
CREATE TABLE IF NOT EXISTS public.team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES public.teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);

-- Workspaces table
CREATE TABLE IF NOT EXISTS public.workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    team_id UUID REFERENCES public.teams(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    config JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

-- Indices
CREATE INDEX IF NOT EXISTS idx_teams_tenant ON public.teams(tenant_id);
CREATE INDEX IF NOT EXISTS idx_team_members_team ON public.team_members(team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user ON public.team_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_tenant ON public.workspaces(tenant_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_team ON public.workspaces(team_id);

-- Enable RLS
ALTER TABLE public.teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.team_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.workspaces ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY team_tenant_isolation ON public.teams
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY team_member_tenant_isolation ON public.team_members
    USING (EXISTS (
        SELECT 1 FROM public.teams t 
        WHERE t.id = team_id 
        AND t.tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID
    ));

CREATE POLICY workspace_tenant_isolation ON public.workspaces
    USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- Triggers for updated_at
CREATE TRIGGER update_teams_updated_at
BEFORE UPDATE ON public.teams
FOR EACH ROW EXECUTE FUNCTION calendar.update_updated_at_column();

CREATE TRIGGER update_team_members_updated_at
BEFORE UPDATE ON public.team_members
FOR EACH ROW EXECUTE FUNCTION calendar.update_updated_at_column();

CREATE TRIGGER update_workspaces_updated_at
BEFORE UPDATE ON public.workspaces
FOR EACH ROW EXECUTE FUNCTION calendar.update_updated_at_column();
