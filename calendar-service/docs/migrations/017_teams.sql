-- ============================================================================
-- Migration 017: Team/Workspace Features
-- ============================================================================
-- Purpose: Support team collaboration and shared calendars
-- Deploy: psql $DB_URL -f docs/migrations/017_teams.sql
-- ============================================================================

-- Enhance teams table with more features
ALTER TABLE public.teams 
ADD COLUMN IF NOT EXISTS slug VARCHAR(100) UNIQUE,
ADD COLUMN IF NOT EXISTS avatar_url TEXT,
ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS billing_email VARCHAR(255),
ADD COLUMN IF NOT EXISTS subscription_tier VARCHAR(50) DEFAULT 'free'
    CHECK (subscription_tier IN ('free', 'pro', 'enterprise'));

-- Create team roles table
CREATE TABLE IF NOT EXISTS public.team_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES public.teams(id) ON DELETE CASCADE,
    role_name VARCHAR(50) NOT NULL,
    
    -- Permissions
    permissions JSONB NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(team_id, role_name)
);

-- Default roles
INSERT INTO public.team_roles (team_id, role_name, permissions)
SELECT 
    t.id,
    role_name,
    permissions
FROM public.teams t
CROSS JOIN (
    VALUES 
        ('owner', '{"calendars": ["create", "read", "update", "delete"], "members": ["create", "read", "update", "delete"], "settings": ["read", "update"], "billing": ["read", "update"]}'::jsonb),
        ('admin', '{"calendars": ["create", "read", "update", "delete"], "members": ["create", "read", "update"], "settings": ["read", "update"], "billing": ["read"]}'::jsonb),
        ('member', '{"calendars": ["create", "read", "update"], "members": ["read"], "settings": ["read"], "billing": []}'::jsonb),
        ('viewer', '{"calendars": ["read"], "members": ["read"], "settings": [], "billing": []}'::jsonb)
) AS roles(role_name, permissions)
ON CONFLICT DO NOTHING;

-- Create shared calendars table
CREATE TABLE IF NOT EXISTS public.shared_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES public.teams(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    
    -- Calendar info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#10b981',
    is_team_calendar BOOLEAN DEFAULT FALSE,
    
    -- Sharing settings
    visibility VARCHAR(20) DEFAULT 'team' CHECK (visibility IN ('private', 'team', 'public')),
    allow_external_sharing BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create calendar sharing permissions table
CREATE TABLE IF NOT EXISTS public.calendar_sharing_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID NOT NULL REFERENCES public.shared_calendars(id) ON DELETE CASCADE,
    
    -- Permission target
    user_id UUID REFERENCES public.users(id) ON DELETE CASCADE,
    team_id UUID REFERENCES public.teams(id) ON DELETE CASCADE,
    email VARCHAR(255), -- For external sharing
    
    -- Permissions
    can_view BOOLEAN DEFAULT TRUE,
    can_edit BOOLEAN DEFAULT FALSE,
    can_share BOOLEAN DEFAULT FALSE,
    can_delete BOOLEAN DEFAULT FALSE,
    
    -- Metadata
    granted_by UUID REFERENCES public.users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    
    UNIQUE(calendar_id, user_id, team_id, email)
);

-- Create team activity log table
CREATE TABLE IF NOT EXISTS public.team_activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES public.teams(id) ON DELETE CASCADE,
    user_id UUID REFERENCES public.users(id) ON DELETE SET NULL,
    
    -- Activity info
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    metadata JSONB DEFAULT '{}',
    
    ip_address INET,
    user_agent TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_teams_slug ON public.teams(slug);
CREATE INDEX idx_team_roles_team ON public.team_roles(team_id);
CREATE INDEX idx_shared_calendars_team ON public.shared_calendars(team_id);
CREATE INDEX idx_calendar_sharing_permissions_calendar ON public.calendar_sharing_permissions(calendar_id);
CREATE INDEX idx_team_activity_log_team ON public.team_activity_log(team_id, created_at DESC);

-- Enable RLS
ALTER TABLE public.team_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.shared_calendars ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.calendar_sharing_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.team_activity_log ENABLE ROW LEVEL SECURITY;

-- Views
CREATE OR REPLACE VIEW public.team_members_with_roles AS
SELECT 
    tm.id,
    tm.team_id,
    tm.user_id,
    tm.role,
    tm.created_at as joined_at,
    u.email,
    u.username as display_name,
    NULL::text as avatar_url,
    tr.permissions
FROM public.team_members tm
JOIN public.users u ON tm.user_id = u.id
LEFT JOIN public.team_roles tr ON tm.team_id = tr.team_id AND tm.role = tr.role_name;

CREATE OR REPLACE VIEW public.team_calendars AS
SELECT 
    sc.id,
    sc.team_id,
    sc.owner_id,
    sc.name,
    sc.description,
    sc.color,
    sc.is_team_calendar,
    sc.visibility,
    sc.created_at,
    u.username as owner_name,
    COUNT(csp.id) as shared_with_count
FROM public.shared_calendars sc
JOIN public.users u ON sc.owner_id = u.id
LEFT JOIN public.calendar_sharing_permissions csp ON sc.id = csp.calendar_id
GROUP BY sc.id, sc.team_id, sc.owner_id, sc.name, sc.description, sc.color, sc.is_team_calendar, sc.visibility, sc.created_at, u.username;

-- Comment columns
COMMENT ON COLUMN public.teams.subscription_tier IS 'Team subscription tier for billing';
COMMENT ON COLUMN public.shared_calendars.visibility IS 'Who can see this calendar';
COMMENT ON COLUMN public.team_activity_log.action IS 'Action performed (e.g., calendar.created, member.invited)';
