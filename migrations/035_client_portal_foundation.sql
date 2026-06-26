-- Migration 035: Client Portal Foundation
-- Personalized dashboard with customizable widgets and analytics

-- =============================================================================
-- 1. CLIENT PORTAL PREFERENCES
-- =============================================================================

CREATE TABLE client_portal_preferences (
    preference_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id) UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Dashboard layout (react-grid-layout format)
    dashboard_layout JSONB DEFAULT '{"widgets": []}',
    /* Example:
    {
        "widgets": [
            {"id": "portfolio", "x": 0, "y": 0, "w": 6, "h": 4, "minW": 4, "minH": 3},
            {"id": "goals", "x": 6, "y": 0, "w": 6, "h": 4, "minW": 4, "minH": 3},
            {"id": "transactions", "x": 0, "y": 4, "w": 12, "h": 3, "minW": 6, "minH": 2},
            {"id": "meetings", "x": 0, "y": 7, "w": 6, "h": 3, "minW": 4, "minH": 2},
            {"id": "documents", "x": 6, "y": 7, "w": 6, "h": 3, "minW": 4, "minH": 2}
        ]
    }
    */
    
    -- Widget visibility & order
    enabled_widgets TEXT[] DEFAULT ARRAY[
        'portfolio', 'goals', 'transactions', 'meetings', 
        'documents', 'account_summary', 'news', 'nba_recommendations'
    ],
    
    -- Appearance customization
    theme VARCHAR(20) DEFAULT 'LIGHT', -- 'LIGHT', 'DARK', 'AUTO'
    accent_color VARCHAR(7) DEFAULT '#1976d2', -- Hex color
    compact_mode BOOLEAN DEFAULT FALSE,
    
    -- Localization
    language VARCHAR(10) DEFAULT 'en-US',
    currency VARCHAR(3) DEFAULT 'USD',
    timezone VARCHAR(50) DEFAULT 'America/New_York',
    date_format VARCHAR(20) DEFAULT 'MM/DD/YYYY',
    
    -- Notification preferences
    email_notifications BOOLEAN DEFAULT TRUE,
    sms_notifications BOOLEAN DEFAULT FALSE,
    push_notifications BOOLEAN DEFAULT TRUE,
    notification_frequency VARCHAR(20) DEFAULT 'REALTIME', -- 'REALTIME', 'DAILY', 'WEEKLY'
    
    -- Notification types
    notify_on_transactions BOOLEAN DEFAULT TRUE,
    notify_on_meetings BOOLEAN DEFAULT TRUE,
    notify_on_documents BOOLEAN DEFAULT TRUE,
    notify_on_messages BOOLEAN DEFAULT TRUE,
    notify_on_market_alerts BOOLEAN DEFAULT FALSE,
    notify_on_nba_actions BOOLEAN DEFAULT TRUE,
    
    -- Performance
    data_refresh_interval INTEGER DEFAULT 300, -- Seconds (5 minutes)
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_portal_prefs_client (client_id),
    INDEX idx_portal_prefs_tenant (tenant_id)
);

-- RLS for multi-tenancy
ALTER TABLE client_portal_preferences ENABLE ROW LEVEL SECURITY;

CREATE POLICY portal_prefs_isolation ON client_portal_preferences
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 2. PORTAL ANALYTICS (Engagement Tracking)
-- =============================================================================

CREATE TABLE client_portal_analytics (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Event details
    event_type VARCHAR(50) NOT NULL,
    /* Event types:
       'LOGIN', 'LOGOUT', 'WIDGET_VIEW', 'WIDGET_INTERACT',
       'DOCUMENT_DOWNLOAD', 'MESSAGE_SENT', 'GOAL_UPDATED',
       'TRANSACTION_INITIATED', 'SETTINGS_CHANGED'
    */
    
    event_data JSONB DEFAULT '{}',
    /* Example for WIDGET_VIEW:
    {
        "widget_id": "portfolio",
        "duration_seconds": 45,
        "interactions": 3
    }
    */
    
    -- Session context
    session_id UUID,
    page_url TEXT,
    
    -- Device & browser info
    device_type VARCHAR(20), -- 'DESKTOP', 'MOBILE', 'TABLET'
    device_os VARCHAR(50),
    browser VARCHAR(50),
    browser_version VARCHAR(20),
    screen_resolution VARCHAR(20),
    
    -- Location
    ip_address VARCHAR(45),
    city TEXT,
    country VARCHAR(2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_analytics_client_time (client_id, created_at DESC),
    INDEX idx_analytics_event_type (event_type, created_at DESC),
    INDEX idx_analytics_session (session_id, created_at DESC)
);

-- Partitioning by month for performance (optional for large scale)
-- CREATE TABLE client_portal_analytics_2025_01 PARTITION OF client_portal_analytics
-- FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

ALTER TABLE client_portal_analytics ENABLE ROW LEVEL SECURITY;

CREATE POLICY portal_analytics_isolation ON client_portal_analytics
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 3. WIDGET FAVORITES (Quick Access)
-- =============================================================================

CREATE TABLE client_widget_favorites (
    favorite_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    widget_id VARCHAR(50) NOT NULL,
    display_order INTEGER DEFAULT 0,
    
    added_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (client_id, widget_id),
    INDEX idx_favorites_client (client_id, display_order)
);

-- =============================================================================
-- 4. HELPER FUNCTIONS
-- =============================================================================

-- Initialize default preferences for new client
CREATE OR REPLACE FUNCTION initialize_client_portal_preferences(p_client_id UUID, p_tenant_id UUID)
RETURNS UUID AS $$
DECLARE
    v_preference_id UUID;
BEGIN
    INSERT INTO client_portal_preferences (client_id, tenant_id, dashboard_layout)
    VALUES (
        p_client_id,
        p_tenant_id,
        '{
            "widgets": [
                {"id": "portfolio", "x": 0, "y": 0, "w": 6, "h": 4},
                {"id": "goals", "x": 6, "y": 0, "w": 6, "h": 4},
                {"id": "transactions", "x": 0, "y": 4, "w": 12, "h": 3},
                {"id": "meetings", "x": 0, "y": 7, "w": 6, "h": 3},
                {"id": "documents", "x": 6, "y": 7, "w": 6, "h": 3}
            ]
        }'::JSONB
    )
    RETURNING preference_id INTO v_preference_id;
    
    RETURN v_preference_id;
END;
$$ LANGUAGE plpgsql;

-- Track portal event
CREATE OR REPLACE FUNCTION track_portal_event(
    p_client_id UUID,
    p_tenant_id UUID,
    p_event_type VARCHAR,
    p_event_data JSONB DEFAULT '{}',
    p_session_id UUID DEFAULT NULL,
    p_device_type VARCHAR DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_event_id UUID;
BEGIN
    INSERT INTO client_portal_analytics (
        client_id, tenant_id, event_type, event_data, 
        session_id, device_type
    ) VALUES (
        p_client_id, p_tenant_id, p_event_type, p_event_data,
        p_session_id, p_device_type
    )
    RETURNING event_id INTO v_event_id;
    
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;

-- Get portal engagement metrics
CREATE OR REPLACE FUNCTION get_portal_engagement_metrics(
    p_client_id UUID,
    p_days INTEGER DEFAULT 30
) RETURNS TABLE (
    total_logins INTEGER,
    avg_session_duration_minutes DECIMAL,
    most_viewed_widget TEXT,
    widget_view_count INTEGER,
    last_login TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    WITH login_events AS (
        SELECT created_at
        FROM client_portal_analytics
        WHERE client_id = p_client_id
        AND event_type = 'LOGIN'
        AND created_at >= NOW() - (p_days || ' days')::INTERVAL
    ),
    widget_views AS (
        SELECT 
            event_data->>'widget_id' AS widget_id,
            COUNT(*) AS view_count
        FROM client_portal_analytics
        WHERE client_id = p_client_id
        AND event_type = 'WIDGET_VIEW'
        AND created_at >= NOW() - (p_days || ' days')::INTERVAL
        GROUP BY event_data->>'widget_id'
        ORDER BY view_count DESC
        LIMIT 1
    )
    SELECT
        (SELECT COUNT(*)::INTEGER FROM login_events),
        0::DECIMAL, -- Placeholder for avg session duration
        (SELECT widget_id FROM widget_views),
        (SELECT view_count::INTEGER FROM widget_views),
        (SELECT MAX(created_at) FROM login_events);
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 5. TRIGGERS
-- =============================================================================

-- Auto-update timestamp
CREATE OR REPLACE FUNCTION update_portal_prefs_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER portal_prefs_update_trigger
BEFORE UPDATE ON client_portal_preferences
FOR EACH ROW
EXECUTE FUNCTION update_portal_prefs_timestamp();

-- =============================================================================
-- 6. INDEXES FOR ANALYTICS QUERIES
-- =============================================================================

-- Engagement funnel analysis
CREATE INDEX idx_analytics_funnel ON client_portal_analytics(client_id, event_type, created_at)
WHERE event_type IN ('LOGIN', 'WIDGET_VIEW', 'WIDGET_INTERACT');

-- Time-series analysis (daily active users)
CREATE INDEX idx_analytics_daily_active ON client_portal_analytics(
    tenant_id, 
    DATE(created_at), 
    client_id
) WHERE event_type = 'LOGIN';

-- =============================================================================
-- 7. COMMENTS
-- =============================================================================

COMMENT ON TABLE client_portal_preferences IS 'Client portal customization preferences including dashboard layout, theme, and notification settings';
COMMENT ON TABLE client_portal_analytics IS 'Portal engagement tracking for analytics and product improvement';
COMMENT ON FUNCTION track_portal_event IS 'Helper function to track portal events with automatic timestamp';
COMMENT ON FUNCTION get_portal_engagement_metrics IS 'Retrieve engagement metrics for a client over specified time period';
