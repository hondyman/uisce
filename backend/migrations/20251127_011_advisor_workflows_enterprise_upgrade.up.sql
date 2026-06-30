-- ============================================================================
-- ENTERPRISE UPGRADE: Best-in-Class Advisor Workflows
-- Migration: 20251127_011_advisor_workflows_enterprise_upgrade
-- ============================================================================
-- Adds AI/ML scoring, sentiment analysis, audit trails, versioning,
-- advanced rule engine, real-time collaboration, and analytics.
-- ============================================================================

-- ============================================================================
-- SECTION 1: AI-POWERED SCREENING & SCORING
-- ============================================================================

-- AI screening models and configurations
CREATE TABLE IF NOT EXISTS screening_rule_sets (
    rule_set_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_set_name TEXT NOT NULL,
    description TEXT,
    opportunity_type VARCHAR(50),
    
    -- Rule definitions (configurable by business)
    rules JSONB NOT NULL DEFAULT '[]'::JSONB,
    -- Example: [{"field": "target_irr_min", "operator": ">=", "value": 15, "weight": 0.2, "required": true}]
    
    -- Scoring weights
    scoring_weights JSONB DEFAULT '{}'::JSONB,
    passing_threshold DECIMAL(5,2) DEFAULT 70.0,
    
    -- Auto-actions on threshold
    auto_pass_threshold DECIMAL(5,2) DEFAULT 90.0,
    auto_fail_threshold DECIMAL(5,2) DEFAULT 30.0,
    auto_escalate_on_fail BOOLEAN DEFAULT TRUE,
    
    -- Versioning
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    effective_from TIMESTAMPTZ DEFAULT NOW(),
    effective_until TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_screening_rules_type ON screening_rule_sets(opportunity_type);
CREATE INDEX IF NOT EXISTS idx_screening_rules_active ON screening_rule_sets(is_active);

-- AI screening executions with detailed breakdowns
CREATE TABLE IF NOT EXISTS screening_executions (
    execution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID NOT NULL REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    rule_set_id UUID REFERENCES screening_rule_sets(rule_set_id),
    
    -- Execution details
    executed_at TIMESTAMPTZ DEFAULT NOW(),
    execution_type VARCHAR(50) DEFAULT 'AUTOMATED' CHECK (execution_type IN (
        'AUTOMATED',
        'MANUAL_OVERRIDE',
        'RESCREEN',
        'APPEAL'
    )),
    
    -- Scores
    total_score DECIMAL(5,2),
    rule_scores JSONB NOT NULL DEFAULT '[]'::JSONB,
    -- [{rule_name, score, max_score, passed, details, data_source}]
    
    -- AI enhancements
    ai_confidence DECIMAL(5,4),
    ai_risk_signals JSONB DEFAULT '[]'::JSONB,
    nlp_sentiment_score DECIMAL(5,2),  -- From pitch deck/PPM analysis
    similar_deals_comparison JSONB,    -- Comparison to historical deals
    
    -- Decision
    outcome VARCHAR(50) NOT NULL CHECK (outcome IN (
        'PASS',
        'FAIL',
        'CONDITIONAL_PASS',
        'ESCALATE',
        'MANUAL_REVIEW_REQUIRED'
    )),
    outcome_reasons TEXT[],
    
    -- Override tracking
    was_overridden BOOLEAN DEFAULT FALSE,
    override_by UUID,
    override_at TIMESTAMPTZ,
    override_reason TEXT,
    override_approved_by UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_screening_exec_opportunity ON screening_executions(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_screening_exec_outcome ON screening_executions(outcome);
CREATE INDEX IF NOT EXISTS idx_screening_exec_date ON screening_executions(executed_at);

-- ============================================================================
-- SECTION 2: DOCUMENT AI PROCESSING
-- ============================================================================

-- Document processing queue with AI extraction
CREATE TABLE IF NOT EXISTS document_processing_queue (
    queue_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    
    -- Document info
    document_type VARCHAR(50) NOT NULL CHECK (document_type IN (
        'PITCH_DECK',
        'PPM',
        'LPA',
        'SUBSCRIPTION_AGREEMENT',
        'SIDE_LETTER',
        'FINANCIAL_STATEMENTS',
        'TRACK_RECORD',
        'REFERENCE_LETTER',
        'DUE_DILIGENCE_QUESTIONNAIRE',
        'ESG_REPORT',
        'VALUATION_REPORT',
        'LEGAL_OPINION',
        'TAX_OPINION',
        'OTHER'
    )),
    document_url TEXT NOT NULL,
    document_name TEXT,
    document_hash TEXT,  -- For change detection
    
    -- Processing status
    status VARCHAR(50) DEFAULT 'QUEUED' CHECK (status IN (
        'QUEUED',
        'PROCESSING',
        'COMPLETED',
        'FAILED',
        'REVIEW_REQUIRED'
    )),
    
    -- AI extraction results
    extracted_data JSONB,
    extraction_confidence DECIMAL(5,4),
    
    -- Key terms extracted
    key_terms JSONB,  -- {management_fee, carried_interest, preferred_return, hurdle_rate, ...}
    risk_factors JSONB,  -- [{factor, severity, location_in_doc}]
    red_flags JSONB,  -- AI-detected concerns
    
    -- NLP analysis
    sentiment_analysis JSONB,  -- {overall: 0.7, by_section: {...}}
    readability_score DECIMAL(5,2),
    complexity_score DECIMAL(5,2),
    
    -- Comparison to standards
    terms_vs_market_avg JSONB,  -- {management_fee: {value: 2.0, market_avg: 1.5, percentile: 80}}
    
    -- Processing details
    processor_version TEXT,
    processing_started_at TIMESTAMPTZ,
    processing_completed_at TIMESTAMPTZ,
    processing_duration_ms INTEGER,
    error_message TEXT,
    
    -- Human review
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,
    corrections_made JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_queue_opportunity ON document_processing_queue(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_doc_queue_status ON document_processing_queue(status);
CREATE INDEX IF NOT EXISTS idx_doc_queue_type ON document_processing_queue(document_type);

-- ============================================================================
-- SECTION 3: COMPREHENSIVE AUDIT TRAIL
-- ============================================================================

-- Master audit log for all workflow actions
CREATE TABLE IF NOT EXISTS workflow_audit_log (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Entity tracking
    entity_type VARCHAR(50) NOT NULL,  -- 'opportunity', 'review', 'allocation', etc.
    entity_id UUID NOT NULL,
    
    -- Action details
    action VARCHAR(50) NOT NULL,  -- 'CREATE', 'UPDATE', 'DELETE', 'STAGE_CHANGE', 'APPROVAL', etc.
    action_category VARCHAR(50),  -- 'DATA_CHANGE', 'WORKFLOW', 'COMPLIANCE', 'ACCESS'
    
    -- Actor
    performed_by UUID,
    performed_by_name TEXT,
    performed_by_role TEXT,
    performed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Change details
    previous_state JSONB,
    new_state JSONB,
    changed_fields TEXT[],
    change_summary TEXT,
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    session_id TEXT,
    request_id TEXT,
    
    -- Compliance flags
    is_material_change BOOLEAN DEFAULT FALSE,
    requires_supervisor_review BOOLEAN DEFAULT FALSE,
    supervisor_reviewed_by UUID,
    supervisor_reviewed_at TIMESTAMPTZ,
    
    -- Related workflow
    workflow_instance_id TEXT,
    workflow_task_id TEXT,
    
    -- Retention
    retention_period_days INTEGER DEFAULT 2555,  -- 7 years default
    archive_after TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON workflow_audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON workflow_audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_performed_by ON workflow_audit_log(performed_by);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON workflow_audit_log(performed_at);
CREATE INDEX IF NOT EXISTS idx_audit_material ON workflow_audit_log(is_material_change) WHERE is_material_change = TRUE;

-- Create audit trigger function
CREATE OR REPLACE FUNCTION log_workflow_audit()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, new_state)
        VALUES (TG_TABLE_NAME, NEW.opportunity_id, 'CREATE', row_to_json(NEW));
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, previous_state, new_state, changed_fields)
        VALUES (
            TG_TABLE_NAME,
            NEW.opportunity_id,
            'UPDATE',
            row_to_json(OLD),
            row_to_json(NEW),
            ARRAY(SELECT key FROM jsonb_each(row_to_json(NEW)::jsonb) 
                  WHERE row_to_json(NEW)::jsonb->key != row_to_json(OLD)::jsonb->key)
        );
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO workflow_audit_log (entity_type, entity_id, action, previous_state)
        VALUES (TG_TABLE_NAME, OLD.opportunity_id, 'DELETE', row_to_json(OLD));
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Apply audit triggers
DROP TRIGGER IF EXISTS audit_investment_opportunities ON investment_opportunities;
CREATE TRIGGER audit_investment_opportunities
    AFTER INSERT OR UPDATE OR DELETE ON investment_opportunities
    FOR EACH ROW EXECUTE FUNCTION log_workflow_audit();

-- ============================================================================
-- SECTION 4: DOCUMENT VERSION CONTROL
-- ============================================================================

CREATE TABLE IF NOT EXISTS document_versions (
    version_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Parent document tracking
    document_id UUID NOT NULL,  -- Logical document ID (same across versions)
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    
    -- Version info
    version_number INTEGER NOT NULL,
    is_current BOOLEAN DEFAULT TRUE,
    
    -- Document details
    document_type VARCHAR(50) NOT NULL,
    document_name TEXT NOT NULL,
    file_url TEXT NOT NULL,
    file_size_bytes BIGINT,
    mime_type TEXT,
    checksum TEXT,
    
    -- Change tracking
    change_summary TEXT,
    change_type VARCHAR(50) CHECK (change_type IN (
        'INITIAL',
        'MINOR_EDIT',
        'MAJOR_REVISION',
        'LEGAL_UPDATE',
        'CORRECTION',
        'FINAL'
    )),
    
    -- Review status
    review_status VARCHAR(50) DEFAULT 'DRAFT' CHECK (review_status IN (
        'DRAFT',
        'PENDING_REVIEW',
        'APPROVED',
        'REJECTED',
        'SUPERSEDED'
    )),
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    review_comments TEXT,
    
    -- Signatures if applicable
    signature_required BOOLEAN DEFAULT FALSE,
    signed_version_url TEXT,
    
    -- Access tracking
    view_count INTEGER DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    last_viewed_by UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    
    UNIQUE(document_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_doc_versions_document ON document_versions(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_versions_opportunity ON document_versions(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_doc_versions_current ON document_versions(is_current) WHERE is_current = TRUE;

-- ============================================================================
-- SECTION 5: REAL-TIME COLLABORATION
-- ============================================================================

-- Collaboration workspaces
CREATE TABLE IF NOT EXISTS collaboration_workspaces (
    workspace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    
    workspace_name TEXT NOT NULL,
    workspace_type VARCHAR(50) DEFAULT 'DUE_DILIGENCE' CHECK (workspace_type IN (
        'DUE_DILIGENCE',
        'INVESTMENT_COMMITTEE',
        'LEGAL_REVIEW',
        'TAX_REVIEW',
        'CLIENT_COLLABORATION'
    )),
    
    -- Members
    members JSONB NOT NULL DEFAULT '[]'::JSONB,
    -- [{user_id, role, permissions: ['read', 'write', 'comment', 'approve'], added_at}]
    
    -- Settings
    settings JSONB DEFAULT '{}'::JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_collab_workspace_opportunity ON collaboration_workspaces(opportunity_id);

-- Real-time comments and discussions
CREATE TABLE IF NOT EXISTS collaboration_comments (
    comment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES collaboration_workspaces(workspace_id) ON DELETE CASCADE,
    
    -- Threading
    parent_comment_id UUID REFERENCES collaboration_comments(comment_id),
    thread_id UUID,  -- Groups related comments
    
    -- Content
    comment_text TEXT NOT NULL,
    comment_type VARCHAR(50) DEFAULT 'COMMENT' CHECK (comment_type IN (
        'COMMENT',
        'QUESTION',
        'ACTION_ITEM',
        'DECISION',
        'APPROVAL',
        'REJECTION',
        'CONCERN',
        'RESOLVED'
    )),
    
    -- Attachments
    attachments JSONB DEFAULT '[]'::JSONB,
    
    -- Mentions
    mentioned_users UUID[],
    
    -- Context (what this comment is about)
    context_type VARCHAR(50),  -- 'document', 'checklist_item', 'risk_flag', etc.
    context_id UUID,
    context_location TEXT,  -- e.g., "page 15, paragraph 3"
    
    -- Status for action items
    status VARCHAR(50) DEFAULT 'ACTIVE' CHECK (status IN (
        'ACTIVE',
        'RESOLVED',
        'ARCHIVED',
        'DELETED'
    )),
    resolved_by UUID,
    resolved_at TIMESTAMPTZ,
    
    -- Reactions
    reactions JSONB DEFAULT '{}'::JSONB,  -- {emoji: [user_id, ...]}
    
    -- Edit history
    is_edited BOOLEAN DEFAULT FALSE,
    edit_history JSONB DEFAULT '[]'::JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_collab_comments_workspace ON collaboration_comments(workspace_id);
CREATE INDEX IF NOT EXISTS idx_collab_comments_thread ON collaboration_comments(thread_id);
CREATE INDEX IF NOT EXISTS idx_collab_comments_context ON collaboration_comments(context_type, context_id);
CREATE INDEX IF NOT EXISTS idx_collab_comments_mentions ON collaboration_comments USING GIN(mentioned_users);

-- ============================================================================
-- SECTION 6: SCENARIO ANALYSIS & STRESS TESTING
-- ============================================================================

CREATE TABLE IF NOT EXISTS scenario_analyses (
    analysis_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id),
    
    -- Scenario definition
    scenario_name TEXT NOT NULL,
    scenario_type VARCHAR(50) NOT NULL CHECK (scenario_type IN (
        'BASE_CASE',
        'UPSIDE',
        'DOWNSIDE',
        'STRESS_TEST',
        'MONTE_CARLO',
        'HISTORICAL_REPLAY',
        'CUSTOM'
    )),
    
    -- Input assumptions
    assumptions JSONB NOT NULL,
    -- {market_return: -20%, interest_rate_change: +200bps, liquidity_shock: true, ...}
    
    -- Portfolio context
    portfolio_snapshot JSONB,  -- Current portfolio state
    proposed_changes JSONB,    -- Changes being analyzed
    
    -- Results
    results JSONB NOT NULL,
    -- {
    --   portfolio_return: -15.2%,
    --   max_drawdown: -25%,
    --   liquidity_shortfall: 500000,
    --   var_95: -12%,
    --   cvar_95: -18%,
    --   sharpe_ratio: 0.3,
    --   time_to_recovery_months: 24
    -- }
    
    -- Risk metrics
    risk_metrics JSONB,
    -- {concentration_risk, liquidity_risk, leverage_risk, correlation_risk}
    
    -- Visualization data
    charts_data JSONB,  -- Pre-computed chart data for frontend
    
    -- Monte Carlo specifics (if applicable)
    monte_carlo_iterations INTEGER,
    monte_carlo_percentiles JSONB,  -- {p5, p25, p50, p75, p95}
    
    -- Model info
    model_version TEXT,
    computation_time_ms INTEGER,
    
    -- Status
    status VARCHAR(50) DEFAULT 'COMPLETED' CHECK (status IN (
        'QUEUED',
        'RUNNING',
        'COMPLETED',
        'FAILED'
    )),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_scenario_client ON scenario_analyses(client_id);
CREATE INDEX IF NOT EXISTS idx_scenario_opportunity ON scenario_analyses(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_scenario_type ON scenario_analyses(scenario_type);

-- ============================================================================
-- SECTION 7: NOTIFICATION & ALERT SYSTEM
-- ============================================================================

-- Notification templates
CREATE TABLE IF NOT EXISTS notification_templates (
    template_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    template_code TEXT NOT NULL UNIQUE,
    template_name TEXT NOT NULL,
    
    -- Notification type
    notification_type VARCHAR(50) NOT NULL CHECK (notification_type IN (
        'OPPORTUNITY_STAGE_CHANGE',
        'TASK_ASSIGNED',
        'TASK_DUE_SOON',
        'TASK_OVERDUE',
        'CAPITAL_CALL_NOTICE',
        'CAPITAL_CALL_DUE',
        'DISTRIBUTION_RECEIVED',
        'REBALANCE_TRIGGER',
        'RISK_FLAG',
        'COMPLIANCE_DEADLINE',
        'DOCUMENT_READY',
        'SIGNATURE_REQUIRED',
        'MEETING_REMINDER',
        'QUARTERLY_REVIEW',
        'SYSTEM_ALERT'
    )),
    
    -- Channels
    channels TEXT[] NOT NULL DEFAULT ARRAY['IN_APP'],  -- ['IN_APP', 'EMAIL', 'SMS', 'PUSH']
    
    -- Templates per channel
    email_subject_template TEXT,
    email_body_template TEXT,
    sms_template TEXT,
    push_template TEXT,
    in_app_template TEXT,
    
    -- Priority
    default_priority VARCHAR(20) DEFAULT 'MEDIUM',
    
    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification queue
CREATE TABLE IF NOT EXISTS notification_queue (
    notification_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID,
    
    -- Recipient
    recipient_id UUID NOT NULL,
    recipient_email TEXT,
    recipient_phone TEXT,
    
    -- Content
    notification_type VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    data JSONB,  -- Template variables and action data
    
    -- Channels
    channels TEXT[] NOT NULL,
    
    -- Priority & scheduling
    priority VARCHAR(20) DEFAULT 'MEDIUM' CHECK (priority IN (
        'CRITICAL',
        'HIGH',
        'MEDIUM',
        'LOW'
    )),
    scheduled_for TIMESTAMPTZ DEFAULT NOW(),
    
    -- Delivery status per channel
    delivery_status JSONB DEFAULT '{}'::JSONB,
    -- {email: {status: 'sent', sent_at: '...', opened_at: '...'}, sms: {...}}
    
    -- In-app status
    read_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,
    actioned_at TIMESTAMPTZ,
    action_taken TEXT,
    
    -- Related entity
    entity_type VARCHAR(50),
    entity_id UUID,
    action_url TEXT,
    
    -- Retry tracking
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    last_error TEXT,
    
    -- Expiration
    expires_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_queue_recipient ON notification_queue(recipient_id);
CREATE INDEX IF NOT EXISTS idx_notification_queue_scheduled ON notification_queue(scheduled_for);
CREATE INDEX IF NOT EXISTS idx_notification_queue_unread ON notification_queue(recipient_id) 
    WHERE read_at IS NULL AND dismissed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notification_queue_type ON notification_queue(notification_type);

-- Add FK to templates only if the referenced column exists (some environments use a different notification_templates schema)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notification_templates' AND column_name = 'template_id')
     AND NOT EXISTS (
       SELECT 1 FROM information_schema.table_constraints tc
       JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
       WHERE tc.table_name = 'notification_queue' AND kcu.column_name = 'template_id' AND tc.constraint_type = 'FOREIGN KEY') THEN
    ALTER TABLE notification_queue ADD CONSTRAINT notification_queue_template_id_fkey FOREIGN KEY (template_id) REFERENCES notification_templates(template_id);
  END IF;
END$$;

-- User notification preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    preference_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    
    -- Global settings
    email_enabled BOOLEAN DEFAULT TRUE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    push_enabled BOOLEAN DEFAULT TRUE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    
    -- Quiet hours
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    quiet_hours_timezone TEXT DEFAULT 'America/New_York',
    
    -- Per-type preferences (override defaults)
    type_preferences JSONB DEFAULT '{}'::JSONB,
    -- {CAPITAL_CALL_NOTICE: {channels: ['EMAIL', 'SMS'], priority_threshold: 'LOW'}}
    
    -- Digest settings
    daily_digest_enabled BOOLEAN DEFAULT FALSE,
    daily_digest_time TIME DEFAULT '08:00',
    weekly_digest_enabled BOOLEAN DEFAULT TRUE,
    weekly_digest_day INTEGER DEFAULT 1,  -- Monday
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- SECTION 8: CALENDAR INTEGRATION
-- ============================================================================

CREATE TABLE IF NOT EXISTS calendar_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Related entities
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id),
    client_id UUID,
    review_id UUID,
    
    -- Event details
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN (
        'DUE_DILIGENCE_CALL',
        'MANAGER_MEETING',
        'INVESTMENT_COMMITTEE',
        'CLIENT_REVIEW',
        'QUARTERLY_REVIEW',
        'COMPLIANCE_DEADLINE',
        'CAPITAL_CALL_DUE',
        'DOCUMENT_DEADLINE',
        'TRAINING',
        'OTHER'
    )),
    
    title TEXT NOT NULL,
    description TEXT,
    location TEXT,
    video_conference_url TEXT,
    
    -- Timing
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,
    timezone TEXT DEFAULT 'America/New_York',
    
    -- Recurrence
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_rule TEXT,  -- RRULE format
    recurrence_parent_id UUID REFERENCES calendar_events(event_id),
    
    -- Attendees
    organizer_id UUID NOT NULL,
    attendees JSONB NOT NULL DEFAULT '[]'::JSONB,
    -- [{user_id, email, name, response_status: 'accepted'|'declined'|'tentative'|'pending', required: true}]
    
    -- External calendar sync
    external_calendar_id TEXT,  -- Google/Outlook event ID
    external_calendar_provider TEXT,
    sync_enabled BOOLEAN DEFAULT TRUE,
    last_synced_at TIMESTAMPTZ,
    
    -- Reminders
    reminders JSONB DEFAULT '[{"minutes": 30, "method": "EMAIL"}, {"minutes": 10, "method": "PUSH"}]'::JSONB,
    
    -- Meeting materials
    materials_url TEXT,
    agenda JSONB,
    
    -- Post-meeting
    meeting_notes TEXT,
    action_items JSONB,
    recording_url TEXT,
    
    -- Status
    status VARCHAR(50) DEFAULT 'CONFIRMED' CHECK (status IN (
        'TENTATIVE',
        'CONFIRMED',
        'CANCELLED',
        'COMPLETED'
    )),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_opportunity ON calendar_events(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_client ON calendar_events(client_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_time ON calendar_events(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_organizer ON calendar_events(organizer_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_type ON calendar_events(event_type);

-- ============================================================================
-- SECTION 9: ANALYTICS & METRICS
-- ============================================================================

-- KPI tracking for advisor dashboards
CREATE TABLE IF NOT EXISTS advisor_kpi_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    advisor_id UUID NOT NULL,
    snapshot_date DATE NOT NULL,
    
    -- Pipeline metrics
    opportunities_in_pipeline INTEGER DEFAULT 0,
    opportunities_by_stage JSONB,
    pipeline_value DECIMAL(15,2),
    avg_days_in_stage JSONB,
    
    -- Conversion metrics
    conversion_rate_to_committee DECIMAL(5,2),
    conversion_rate_to_funded DECIMAL(5,2),
    win_rate_30d DECIMAL(5,2),
    win_rate_90d DECIMAL(5,2),
    
    -- Activity metrics
    tasks_completed_7d INTEGER,
    tasks_overdue INTEGER,
    avg_response_time_hours DECIMAL(10,2),
    due_diligence_completion_rate DECIMAL(5,2),
    
    -- Client metrics
    clients_with_alt_exposure INTEGER,
    avg_alt_allocation_pct DECIMAL(5,2),
    total_alt_aum DECIMAL(15,2),
    unfunded_commitments DECIMAL(15,2),
    
    -- Performance metrics
    ytd_alternatives_return DECIMAL(5,2),
    ytd_alpha_vs_benchmark DECIMAL(5,2),
    
    -- Compliance metrics
    compliance_items_pending INTEGER,
    filings_due_30d INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(advisor_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_kpi_snapshots_advisor ON advisor_kpi_snapshots(advisor_id);
CREATE INDEX IF NOT EXISTS idx_kpi_snapshots_date ON advisor_kpi_snapshots(snapshot_date);

-- Opportunity funnel analytics
CREATE TABLE IF NOT EXISTS opportunity_funnel_metrics (
    metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Time period
    period_type VARCHAR(20) NOT NULL,  -- 'DAILY', 'WEEKLY', 'MONTHLY', 'QUARTERLY'
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    
    -- Segmentation
    opportunity_type VARCHAR(50),
    advisor_id UUID,
    
    -- Funnel stages
    intake_count INTEGER DEFAULT 0,
    screening_count INTEGER DEFAULT 0,
    due_diligence_count INTEGER DEFAULT 0,
    committee_count INTEGER DEFAULT 0,
    approved_count INTEGER DEFAULT 0,
    funded_count INTEGER DEFAULT 0,
    closed_won_count INTEGER DEFAULT 0,
    closed_lost_count INTEGER DEFAULT 0,
    
    -- Values
    intake_value DECIMAL(15,2) DEFAULT 0,
    funded_value DECIMAL(15,2) DEFAULT 0,
    
    -- Timing
    avg_days_intake_to_funded DECIMAL(10,2),
    avg_days_in_due_diligence DECIMAL(10,2),
    
    -- Drop-off analysis
    screening_to_dd_rate DECIMAL(5,2),
    dd_to_committee_rate DECIMAL(5,2),
    committee_to_funded_rate DECIMAL(5,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(period_type, period_start, opportunity_type, advisor_id)
);

CREATE INDEX IF NOT EXISTS idx_funnel_metrics_period ON opportunity_funnel_metrics(period_type, period_start);

-- ============================================================================
-- SECTION 10: EXTERNAL INTEGRATIONS
-- ============================================================================

-- Integration configurations
CREATE TABLE IF NOT EXISTS integration_configs (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    integration_type VARCHAR(50) NOT NULL CHECK (integration_type IN (
        'FUND_DATABASE',           -- PitchBook, Preqin, etc.
        'MARKET_DATA',             -- Bloomberg, Refinitiv
        'CUSTODIAN',               -- Schwab, Fidelity, Pershing
        'OMS',                     -- Order Management System
        'CRM',                     -- Salesforce, HubSpot
        'DOCUMENT_STORAGE',        -- S3, Azure Blob
        'ESIGNATURE',              -- DocuSign, Adobe Sign
        'CALENDAR',                -- Google, Outlook
        'EMAIL',                   -- SendGrid, SES
        'COMPLIANCE',              -- ComplySci, etc.
        'PORTFOLIO_ACCOUNTING',    -- Advent, SS&C
        'TAX',                     -- Tax prep systems
        'WEBHOOK_OUTBOUND'
    )),
    
    provider_name TEXT NOT NULL,
    
    -- Connection details (encrypted in practice)
    api_base_url TEXT,
    credentials JSONB,  -- Encrypted at rest
    
    -- Settings
    settings JSONB DEFAULT '{}'::JSONB,
    rate_limit_per_minute INTEGER,
    
    -- Status
    is_enabled BOOLEAN DEFAULT TRUE,
    last_health_check_at TIMESTAMPTZ,
    health_status VARCHAR(50),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Webhook subscriptions for external systems
CREATE TABLE IF NOT EXISTS webhook_subscriptions (
    subscription_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Target
    webhook_url TEXT NOT NULL,
    secret_key TEXT,  -- For HMAC signing
    
    -- Events to subscribe to
    event_types TEXT[] NOT NULL,
    -- ['opportunity.created', 'opportunity.stage_changed', 'capital_call.created', ...]
    
    -- Filters
    filters JSONB DEFAULT '{}'::JSONB,
    -- {opportunity_type: ['PRIVATE_EQUITY'], advisor_id: '...'}
    
    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    retry_policy JSONB DEFAULT '{"max_retries": 3, "backoff_seconds": [5, 30, 300]}'::JSONB,
    
    -- Stats
    total_deliveries INTEGER DEFAULT 0,
    successful_deliveries INTEGER DEFAULT 0,
    last_delivery_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ,
    last_failure_reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_webhook_subs_active ON webhook_subscriptions(is_active);
CREATE INDEX IF NOT EXISTS idx_webhook_subs_events ON webhook_subscriptions USING GIN(event_types);

-- Webhook delivery log
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    delivery_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID REFERENCES webhook_subscriptions(subscription_id) ON DELETE CASCADE,
    
    -- Event
    event_type TEXT NOT NULL,
    event_id UUID,
    payload JSONB NOT NULL,
    
    -- Delivery attempt
    attempt_number INTEGER DEFAULT 1,
    
    -- Response
    response_status INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    
    -- Status
    status VARCHAR(50) NOT NULL CHECK (status IN (
        'PENDING',
        'SUCCESS',
        'FAILED',
        'RETRYING'
    )),
    
    error_message TEXT,
    next_retry_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_subscription ON webhook_deliveries(subscription_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_retry ON webhook_deliveries(next_retry_at) WHERE status = 'RETRYING';

-- ============================================================================
-- SECTION 11: DATA QUALITY & VALIDATION
-- ============================================================================

-- Data quality rules
CREATE TABLE IF NOT EXISTS data_quality_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    rule_name TEXT NOT NULL,
    rule_code TEXT NOT NULL UNIQUE,
    description TEXT,
    
    -- Target
    target_table TEXT NOT NULL,
    target_column TEXT,
    
    -- Rule definition
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN (
        'NOT_NULL',
        'UNIQUE',
        'RANGE',
        'PATTERN',
        'REFERENCE',
        'CUSTOM_SQL',
        'BUSINESS_LOGIC'
    )),
    rule_expression TEXT,  -- SQL expression or regex
    rule_parameters JSONB,
    
    -- Severity
    severity VARCHAR(20) DEFAULT 'WARNING' CHECK (severity IN (
        'INFO',
        'WARNING',
        'ERROR',
        'CRITICAL'
    )),
    
    -- Actions
    block_on_failure BOOLEAN DEFAULT FALSE,
    auto_fix_enabled BOOLEAN DEFAULT FALSE,
    auto_fix_expression TEXT,
    
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Data quality issues log
CREATE TABLE IF NOT EXISTS data_quality_issues (
    issue_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES data_quality_rules(rule_id),
    
    -- Affected record
    table_name TEXT NOT NULL,
    record_id UUID NOT NULL,
    column_name TEXT,
    
    -- Issue details
    issue_type TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT,
    current_value TEXT,
    expected_value TEXT,
    
    -- Status
    status VARCHAR(50) DEFAULT 'OPEN' CHECK (status IN (
        'OPEN',
        'ACKNOWLEDGED',
        'FIXED',
        'IGNORED',
        'AUTO_FIXED'
    )),
    
    -- Resolution
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,
    
    detected_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dq_issues_table_record ON data_quality_issues(table_name, record_id);
CREATE INDEX IF NOT EXISTS idx_dq_issues_status ON data_quality_issues(status);
CREATE INDEX IF NOT EXISTS idx_dq_issues_severity ON data_quality_issues(severity);

-- ============================================================================
-- SECTION 12: UPGRADE EXISTING TABLES
-- ============================================================================

-- Add AI scoring fields to investment_opportunities
ALTER TABLE investment_opportunities 
ADD COLUMN IF NOT EXISTS ai_screening_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS ai_fit_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS ai_risk_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS ai_timing_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS ai_confidence DECIMAL(5,4),
ADD COLUMN IF NOT EXISTS ai_recommendations JSONB,
ADD COLUMN IF NOT EXISTS nlp_sentiment_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS similar_deals JSONB,
ADD COLUMN IF NOT EXISTS market_timing_signals JSONB,
ADD COLUMN IF NOT EXISTS last_ai_analysis_at TIMESTAMPTZ;

-- Add collaboration fields
ALTER TABLE investment_opportunities
ADD COLUMN IF NOT EXISTS collaboration_workspace_id UUID,
ADD COLUMN IF NOT EXISTS watchers UUID[] DEFAULT ARRAY[]::UUID[];

-- Add document version tracking
ALTER TABLE investment_opportunities
ADD COLUMN IF NOT EXISTS pitch_deck_version_id UUID,
ADD COLUMN IF NOT EXISTS ppm_version_id UUID,
ADD COLUMN IF NOT EXISTS subscription_doc_version_id UUID;

-- Add compliance tracking
ALTER TABLE investment_opportunities
ADD COLUMN IF NOT EXISTS compliance_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS compliance_flags JSONB DEFAULT '[]'::JSONB,
ADD COLUMN IF NOT EXISTS last_compliance_check_at TIMESTAMPTZ;

-- Enhance allocation_recommendations with ML details
ALTER TABLE allocation_recommendations
ADD COLUMN IF NOT EXISTS scenario_analysis_id UUID,
ADD COLUMN IF NOT EXISTS monte_carlo_results JSONB,
ADD COLUMN IF NOT EXISTS stress_test_results JSONB,
ADD COLUMN IF NOT EXISTS correlation_impact JSONB,
ADD COLUMN IF NOT EXISTS liquidity_forecast JSONB,
ADD COLUMN IF NOT EXISTS tax_efficiency_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS rebalancing_trades JSONB;

-- Enhance quarterly_reviews with analytics
ALTER TABLE quarterly_reviews
ADD COLUMN IF NOT EXISTS performance_attribution JSONB,
ADD COLUMN IF NOT EXISTS peer_comparison JSONB,
ADD COLUMN IF NOT EXISTS trend_analysis JSONB,
ADD COLUMN IF NOT EXISTS forecast_next_quarter JSONB,
ADD COLUMN IF NOT EXISTS client_satisfaction_score DECIMAL(5,2),
ADD COLUMN IF NOT EXISTS nps_score INTEGER;

-- ============================================================================
-- SECTION 13: MATERIALIZED VIEWS FOR DASHBOARDS
-- ============================================================================

-- Real-time pipeline summary
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_pipeline_summary AS
SELECT 
    advisor_id,
    opportunity_type,
    current_stage,
    COUNT(*) as count,
    SUM(target_commitment) as total_value,
    AVG(screening_score) as avg_screening_score,
    AVG(ai_fit_score) as avg_fit_score,
    MIN(created_at) as oldest_created,
    AVG(EXTRACT(EPOCH FROM (NOW() - stage_updated_at))/86400)::INTEGER as avg_days_in_stage
FROM investment_opportunities
WHERE current_stage NOT IN ('CLOSED_WON', 'CLOSED_LOST')
GROUP BY advisor_id, opportunity_type, current_stage;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_pipeline_summary ON mv_pipeline_summary (advisor_id, opportunity_type, current_stage);

-- Client alternative exposure summary
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_client_alt_exposure AS
SELECT 
    cat.client_id,
    cat.max_alternatives_pct,
    cat.min_liquid_assets_pct,
    COALESCE(SUM(CASE WHEN io.current_stage = 'FUNDED' THEN io.target_commitment ELSE 0 END), 0) as committed_value,
    COALESCE(SUM(CASE WHEN io.current_stage IN ('INTAKE', 'INITIAL_SCREEN', 'DUE_DILIGENCE', 'INVESTMENT_COMMITTEE', 'APPROVED', 'DOCUMENTATION') 
        THEN io.target_commitment ELSE 0 END), 0) as pipeline_value,
    COUNT(DISTINCT io.opportunity_id) as total_opportunities,
    COUNT(DISTINCT CASE WHEN io.current_stage = 'FUNDED' THEN io.opportunity_id END) as funded_count
FROM client_allocation_targets cat
LEFT JOIN investment_opportunities io ON cat.client_id = io.client_id
WHERE cat.effective_date <= CURRENT_DATE AND (cat.end_date IS NULL OR cat.end_date > CURRENT_DATE)
GROUP BY cat.client_id, cat.max_alternatives_pct, cat.min_liquid_assets_pct;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_client_alt_exposure ON mv_client_alt_exposure (client_id);

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_advisor_mvs()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_pipeline_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_client_alt_exposure;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SECTION 14: INSERT DEFAULT DATA
-- ============================================================================

-- Insert default notification templates (compat-aware)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notification_templates' AND column_name = 'template_code') THEN
    INSERT INTO notification_templates (template_code, template_name, notification_type, channels, email_subject_template, email_body_template, in_app_template, default_priority)
    VALUES 
      ('OPP_STAGE_CHANGE', 'Opportunity Stage Changed', 'OPPORTUNITY_STAGE_CHANGE', ARRAY['IN_APP', 'EMAIL'], '{{opportunity_name}} moved to {{new_stage}}', 'The opportunity {{opportunity_name}} has moved from {{old_stage}} to {{new_stage}}. {{#if notes}}Notes: {{notes}}{{/if}}', '{{opportunity_name}} → {{new_stage}}', 'MEDIUM'),
      ('TASK_ASSIGNED', 'Task Assigned', 'TASK_ASSIGNED', ARRAY['IN_APP', 'EMAIL'], 'New Task: {{task_title}}', 'You have been assigned a new task: {{task_title}}. Due: {{due_date}}. {{#if description}}Description: {{description}}{{/if}}', 'New task: {{task_title}}', 'MEDIUM'),
      ('CAPITAL_CALL', 'Capital Call Notice', 'CAPITAL_CALL_NOTICE', ARRAY['IN_APP', 'EMAIL', 'SMS'], 'Capital Call Notice: {{fund_name}} - ${{amount}}', 'A capital call has been issued for {{fund_name}}. Amount: ${{amount}}. Due Date: {{due_date}}. Please ensure sufficient liquidity.', 'Capital Call: {{fund_name}} - ${{amount}} due {{due_date}}', 'HIGH'),
      ('RISK_FLAG', 'Risk Flag Alert', 'RISK_FLAG', ARRAY['IN_APP', 'EMAIL'], '⚠️ Risk Flag: {{flag_type}} for {{client_name}}', 'A {{severity}} risk flag has been raised for {{client_name}}: {{description}}. Please review and take appropriate action.', '⚠️ {{severity}} Risk: {{flag_type}}', 'HIGH'),
      ('COMPLIANCE_DEADLINE', 'Compliance Deadline Approaching', 'COMPLIANCE_DEADLINE', ARRAY['IN_APP', 'EMAIL'], 'Compliance Deadline: {{filing_type}} due {{due_date}}', 'The {{filing_type}} filing is due on {{due_date}}. Current status: {{status}}. Please ensure all required information is submitted.', '📋 {{filing_type}} due {{due_date}}', 'HIGH')
    ON CONFLICT (template_code) DO NOTHING;

  ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notification_templates' AND column_name = 'key') THEN
    -- Legacy notification_templates schema uses (id, key, name, category, subject_template, body_template)
    INSERT INTO notification_templates (tenant_id, key, name, category, channels, subject_template, body_template)
    VALUES 
      ('default-tenant', 'OPP_STAGE_CHANGE', 'Opportunity Stage Changed', 'OPPORTUNITY_STAGE_CHANGE', ARRAY['IN_APP', 'EMAIL'], '{{opportunity_name}} moved to {{new_stage}}', 'The opportunity {{opportunity_name}} has moved from {{old_stage}} to {{new_stage}}. {{#if notes}}Notes: {{notes}}{{/if}}'),
      ('default-tenant', 'TASK_ASSIGNED', 'Task Assigned', 'TASK_ASSIGNED', ARRAY['IN_APP', 'EMAIL'], 'New Task: {{task_title}}', 'You have been assigned a new task: {{task_title}}. Due: {{due_date}}. {{#if description}}Description: {{description}}{{/if}}'),
      ('default-tenant', 'CAPITAL_CALL', 'Capital Call Notice', 'CAPITAL_CALL_NOTICE', ARRAY['IN_APP', 'EMAIL', 'SMS'], 'Capital Call Notice: {{fund_name}} - ${{amount}}', 'A capital call has been issued for {{fund_name}}. Amount: ${{amount}}. Due Date: {{due_date}}. Please ensure sufficient liquidity.'),
      ('default-tenant', 'RISK_FLAG', 'Risk Flag Alert', 'RISK_FLAG', ARRAY['IN_APP', 'EMAIL'], '⚠️ Risk Flag: {{flag_type}} for {{client_name}}', 'A {{severity}} risk flag has been raised for {{client_name}}: {{description}}. Please review and take appropriate action.'),
      ('default-tenant', 'COMPLIANCE_DEADLINE', 'Compliance Deadline Approaching', 'COMPLIANCE_DEADLINE', ARRAY['IN_APP', 'EMAIL'], 'Compliance Deadline: {{filing_type}} due {{due_date}}', 'The {{filing_type}} filing is due on {{due_date}}. Current status: {{status}}. Please ensure all required information is submitted.')
    ON CONFLICT (tenant_id, key) DO NOTHING;
  END IF;
END$$;

-- Insert default screening rule set
INSERT INTO screening_rule_sets (rule_set_name, description, opportunity_type, rules, scoring_weights, passing_threshold)
VALUES (
    'Standard PE/VC Screening',
    'Default screening rules for Private Equity and Venture Capital opportunities',
    'PRIVATE_EQUITY',
    '[
        {"field": "fund_size", "operator": ">=", "value": 100000000, "weight": 0.1, "required": false, "label": "Fund Size >= $100M"},
        {"field": "track_record_years_min", "operator": ">=", "value": 5, "weight": 0.15, "required": true, "label": "Track Record >= 5 Years"},
        {"field": "target_irr_min", "operator": ">=", "value": 15, "weight": 0.2, "required": true, "label": "Target IRR >= 15%"},
        {"field": "management_fee_rate", "operator": "<=", "value": 0.02, "weight": 0.1, "required": false, "label": "Management Fee <= 2%"},
        {"field": "carried_interest_rate", "operator": "<=", "value": 0.25, "weight": 0.1, "required": false, "label": "Carried Interest <= 25%"},
        {"field": "max_leverage_ratio", "operator": "<=", "value": 3.0, "weight": 0.15, "required": true, "label": "Leverage <= 3x"},
        {"field": "esg_score", "operator": ">=", "value": 70, "weight": 0.1, "required": false, "label": "ESG Score >= 70"},
        {"field": "operational_due_diligence", "operator": "=", "value": "passed", "weight": 0.1, "required": true, "label": "ODD Passed"}
    ]'::JSONB,
    '{"fund_quality": 0.3, "terms": 0.25, "track_record": 0.25, "risk": 0.2}'::JSONB,
    70.0
)
ON CONFLICT DO NOTHING;

-- Insert default data quality rules
INSERT INTO data_quality_rules (rule_name, rule_code, target_table, rule_type, rule_expression, severity, block_on_failure)
VALUES 
    ('Opportunity Must Have Fund Name', 'OPP_FUND_NAME_REQUIRED', 'investment_opportunities', 'NOT_NULL', 'fund_name IS NOT NULL', 'ERROR', true),
    ('Target Commitment Positive', 'OPP_COMMITMENT_POSITIVE', 'investment_opportunities', 'RANGE', 'target_commitment > 0', 'WARNING', false),
    ('IRR Range Valid', 'OPP_IRR_RANGE', 'investment_opportunities', 'RANGE', 'target_irr_min <= target_irr_max', 'WARNING', false),
    ('Due Date in Future', 'CAP_EVENT_DUE_FUTURE', 'capital_events', 'CUSTOM_SQL', 'due_date >= CURRENT_DATE OR status IN (''FUNDED'', ''PAID'', ''CANCELLED'')', 'INFO', false)
ON CONFLICT (rule_code) DO NOTHING;

-- ============================================================================
-- END OF ENTERPRISE UPGRADE MIGRATION
-- ============================================================================
