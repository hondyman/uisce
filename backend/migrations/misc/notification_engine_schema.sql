-- ============================================================================
-- Advanced Notification Engine Schema
-- Run with: psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -f backend/migrations/misc/notification_engine_schema.sql
-- ============================================================================

-- ============================================================================
-- NOTIFICATION TEMPLATES
-- ============================================================================

CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Template identification
    template_key VARCHAR(100) NOT NULL,
    template_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50), -- approval, reminder, alert, info, escalation
    
    -- Content
    subject_template TEXT NOT NULL,
    body_template TEXT NOT NULL,
    template_variables TEXT[], -- List of available variables: {user_name}, {process_name}, etc.
    
    -- Channels
    enabled_channels TEXT[], -- email, sms, slack, teams, push
    default_channel VARCHAR(50) DEFAULT 'email',
    
    -- Conditional rules (JSONB for flexibility)
    send_conditions JSONB, -- {field: "priority", operator: "==", value: "high"}
    
    -- Scheduling
    send_delay_minutes INTEGER DEFAULT 0, -- Delay before sending
    digest_mode VARCHAR(50) DEFAULT 'immediate', -- immediate, hourly, daily, weekly
    
    -- Escalation
    escalation_enabled BOOLEAN DEFAULT false,
    escalation_delay_minutes INTEGER, -- Resend if no response
    escalation_recipient_roles TEXT[], -- Roles to escalate to
    
    -- Metadata
    is_system BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high, urgent
    
    -- Rich content
    include_attachments BOOLEAN DEFAULT false,
    include_quick_actions BOOLEAN DEFAULT false, -- Buttons in notification
    quick_actions JSONB, -- [{label: "Approve", action: "approve_step"}]
    
    -- Timestamps
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, datasource_id, template_key)
);

-- Indexes for notification templates
CREATE INDEX IF NOT EXISTS idx_notif_templates_tenant ON notification_templates(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_notif_templates_key ON notification_templates(template_key);
CREATE INDEX IF NOT EXISTS idx_notif_templates_category ON notification_templates(category, is_active);
CREATE INDEX IF NOT EXISTS idx_notif_templates_digest ON notification_templates(digest_mode) WHERE digest_mode != 'immediate';

-- ============================================================================
-- USER NOTIFICATION PREFERENCES
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    
    -- Channel preferences
    email_enabled BOOLEAN DEFAULT true,
    email_address VARCHAR(255),
    
    sms_enabled BOOLEAN DEFAULT false,
    phone_number VARCHAR(50),
    
    slack_enabled BOOLEAN DEFAULT false,
    slack_user_id VARCHAR(255),
    slack_webhook_url TEXT,
    
    teams_enabled BOOLEAN DEFAULT false,
    teams_user_id VARCHAR(255),
    teams_webhook_url TEXT,
    
    push_enabled BOOLEAN DEFAULT true,
    push_token TEXT,
    
    -- Digest preferences
    digest_mode VARCHAR(50) DEFAULT 'immediate', -- immediate, hourly, daily, weekly
    digest_time TIME, -- Preferred time for daily digest (e.g., 9:00 AM)
    digest_days INTEGER[], -- Days of week for weekly digest (1=Mon, 7=Sun)
    
    -- Content preferences
    include_summary BOOLEAN DEFAULT true,
    include_full_details BOOLEAN DEFAULT false,
    
    -- Do Not Disturb
    dnd_enabled BOOLEAN DEFAULT false,
    dnd_start_time TIME,
    dnd_end_time TIME,
    
    -- Priority filtering
    min_priority VARCHAR(20) DEFAULT 'low', -- Only send notifications of this priority or higher
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, datasource_id, user_id)
);

-- Indexes for user preferences
CREATE INDEX IF NOT EXISTS idx_notif_prefs_user ON user_notification_preferences(tenant_id, datasource_id, user_id);
CREATE INDEX IF NOT EXISTS idx_notif_prefs_digest ON user_notification_preferences(digest_mode) WHERE digest_mode != 'immediate';

-- ============================================================================
-- NOTIFICATION DELIVERY LOGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Template reference
    template_id UUID REFERENCES notification_templates(id) ON DELETE SET NULL,
    template_key VARCHAR(100),
    
    -- Recipient
    recipient_user_id VARCHAR(255) NOT NULL,
    recipient_email VARCHAR(255),
    recipient_phone VARCHAR(50),
    
    -- Content
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    rendered_content JSONB, -- Final rendered content with variables replaced
    
    -- Delivery
    channel VARCHAR(50) NOT NULL, -- email, sms, slack, teams, push
    status VARCHAR(50) NOT NULL, -- pending, sent, delivered, failed, bounced
    delivery_provider VARCHAR(100), -- sendgrid, twilio, slack-api, etc.
    
    -- Tracking
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    opened_at TIMESTAMP,
    clicked_at TIMESTAMP,
    
    -- Response tracking (for actions like approve/reject)
    action_taken VARCHAR(100),
    action_taken_at TIMESTAMP,
    
    -- Error handling
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    next_retry_at TIMESTAMP,
    
    -- Context
    process_id UUID,
    process_instance_id UUID,
    step_id UUID,
    related_entity_type VARCHAR(100),
    related_entity_id VARCHAR(255),
    
    -- Metadata
    priority VARCHAR(20) DEFAULT 'normal',
    is_digest BOOLEAN DEFAULT false,
    digest_batch_id UUID, -- Groups notifications sent as digest
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for notification logs
CREATE INDEX IF NOT EXISTS idx_notif_logs_tenant ON notification_logs(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_notif_logs_recipient ON notification_logs(recipient_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_logs_status ON notification_logs(status, next_retry_at);
CREATE INDEX IF NOT EXISTS idx_notif_logs_template ON notification_logs(template_id);
CREATE INDEX IF NOT EXISTS idx_notif_logs_process ON notification_logs(process_id, process_instance_id);
CREATE INDEX IF NOT EXISTS idx_notif_logs_digest ON notification_logs(digest_batch_id) WHERE digest_batch_id IS NOT NULL;

-- ============================================================================
-- NOTIFICATION DIGESTS (Pending notifications to batch)
-- ============================================================================

CREATE TABLE IF NOT EXISTS notification_digests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Recipient
    recipient_user_id VARCHAR(255) NOT NULL,
    
    -- Batch info
    digest_period VARCHAR(50) NOT NULL, -- hourly, daily, weekly
    notification_count INTEGER DEFAULT 0,
    notification_ids UUID[], -- Array of notification_log IDs
    
    -- Scheduling
    scheduled_send_at TIMESTAMP NOT NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, cancelled
    sent_at TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for digests
CREATE INDEX IF NOT EXISTS idx_notif_digests_schedule ON notification_digests(scheduled_send_at, status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_notif_digests_recipient ON notification_digests(recipient_user_id, status);

-- ============================================================================
-- ESCALATION TRACKING
-- ============================================================================

CREATE TABLE IF NOT EXISTS notification_escalations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Original notification
    original_notification_id UUID REFERENCES notification_logs(id) ON DELETE CASCADE,
    template_id UUID REFERENCES notification_templates(id) ON DELETE SET NULL,
    
    -- Escalation details
    escalation_level INTEGER DEFAULT 1, -- 1st reminder, 2nd reminder, etc.
    escalation_recipient_role VARCHAR(100),
    escalation_recipient_user_id VARCHAR(255),
    
    -- Timing
    triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    escalation_notification_id UUID REFERENCES notification_logs(id) ON DELETE SET NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, resolved, cancelled
    resolved_at TIMESTAMP,
    resolution_action VARCHAR(100), -- User took action, manual resolution, etc.
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for escalations
CREATE INDEX IF NOT EXISTS idx_notif_escalations_original ON notification_escalations(original_notification_id);
CREATE INDEX IF NOT EXISTS idx_notif_escalations_status ON notification_escalations(status, triggered_at);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update timestamp trigger function
CREATE OR REPLACE FUNCTION update_notification_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
DROP TRIGGER IF EXISTS notification_templates_updated ON notification_templates;
CREATE TRIGGER notification_templates_updated
    BEFORE UPDATE ON notification_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_timestamp();

DROP TRIGGER IF EXISTS user_notification_preferences_updated ON user_notification_preferences;
CREATE TRIGGER user_notification_preferences_updated
    BEFORE UPDATE ON user_notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_timestamp();

DROP TRIGGER IF EXISTS notification_logs_updated ON notification_logs;
CREATE TRIGGER notification_logs_updated
    BEFORE UPDATE ON notification_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_timestamp();

DROP TRIGGER IF EXISTS notification_digests_updated ON notification_digests;
CREATE TRIGGER notification_digests_updated
    BEFORE UPDATE ON notification_digests
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_timestamp();

DROP TRIGGER IF EXISTS notification_escalations_updated ON notification_escalations;
CREATE TRIGGER notification_escalations_updated
    BEFORE UPDATE ON notification_escalations
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_timestamp();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE notification_templates IS 'Reusable notification templates with multi-channel support and conditional rules';
COMMENT ON TABLE user_notification_preferences IS 'Per-user notification preferences including channels, digest mode, and DND settings';
COMMENT ON TABLE notification_logs IS 'Complete audit trail of all notifications sent with delivery tracking';
COMMENT ON TABLE notification_digests IS 'Batched notifications pending delivery in digest mode';
COMMENT ON TABLE notification_escalations IS 'Escalation tracking for notifications requiring follow-up';

COMMENT ON COLUMN notification_templates.template_variables IS 'Available placeholders like {user_name}, {process_name}, {due_date}';
COMMENT ON COLUMN notification_templates.send_conditions IS 'JSON rules for conditional sending based on context';
COMMENT ON COLUMN notification_templates.quick_actions IS 'Interactive buttons/actions embedded in notification';
COMMENT ON COLUMN notification_logs.rendered_content IS 'Final notification content with all variables replaced';
COMMENT ON COLUMN user_notification_preferences.digest_time IS 'Preferred time for daily digest delivery';
