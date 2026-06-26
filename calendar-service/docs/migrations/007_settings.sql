-- User Settings Table
CREATE TABLE IF NOT EXISTS public.user_settings (
    user_id UUID PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Profile
    display_name VARCHAR(255),
    email VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    timezone VARCHAR(100) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    
    -- Sync Settings
    sync_frequency VARCHAR(20) DEFAULT 'hourly',
    auto_sync_enabled BOOLEAN DEFAULT TRUE,
    default_calendar_id UUID,
    sync_conflicts_auto_resolve BOOLEAN DEFAULT FALSE,
    sync_conflicts_strategy VARCHAR(50) DEFAULT 'manual',
    
    -- Notifications
    email_notifications BOOLEAN DEFAULT TRUE,
    push_notifications BOOLEAN DEFAULT FALSE,
    sync_complete_notification BOOLEAN DEFAULT TRUE,
    conflict_notification BOOLEAN DEFAULT TRUE,
    error_notification BOOLEAN DEFAULT TRUE,
    
    -- Privacy
    data_retention_days INT DEFAULT 365,
    share_analytics BOOLEAN DEFAULT FALSE,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE public.user_settings ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_settings_tenant_isolation 
ON public.user_settings
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY user_settings_user_access 
ON public.user_settings
FOR SELECT
USING (user_id = NULLIF(current_setting('request.user_id', TRUE), '')::UUID OR 
       current_setting('request.is_admin', TRUE) = 'true');

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_settings_tenant ON public.user_settings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_settings_email ON public.user_settings(email);

-- Trigger for updated_at
CREATE TRIGGER update_user_settings_updated_at
BEFORE UPDATE ON public.user_settings
FOR EACH ROW
EXECUTE FUNCTION calendar.update_updated_at_column();
