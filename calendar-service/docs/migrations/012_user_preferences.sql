-- ==========================================================================
-- Migration 012: User Notification Preferences
-- Description: Allow users to customize their alert settings
-- ==========================================================================

CREATE TABLE IF NOT EXISTS public.user_notification_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Notification Toggles
    email_sync_complete BOOLEAN DEFAULT TRUE,
    email_sync_failed BOOLEAN DEFAULT TRUE,
    email_conflict_detected BOOLEAN DEFAULT TRUE,
    email_token_expiring BOOLEAN DEFAULT TRUE,
    
    push_sync_complete BOOLEAN DEFAULT FALSE,
    push_sync_failed BOOLEAN DEFAULT TRUE,
    push_conflict_detected BOOLEAN DEFAULT TRUE,
    
    -- Frequency
    digest_frequency VARCHAR(20) DEFAULT 'none' CHECK (digest_frequency IN ('none', 'daily', 'weekly')),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(user_id)
);

-- Enable RLS
ALTER TABLE public.user_notification_settings ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY user_notif_settings_isolation ON public.user_notification_settings
    USING (user_id = NULLIF(current_setting('request.user_id', TRUE), '')::UUID);

CREATE TRIGGER update_user_notif_settings_updated_at
BEFORE UPDATE ON public.user_notification_settings
FOR EACH ROW EXECUTE FUNCTION calendar.update_updated_at_column();
