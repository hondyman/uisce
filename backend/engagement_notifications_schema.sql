-- Engagement Notification System Schema
-- Extends the existing notification system with advanced engagement features

-- Enhanced notification table with engagement tracking
CREATE TABLE IF NOT EXISTS engagement_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- welcome, feature, recommendation, alert, campaign
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    rich_content JSONB,
    priority INTEGER DEFAULT 2, -- 1=low, 2=normal, 3=high, 4=critical
    channels TEXT[] NOT NULL DEFAULT ARRAY['in_app'], -- email, sms, push, in_app
    status VARCHAR(50) DEFAULT 'draft', -- draft, scheduled, sent, delivered, read, clicked, dismissed
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Engagement tracking
    engagement_score DECIMAL(5,4) DEFAULT 0,
    user_segment VARCHAR(100),
    ab_test_variant VARCHAR(100),

    -- Template and personalization
    template_id VARCHAR(255),
    personalization JSONB,

    -- Actions and CTAs
    actions JSONB,
    cta JSONB
);

-- Notification templates for reusable content
CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    subject VARCHAR(500),
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    rich_content JSONB,
    variables TEXT[],
    channels TEXT[] NOT NULL DEFAULT ARRAY['in_app'],
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User notification preferences
CREATE TABLE IF NOT EXISTS user_notification_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    email_enabled BOOLEAN DEFAULT true,
    sms_enabled BOOLEAN DEFAULT false,
    push_enabled BOOLEAN DEFAULT true,
    in_app_enabled BOOLEAN DEFAULT true,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    timezone VARCHAR(100) DEFAULT 'UTC',
    channel_preferences JSONB DEFAULT '{}',
    type_preferences JSONB DEFAULT '{}',
    frequency_preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification campaigns for automated sequences
CREATE TABLE IF NOT EXISTS notification_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(100) NOT NULL, -- onboarding, feature_adoption, re_engagement
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, paused, completed
    target_users TEXT[],
    user_segment VARCHAR(100),
    steps JSONB NOT NULL DEFAULT '[]',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification analytics for engagement tracking
CREATE TABLE IF NOT EXISTS notification_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- sent, delivered, opened, clicked, dismissed
    event_timestamp TIMESTAMPTZ DEFAULT NOW(),
    user_agent TEXT,
    ip_address INET,
    device_type VARCHAR(100),
    location VARCHAR(255),
    session_id VARCHAR(255),
    additional_metadata JSONB
);

-- User engagement profiles
CREATE TABLE IF NOT EXISTS user_engagement_profiles (
    user_id VARCHAR(255) PRIMARY KEY,
    total_notifications INTEGER DEFAULT 0,
    opened_notifications INTEGER DEFAULT 0,
    clicked_notifications INTEGER DEFAULT 0,
    dismissed_notifications INTEGER DEFAULT 0,
    avg_open_rate DECIMAL(5,4) DEFAULT 0,
    avg_click_rate DECIMAL(5,4) DEFAULT 0,
    last_activity TIMESTAMPTZ DEFAULT NOW(),
    engagement_score DECIMAL(5,4) DEFAULT 0,
    segment VARCHAR(100) DEFAULT 'new_user', -- highly_engaged, moderately_engaged, low_engaged, inactive
    preferred_channels TEXT[] DEFAULT ARRAY['in_app'],
    preferred_times TEXT[] DEFAULT ARRAY['morning', 'afternoon'],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_engagement_notifications_user_id ON engagement_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_engagement_notifications_status ON engagement_notifications(status);
CREATE INDEX IF NOT EXISTS idx_engagement_notifications_type ON engagement_notifications(type);
CREATE INDEX IF NOT EXISTS idx_engagement_notifications_scheduled_at ON engagement_notifications(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_engagement_notifications_created_at ON engagement_notifications(created_at);

CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates(type);

CREATE INDEX IF NOT EXISTS idx_notification_campaigns_status ON notification_campaigns(status);
CREATE INDEX IF NOT EXISTS idx_notification_campaigns_type ON notification_campaigns(type);

CREATE INDEX IF NOT EXISTS idx_notification_analytics_notification_id ON notification_analytics(notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_analytics_user_id ON notification_analytics(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_analytics_event_type ON notification_analytics(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_analytics_timestamp ON notification_analytics(event_timestamp);

CREATE INDEX IF NOT EXISTS idx_user_engagement_profiles_segment ON user_engagement_profiles(segment);
CREATE INDEX IF NOT EXISTS idx_user_engagement_profiles_score ON user_engagement_profiles(engagement_score);

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_engagement_notifications_updated_at
    BEFORE UPDATE ON engagement_notifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_templates_updated_at
    BEFORE UPDATE ON notification_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_notification_preferences_updated_at
    BEFORE UPDATE ON user_notification_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_campaigns_updated_at
    BEFORE UPDATE ON notification_campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_engagement_profiles_updated_at
    BEFORE UPDATE ON user_engagement_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
