-- Migration 036: Secure Messaging System
-- Encrypted client-advisor communication with notifications

-- =============================================================================
-- 1. MESSAGE THREADS
-- =============================================================================

CREATE TABLE message_threads (
    thread_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    advisor_id UUID REFERENCES users(user_id),
    
    -- Thread details
    subject TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE', -- 'ACTIVE', 'ARCHIVED', 'CLOSED'
    
    -- Metadata
    last_message_at TIMESTAMPTZ,
    message_count INTEGER DEFAULT 0,
    unread_count_client INTEGER DEFAULT 0,
    unread_count_advisor INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_threads_client (client_id, last_message_at DESC),
    INDEX idx_threads_advisor (advisor_id, last_message_at DESC),
    INDEX idx_threads_tenant (tenant_id)
);

ALTER TABLE message_threads ENABLE ROW LEVEL SECURITY;

CREATE POLICY message_threads_isolation ON message_threads
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 2. MESSAGES
-- =============================================================================

CREATE TABLE client_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES message_threads(thread_id) ON DELETE CASCADE,
    
    -- Sender info
    sender_type VARCHAR(20) NOT NULL, -- 'CLIENT', 'ADVISOR'
    sender_id UUID NOT NULL, -- client_id or user_id
   
    -- Message content (encrypted at rest in production)
    message_content TEXT NOT NULL,
    encrypted BOOLEAN DEFAULT FALSE,
    
    -- Attachments
    attachments JSONB DEFAULT '[]',
    /* Example:
    [
        {
            "file_id": "uuid",
            "filename": "tax_form.pdf",
            "file_size": 1024000,
            "mime_type": "application/pdf",
            "storage_path": "s3://..."
        }
    ]
    */
    
    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    read_by UUID,
    
    -- Reply chain
    reply_to_message_id UUID REFERENCES client_messages(message_id),
    
    -- Metadata
    sent_from_ip VARCHAR(45),
    sent_from_device VARCHAR(100),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_messages_thread_time (thread_id, created_at DESC),
    INDEX idx_messages_sender (sender_id, created_at DESC),
    INDEX idx_messages_unread (thread_id, is_read) WHERE is_read = FALSE
);

-- =============================================================================
-- 3. NOTIFICATIONS
-- =============================================================================

CREATE TYPE notification_priority AS ENUM ('LOW', 'NORMAL', 'HIGH', 'URGENT');

CREATE TABLE client_notifications (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    -- Notification details
    notification_type VARCHAR(50) NOT NULL,
    /* Types:
       'NEW_MESSAGE', 'NEW_DOCUMENT', 'UPCOMING_MEETING',
       'TRANSACTION_COMPLETE', 'GOAL_MILESTONE', 'MARKET_ALERT',
       'NBA_RECOMMENDATION', 'ACCOUNT_UPDATE'
    */
    
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    action_url TEXT,
    action_label VARCHAR(50), -- e.g., "View Message", "Download Document"
    
    -- Delivery channels
    delivered_via TEXT[] DEFAULT '{}', -- 'EMAIL', 'SMS', 'PUSH', 'IN_APP'
    
    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    
    -- Priority
    priority notification_priority DEFAULT 'NORMAL',
    
    -- Metadata
    related_entity_type VARCHAR(50), -- 'MESSAGE', 'DOCUMENT', 'MEETING', etc.
    related_entity_id UUID,
    
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_notifications_client_unread (client_id, is_read, created_at DESC),
    INDEX idx_notifications_priority (client_id, priority, created_at DESC) WHERE is_read = FALSE
);

ALTER TABLE client_notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY notifications_isolation ON client_notifications
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 4. NOTIFICATION PREFERENCES
-- =============================================================================

CREATE TABLE client_notification_settings (
    setting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id) UNIQUE,
    
    -- Channel preferences per notification type
    message_notifications JSONB DEFAULT '{"email": true, "sms": false, "push": true, "in_app": true}',
    document_notifications JSONB DEFAULT '{"email": true, "sms": false, "push": true, "in_app": true}',
    meeting_notifications JSONB DEFAULT '{"email": true, "sms": true, "push": true, "in_app": true}',
    transaction_notifications JSONB DEFAULT '{"email": true, "sms": false, "push": true, "in_app": true}',
    market_alert_notifications JSONB DEFAULT '{"email": false, "sms": false, "push": false, "in_app": true}',
    nba_notifications JSONB DEFAULT '{"email": true, "sms": false, "push": true, "in_app": true}',
    
    -- Quiet hours (no SMS/push during these times)
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME DEFAULT '22:00:00',
    quiet_hours_end TIME DEFAULT '08:00:00',
    
    -- Digest preferences
    daily_digest_enabled BOOLEAN DEFAULT FALSE,
    digest_delivery_time TIME DEFAULT '08:00:00',
    
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- 5. MESSAGE ATTACHMENTS
-- =============================================================================

CREATE TABLE message_attachments (
    attachment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES client_messages(message_id) ON DELETE CASCADE,
    
    filename TEXT NOT NULL,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    
    -- Storage
    storage_path TEXT NOT NULL,
    storage_bucket VARCHAR(255),
    
    -- Security
    checksum VARCHAR(64), -- SHA-256
    virus_scanned BOOLEAN DEFAULT FALSE,
    scan_result VARCHAR(20), -- 'CLEAN', 'INFECTED', 'UNKNOWN'
    
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_attachments_message (message_id)
);

-- =============================================================================
-- 6. HELPER FUNCTIONS
-- =============================================================================

-- Send a message
CREATE OR REPLACE FUNCTION send_message(
    p_thread_id UUID,
    p_sender_type VARCHAR,
    p_sender_id UUID,
    p_content TEXT,
    p_attachments JSONB DEFAULT '[]'
) RETURNS UUID AS $$
DECLARE
    v_message_id UUID;
    v_client_id UUID;
    v_advisor_id UUID;
BEGIN
    -- Insert message
    INSERT INTO client_messages (
        thread_id, sender_type, sender_id, message_content, attachments
    ) VALUES (
        p_thread_id, p_sender_type, p_sender_id, p_content, p_attachments
    )
    RETURNING message_id INTO v_message_id;
    
    -- Update thread metadata
    UPDATE message_threads
    SET last_message_at = NOW(),
        message_count = message_count + 1,
        unread_count_client = CASE WHEN p_sender_type = 'ADVISOR' THEN unread_count_client + 1 ELSE unread_count_client END,
        unread_count_advisor = CASE WHEN p_sender_type = 'CLIENT' THEN unread_count_advisor + 1 ELSE unread_count_advisor END
    WHERE thread_id = p_thread_id
    RETURNING client_id, advisor_id INTO v_client_id, v_advisor_id;
    
    -- Create notification for recipient
    IF p_sender_type = 'CLIENT' THEN
        -- Notify advisor
        INSERT INTO client_notifications (
            tenant_id, client_id, notification_type, title, message, action_url, priority
        )
        SELECT 
            tenant_id, 
            v_client_id,
            'NEW_MESSAGE',
            'New message from your advisor',
            LEFT(p_content, 100),
            '/messages/' || p_thread_id,
            'NORMAL'
        FROM message_threads WHERE thread_id = p_thread_id;
    ELSE
        -- Notify client
        INSERT INTO client_notifications (
            tenant_id, client_id, notification_type, title, message, action_url, priority
        )
        SELECT 
            tenant_id,
            v_client_id,
            'NEW_MESSAGE',
            'New message from your advisor',
            LEFT(p_content, 100),
            '/messages/' || p_thread_id,
            'NORMAL'
        FROM message_threads WHERE thread_id = p_thread_id;
    END IF;
    
    RETURN v_message_id;
END;
$$ LANGUAGE plpgsql;

-- Mark message as read
CREATE OR REPLACE FUNCTION mark_message_read(p_message_id UUID, p_reader_id UUID)
RETURNS VOID AS $$
DECLARE
    v_thread_id UUID;
    v_sender_type VARCHAR;
BEGIN
    -- Update message
    UPDATE client_messages
    SET is_read = TRUE,
        read_at = NOW(),
        read_by = p_reader_id
    WHERE message_id = p_message_id
    RETURNING thread_id, sender_type INTO v_thread_id, v_sender_type;
    
    -- Update thread unread count
    UPDATE message_threads
    SET unread_count_client = CASE WHEN v_sender_type = 'ADVISOR' 
                                   THEN GREATEST(0, unread_count_client - 1) 
                                   ELSE unread_count_client END,
        unread_count_advisor = CASE WHEN v_sender_type = 'CLIENT' 
                                    THEN GREATEST(0, unread_count_advisor - 1) 
                                    ELSE unread_count_advisor END
    WHERE thread_id = v_thread_id;
END;
$$ LANGUAGE plpgsql;

-- Get unread notification count
CREATE OR REPLACE FUNCTION get_unread_notification_count(p_client_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)::INTEGER
        FROM client_notifications
        WHERE client_id = p_client_id
        AND is_read = FALSE
        AND (expires_at IS NULL OR expires_at > NOW())
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- 7. TRIGGERS
-- =============================================================================

-- Auto-update thread timestamp
CREATE OR REPLACE FUNCTION update_thread_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER thread_update_trigger
BEFORE UPDATE ON message_threads
FOR EACH ROW
EXECUTE FUNCTION update_thread_timestamp();

-- =============================================================================
-- 8. INDEXES FOR PERFORMANCE
-- =============================================================================

-- Real-time message queries
CREATE INDEX idx_messages_realtime ON client_messages(thread_id, created_at DESC)
WHERE created_at > NOW() - INTERVAL '24 hours';

-- Unread messages
CREATE INDEX idx_unread_threads ON message_threads(client_id, unread_count_client)
WHERE unread_count_client > 0;

-- =============================================================================
-- 9. COMMENTS
-- =============================================================================

COMMENT ON TABLE message_threads IS 'Secure messaging threads between clients and advisors';
COMMENT ON TABLE client_messages IS 'Individual messages with encryption support and attachments';
COMMENT ON TABLE client_notifications IS 'Multi-channel notification system with priority levels';
COMMENT ON FUNCTION send_message IS 'Send a message and automatically create notification for recipient';
COMMENT ON FUNCTION mark_message_read IS 'Mark a message as read and update thread unread counts';
