-- Migration: Add Semantic Reporting Analytics & Collaboration Tables
-- Version: 2024_002
-- Description: Creates tables for report analytics, usage tracking, comments, sharing, and version history

-- ============================================================================
-- REPORT ANALYTICS & USAGE TRACKING
-- ============================================================================

-- Report usage events for analytics
CREATE TABLE IF NOT EXISTS report_usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_id UUID NOT NULL,
    user_id UUID,
    event_type VARCHAR(50) NOT NULL, -- view, render, export, download, schedule, share
    format VARCHAR(20),
    duration_ms BIGINT,
    row_count INTEGER,
    parameter_hash VARCHAR(64),
    parameters JSONB,
    
    -- Context
    source VARCHAR(50), -- web, mobile, api, schedule
    ip_address VARCHAR(45),
    user_agent TEXT,
    session_id VARCHAR(100),
    request_id VARCHAR(100),
    
    -- Outcome
    status VARCHAR(20) NOT NULL DEFAULT 'success', -- success, error, cancelled
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Performance metrics
    query_time_ms BIGINT,
    render_time_ms BIGINT,
    cache_hit BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Partition by month for large datasets
    CONSTRAINT fk_report_usage_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) PARTITION BY RANGE (created_at);

-- Create initial partitions (extend as needed)
CREATE TABLE IF NOT EXISTS report_usage_events_2024_01 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_02 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_03 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_04 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_05 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_06 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_07 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_08 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_09 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_10 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_11 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE IF NOT EXISTS report_usage_events_2024_12 PARTITION OF report_usage_events
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Indexes for usage events
CREATE INDEX IF NOT EXISTS idx_report_usage_tenant ON report_usage_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_usage_report ON report_usage_events(report_id);
CREATE INDEX IF NOT EXISTS idx_report_usage_user ON report_usage_events(user_id);
CREATE INDEX IF NOT EXISTS idx_report_usage_type ON report_usage_events(event_type);
CREATE INDEX IF NOT EXISTS idx_report_usage_created ON report_usage_events(created_at);

-- Aggregated report analytics (materialized for performance)
CREATE TABLE IF NOT EXISTS report_analytics_daily (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_id UUID NOT NULL,
    date DATE NOT NULL,
    
    -- Counts
    view_count INTEGER NOT NULL DEFAULT 0,
    render_count INTEGER NOT NULL DEFAULT 0,
    export_count INTEGER NOT NULL DEFAULT 0,
    unique_users INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    
    -- Performance
    avg_render_time_ms NUMERIC(10,2),
    max_render_time_ms BIGINT,
    min_render_time_ms BIGINT,
    p95_render_time_ms BIGINT,
    
    -- Data
    total_rows_returned BIGINT NOT NULL DEFAULT 0,
    cache_hit_rate NUMERIC(5,4),
    
    -- Formats used
    formats_used JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, report_id, date)
);

CREATE INDEX IF NOT EXISTS idx_analytics_daily_tenant ON report_analytics_daily(tenant_id);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_report ON report_analytics_daily(report_id);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_date ON report_analytics_daily(date);

-- Tenant usage quotas and limits
CREATE TABLE IF NOT EXISTS tenant_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id),
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    
    -- Storage
    max_storage_bytes BIGINT NOT NULL DEFAULT 104857600, -- 100MB
    used_storage_bytes BIGINT NOT NULL DEFAULT 0,
    
    -- Reports
    max_report_definitions INTEGER NOT NULL DEFAULT 5,
    used_report_definitions INTEGER NOT NULL DEFAULT 0,
    
    -- Schedules
    max_schedules INTEGER NOT NULL DEFAULT 2,
    active_schedules INTEGER NOT NULL DEFAULT 0,
    
    -- Concurrent renders
    max_concurrent_renders INTEGER NOT NULL DEFAULT 1,
    current_renders INTEGER NOT NULL DEFAULT 0,
    
    -- Data retention
    history_retention_days INTEGER NOT NULL DEFAULT 7,
    
    -- Features
    allowed_output_formats JSONB NOT NULL DEFAULT '["html", "pdf"]'::jsonb,
    ai_features_enabled BOOLEAN NOT NULL DEFAULT false,
    advanced_scheduling BOOLEAN NOT NULL DEFAULT false,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- AUDIT LOGGING
-- ============================================================================

CREATE TABLE IF NOT EXISTS report_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    action VARCHAR(100) NOT NULL,
    details JSONB,
    
    -- Context
    ip_address VARCHAR(45),
    user_agent TEXT,
    session_id VARCHAR(100),
    request_id VARCHAR(100),
    
    -- Outcome
    outcome VARCHAR(20) NOT NULL, -- success, failure, denied
    error_message TEXT,
    
    -- Compliance
    data_classification VARCHAR(50),
    retention_policy VARCHAR(50),
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create audit log partitions
CREATE TABLE IF NOT EXISTS report_audit_log_2024_01 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_02 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_03 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_04 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_05 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_06 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_07 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_08 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_09 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_10 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_11 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE IF NOT EXISTS report_audit_log_2024_12 PARTITION OF report_audit_log
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

CREATE INDEX IF NOT EXISTS idx_audit_tenant ON report_audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_user ON report_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON report_audit_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_event ON report_audit_log(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_created ON report_audit_log(created_at);

-- ============================================================================
-- COLLABORATION: COMMENTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS report_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_id UUID NOT NULL,
    parent_id UUID REFERENCES report_comments(id),
    user_id UUID NOT NULL,
    user_name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    
    -- Anchor information
    anchor_element_id VARCHAR(100),
    anchor_element_type VARCHAR(50),
    anchor_selection JSONB,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- open, resolved, archived
    resolved_by UUID,
    resolved_at TIMESTAMP WITH TIME ZONE,
    
    -- Mentions
    mentions UUID[],
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_comment_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comments_tenant ON report_comments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_comments_report ON report_comments(report_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON report_comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comments_user ON report_comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_status ON report_comments(status);

-- ============================================================================
-- COLLABORATION: SHARING
-- ============================================================================

CREATE TABLE IF NOT EXISTS report_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_id UUID NOT NULL,
    shared_by UUID NOT NULL,
    
    -- Share type
    share_type VARCHAR(20) NOT NULL, -- direct, link, public
    
    -- Recipient (for direct shares)
    recipient_id UUID,
    recipient_type VARCHAR(20), -- user, team, role
    
    -- Link sharing
    share_link VARCHAR(100),
    link_expiry TIMESTAMP WITH TIME ZONE,
    password_hash VARCHAR(255),
    
    -- Permissions
    permission VARCHAR(20) NOT NULL, -- view, comment, edit, admin
    
    -- Restrictions
    allow_export BOOLEAN NOT NULL DEFAULT true,
    allow_print BOOLEAN NOT NULL DEFAULT true,
    watermark BOOLEAN NOT NULL DEFAULT false,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT fk_share_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_shares_tenant ON report_shares(tenant_id);
CREATE INDEX IF NOT EXISTS idx_shares_report ON report_shares(report_id);
CREATE INDEX IF NOT EXISTS idx_shares_recipient ON report_shares(recipient_id);
CREATE INDEX IF NOT EXISTS idx_shares_link ON report_shares(share_link);
CREATE INDEX IF NOT EXISTS idx_shares_type ON report_shares(share_type);

-- ============================================================================
-- VERSION HISTORY
-- ============================================================================

CREATE TABLE IF NOT EXISTS report_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    content JSONB NOT NULL,
    
    -- Change tracking
    change_type VARCHAR(20) NOT NULL, -- create, update, restore
    change_summary TEXT,
    
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_version_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(tenant_id, report_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_versions_tenant ON report_versions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_versions_report ON report_versions(report_id);
CREATE INDEX IF NOT EXISTS idx_versions_created ON report_versions(created_at);

-- ============================================================================
-- TRANSLATIONS (I18N)
-- ============================================================================

CREATE TABLE IF NOT EXISTS report_translations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_id UUID NOT NULL,
    language_code VARCHAR(10) NOT NULL,
    
    -- Translated content
    translations JSONB NOT NULL, -- { "field.path": "translated value" }
    
    -- Metadata
    is_auto_translated BOOLEAN NOT NULL DEFAULT false,
    translator_id UUID,
    reviewed_by UUID,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_translation_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(tenant_id, report_id, language_code)
);

CREATE INDEX IF NOT EXISTS idx_translations_tenant ON report_translations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_translations_report ON report_translations(report_id);
CREATE INDEX IF NOT EXISTS idx_translations_language ON report_translations(language_code);

-- ============================================================================
-- DATA MASKING RULES
-- ============================================================================

CREATE TABLE IF NOT EXISTS data_masking_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    field_pattern VARCHAR(500) NOT NULL, -- Regex pattern
    mask_type VARCHAR(20) NOT NULL, -- redact, partial, hash, tokenize, encrypt, null, email, phone, credit_card, ssn
    custom_pattern VARCHAR(500),
    replacement_value VARCHAR(500),
    
    -- Conditions
    data_classifications VARCHAR(50)[],
    apply_to_roles VARCHAR(100)[],
    exclude_roles VARCHAR(100)[],
    
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_masking_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_masking_tenant ON data_masking_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_masking_active ON data_masking_rules(is_active);

-- ============================================================================
-- ROW-LEVEL SECURITY POLICIES
-- ============================================================================

CREATE TABLE IF NOT EXISTS row_level_security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    data_source VARCHAR(100) NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    
    -- Policy conditions
    conditions JSONB NOT NULL, -- Array of {field, operator, value}
    
    -- Applies to
    roles VARCHAR(100)[],
    users UUID[],
    
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_rls_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rls_tenant ON row_level_security_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rls_datasource ON row_level_security_policies(data_source, table_name);
CREATE INDEX IF NOT EXISTS idx_rls_enabled ON row_level_security_policies(is_enabled);

-- ============================================================================
-- FUNCTIONS FOR QUOTA MANAGEMENT
-- ============================================================================

-- Function to update report count quota
CREATE OR REPLACE FUNCTION update_report_count_quota()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tenant_quotas 
        SET used_report_definitions = used_report_definitions + 1,
            updated_at = NOW()
        WHERE tenant_id = NEW.tenant_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tenant_quotas 
        SET used_report_definitions = GREATEST(0, used_report_definitions - 1),
            updated_at = NOW()
        WHERE tenant_id = OLD.tenant_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to aggregate daily analytics
CREATE OR REPLACE FUNCTION aggregate_daily_analytics(p_date DATE)
RETURNS void AS $$
BEGIN
    INSERT INTO report_analytics_daily (
        tenant_id,
        report_id,
        date,
        view_count,
        render_count,
        export_count,
        unique_users,
        error_count,
        avg_render_time_ms,
        max_render_time_ms,
        min_render_time_ms,
        total_rows_returned,
        cache_hit_rate,
        formats_used
    )
    SELECT
        tenant_id,
        report_id,
        p_date,
        COUNT(*) FILTER (WHERE event_type = 'view'),
        COUNT(*) FILTER (WHERE event_type = 'render'),
        COUNT(*) FILTER (WHERE event_type = 'export'),
        COUNT(DISTINCT user_id),
        COUNT(*) FILTER (WHERE status = 'error'),
        AVG(render_time_ms)::NUMERIC(10,2),
        MAX(render_time_ms),
        MIN(render_time_ms) FILTER (WHERE render_time_ms > 0),
        SUM(row_count),
        AVG(CASE WHEN cache_hit THEN 1.0 ELSE 0.0 END)::NUMERIC(5,4),
        jsonb_object_agg(COALESCE(format, 'unknown'), COUNT(*)) FILTER (WHERE format IS NOT NULL)
    FROM report_usage_events
    WHERE DATE(created_at) = p_date
    GROUP BY tenant_id, report_id
    ON CONFLICT (tenant_id, report_id, date) 
    DO UPDATE SET
        view_count = EXCLUDED.view_count,
        render_count = EXCLUDED.render_count,
        export_count = EXCLUDED.export_count,
        unique_users = EXCLUDED.unique_users,
        error_count = EXCLUDED.error_count,
        avg_render_time_ms = EXCLUDED.avg_render_time_ms,
        max_render_time_ms = EXCLUDED.max_render_time_ms,
        min_render_time_ms = EXCLUDED.min_render_time_ms,
        total_rows_returned = EXCLUDED.total_rows_returned,
        cache_hit_rate = EXCLUDED.cache_hit_rate,
        formats_used = EXCLUDED.formats_used,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE report_usage_events IS 'Tracks all report usage events for analytics and telemetry';
COMMENT ON TABLE report_analytics_daily IS 'Aggregated daily analytics per report';
COMMENT ON TABLE tenant_quotas IS 'Resource quotas and limits per tenant';
COMMENT ON TABLE report_audit_log IS 'Audit trail for all report-related actions';
COMMENT ON TABLE report_comments IS 'Comments and annotations on reports';
COMMENT ON TABLE report_shares IS 'Report sharing configurations';
COMMENT ON TABLE report_versions IS 'Version history for reports';
COMMENT ON TABLE report_translations IS 'Translations for report content (i18n)';
COMMENT ON TABLE data_masking_rules IS 'PII and sensitive data masking rules';
COMMENT ON TABLE row_level_security_policies IS 'Row-level security policies for data access control';
