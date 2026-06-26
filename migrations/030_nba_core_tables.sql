-- Migration 030: NBA (Next Best Action) System - Core Tables
-- Description: AI-powered advisor productivity platform with signal detection,
--              ML recommendations, action catalog, and outcome tracking

-- =============================================================================
-- 1. SIGNAL DETECTION & STORAGE
-- =============================================================================

CREATE TYPE nba_signal_category AS ENUM (
    'PORTFOLIO',      -- Portfolio-based signals (losses, concentration, etc.)
    'BEHAVIORAL',     -- Client behavior signals (disengagement, portal usage)
    'LIFECYCLE',      -- Life event signals (age milestones, job changes)
    'MARKET',         -- Market event signals (volatility, sector rotation)
    'COMPLIANCE'      -- Regulatory signals (RMD age, filing deadlines)
);

CREATE TABLE nba_signals (
    signal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    client_id UUID NOT NULL,
    
    -- Signal classification
    signal_type VARCHAR(100) NOT NULL, -- Specific signal: 'UNREALIZED_LOSS_DETECTED', 'PORTAL_DISENGAGEMENT'
    signal_category nba_signal_category NOT NULL,
    signal_strength DECIMAL(3,2) CHECK (signal_strength BETWEEN 0 AND 1), -- 0.0 to 1.0
    
    -- Signal data
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    signal_data JSONB NOT NULL, -- Event-specific details
    metadata JSONB, -- Additional context (market conditions, thresholds, etc.)
    
    -- Processing status
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    recommendation_generated BOOLEAN DEFAULT FALSE,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes for performance
    INDEX idx_nba_signals_unprocessed (tenant_id, processed, detected_at DESC),
    INDEX idx_nba_signals_client (client_id, detected_at DESC),
    INDEX idx_nba_signals_type (signal_type, detected_at DESC)
);

-- RLS policy
ALTER TABLE nba_signals ENABLE ROW LEVEL SECURITY;

CREATE POLICY nba_signals_tenant_isolation ON nba_signals
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- =============================================================================
-- 2. ACTION CATALOG (Pre-configured advisor actions)
-- =============================================================================

CREATE TYPE nba_action_channel AS ENUM (
    'PHONE',
    'EMAIL',
    'VIDEO_CALL',
    'IN_PERSON',
    'PORTAL_MESSAGE',
    'AUTOMATED'
);

CREATE TYPE nba_action_category AS ENUM (
    'PROACTIVE_OUTREACH',
    'SERVICE_DELIVERY',
    'PORTFOLIO_MANAGEMENT',
    'RELATIONSHIP_BUILDING',
    'COMPLIANCE',
    'PLANNING'
);

CREATE TABLE nba_action_catalog (
    action_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_code VARCHAR(100) UNIQUE NOT NULL, -- e.g., 'PROACTIVE_TAX_LOSS_HARVEST'
    action_name TEXT NOT NULL,
    action_category nba_action_category NOT NULL,
    description TEXT,
    
    -- Default settings
    default_channel nba_action_channel NOT NULL,
    estimated_duration_minutes INTEGER,
    estimated_revenue_impact DECIMAL(10,2), -- Expected additional revenue
    client_value_impact DECIMAL(3,2), -- Expected satisfaction increase (0.0 to 1.0)
    
    -- Automation & templates
    automation_eligible BOOLEAN DEFAULT FALSE,
    template_content JSONB, -- {email_subject, email_body, call_script, meeting_agenda}
    
    -- Requirements
    required_advisor_skills TEXT[], -- ['TAX_PLANNING', 'ESTATE_PLANNING']
    min_client_aum DECIMAL(12,2), -- Minimum AUM required
    compliance_review_required BOOLEAN DEFAULT FALSE,
    
    -- Success tracking
    success_metrics JSONB, -- {success_metric: 'tax_loss_harvested_amount', target_value: 10000}
    
    -- Status
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed data with sample actions (inserted after table creation)

-- =============================================================================
-- 3. NBA RECOMMENDATIONS (ML-generated)
-- =============================================================================

CREATE TYPE nba_recommendation_status AS ENUM (
    'PENDING',
    'VIEWED',
    'EXECUTING',
    'COMPLETED',
    'DISMISSED',
    'EXPIRED'
);

CREATE TABLE nba_recommendations (
    recommendation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    client_id UUID NOT NULL,
    advisor_id UUID NOT NULL,
    action_id UUID NOT NULL REFERENCES nba_action_catalog(action_id),
    trigger_signal_id UUID REFERENCES nba_signals(signal_id),
    
    -- ML model predictions
    model_version VARCHAR(50), -- e.g., 'nba-v1.2.0'
    confidence_score DECIMAL(3,2) CHECK (confidence_score BETWEEN 0 AND 1),
    urgency_score DECIMAL(3,2) CHECK (urgency_score BETWEEN 0 AND 1),
    expected_value DECIMAL(10,2), -- Expected revenue
    success_probability DECIMAL(3,2) CHECK (success_probability BETWEEN 0 AND 1),
    
    -- Ranking & prioritization
    overall_score DECIMAL(10,2), -- urgency × value × success_prob
    rank_for_advisor INTEGER, -- Rank within advisor's queue
    
    -- AI explanation
    reasoning TEXT NOT NULL, -- Human-readable explanation
    supporting_data JSONB, -- Client context, signal details, market conditions
    
    -- Execution tracking
    status nba_recommendation_status DEFAULT 'PENDING',
    recommended_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    viewed_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    dismissed_at TIMESTAMPTZ,
    dismissal_reason VARCHAR(100),
    dismissal_notes TEXT,
    
    -- Expiration
    expires_at TIMESTAMPTZ, -- Time-sensitive recommendations expire
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_nba_rec_pending (tenant_id, advisor_id, status, overall_score DESC, recommended_at DESC),
    INDEX idx_nba_rec_client (client_id, recommended_at DESC),
    INDEX idx_nba_rec_signal (trigger_signal_id)
);

-- RLS policy
ALTER TABLE nba_recommendations ENABLE ROW LEVEL SECURITY;

CREATE POLICY nba_recommendations_tenant_isolation ON nba_recommendations
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- =============================================================================
-- 4. ACTION OUTCOMES (Training data for ML model)
-- =============================================================================

CREATE TABLE nba_action_outcomes (
    outcome_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id UUID NOT NULL REFERENCES nba_recommendations(recommendation_id),
    client_id UUID NOT NULL,
    advisor_id UUID NOT NULL,
    action_id UUID NOT NULL REFERENCES nba_action_catalog(action_id),
    trigger_signal_type VARCHAR(100),
    
    -- Execution details
    execution_channel nba_action_channel,
    execution_time_minutes INTEGER,
    execution_notes TEXT,
    
    -- Outcome metrics (for model training)
    client_responded BOOLEAN,
    response_time_hours INTEGER,
    action_successful BOOLEAN,
    revenue_generated DECIMAL(10,2),
    client_satisfaction_change DECIMAL(3,2), -- Change in satisfaction score
    aum_change DECIMAL(12,2), -- Change in AUM
    
    -- Advisor feedback
    advisor_feedback TEXT,
    advisor_rating INTEGER CHECK (advisor_rating BETWEEN 1 AND 5),
    would_recommend_again BOOLEAN,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes for analytics
    INDEX idx_nba_outcomes_action (action_id, action_successful),
    INDEX idx_nba_outcomes_advisor (advisor_id, created_at DESC),
    INDEX idx_nba_outcomes_training (action_id, action_successful, created_at DESC)
);

-- RLS policy
ALTER TABLE nba_action_outcomes ENABLE ROW LEVEL SECURITY;

CREATE POLICY nba_action_outcomes_tenant_isolation ON nba_action_outcomes
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM nba_recommendations
            WHERE nba_recommendations.recommendation_id = nba_action_outcomes.recommendation_id
            AND nba_recommendations.tenant_id = current_setting('app.current_tenant_id')::UUID
        )
    );

-- =============================================================================
-- 5. HELPER FUNCTIONS
-- =============================================================================

-- Function to calculate overall recommendation score
CREATE OR REPLACE FUNCTION calculate_nba_overall_score(
    urgency DECIMAL,
    expected_val DECIMAL,
    success_prob DECIMAL
) RETURNS DECIMAL AS $$
BEGIN
    RETURN (urgency * 0.4 + (expected_val / 10000) * 0.3 + success_prob * 0.3);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to expire old recommendations
CREATE OR REPLACE FUNCTION expire_old_nba_recommendations() RETURNS VOID AS $$
BEGIN
    UPDATE nba_recommendations
    SET status = 'EXPIRED',
        updated_at = NOW()
    WHERE status = 'PENDING'
    AND (
        expires_at < NOW()
        OR recommended_at < NOW() - INTERVAL '7 days'
    );
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 6. TRIGGERS
-- =============================================================================

-- Auto-update updated_at timestamps
CREATE OR REPLACE FUNCTION update_nba_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER nba_recommendations_updated_at
    BEFORE UPDATE ON nba_recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_nba_updated_at();

CREATE TRIGGER nba_action_catalog_updated_at
    BEFORE UPDATE ON nba_action_catalog
    FOR EACH ROW
    EXECUTE FUNCTION update_nba_updated_at();

-- =============================================================================
-- 7. SEED DATA - Sample Action Catalog Entries
-- =============================================================================

INSERT INTO nba_action_catalog (
    action_code,
    action_name,
    action_category,
    description,
    default_channel,
    estimated_duration_minutes,
    estimated_revenue_impact,
    client_value_impact,
    automation_eligible,
    template_content,
    required_advisor_skills,
    success_metrics
) VALUES
-- Tax-Loss Harvesting
(
    'PROACTIVE_TAX_LOSS_HARVEST',
    'Initiate Tax-Loss Harvesting Review',
    'PORTFOLIO_MANAGEMENT',
    'Proactively reach out to discuss tax-loss harvesting opportunities based on unrealized losses detected in portfolio',
    'PHONE',
    30,
    2500.00,
    0.15,
    FALSE,
    '{
        "email_subject": "Opportunity to Reduce Your 2025 Tax Bill",
        "email_body": "Hi {client_first_name},\n\nI noticed some unrealized losses in your portfolio that could save you approximately ${estimated_tax_savings:,.0f} in taxes this year through strategic tax-loss harvesting.\n\nWould you have 20 minutes this week to discuss this opportunity?\n\nBest regards,\n{advisor_name}",
        "call_script": "Opening: I wanted to reach out because our system flagged a potential tax savings opportunity in your account.\n\nKey Points:\n- Current unrealized losses: ${total_loss}\n- Estimated tax savings: ${tax_benefit}\n- Recommended action: Harvest losses and reinvest in similar securities\n\nClose: Can we schedule 20 minutes to walk through the specific positions?"
    }'::jsonb,
    ARRAY['TAX_PLANNING'],
    '{"success_metric": "tax_loss_harvested_amount", "target_value": 10000}'::jsonb
),
-- Client Re-engagement
(
    'REENGAGEMENT_OUTREACH',
    'Client Re-engagement Call',
    'RELATIONSHIP_BUILDING',
    'Reach out to client showing signs of disengagement (low portal logins, low email opens)',
    'PHONE',
    20,
    5000.00,
    0.25,
    FALSE,
    '{
        "call_script": "Hi {client_first_name}, I realized we haven''t connected in a while and wanted to check in. How have things been going for you?\n\n[Listen actively]\n\nI want to make sure we''re providing the level of service and communication that works best for you. Is there anything we could be doing differently?\n\n[Adjust communication preferences if needed]\n\nLet''s schedule a portfolio review in the next couple weeks. What works better for you - morning or afternoon?",
        "follow_up_email": "Great talking with you today! As discussed, I''m scheduling our portfolio review for {meeting_date}. Looking forward to it."
    }'::jsonb,
    ARRAY['RELATIONSHIP_MANAGEMENT'],
    '{"success_metric": "engagement_score_increase", "target_value": 0.3}'::jsonb
),
-- RMD Planning
(
    'RMD_PLANNING_AGE_MILESTONE',
    'Required Minimum Distribution Planning',
    'PLANNING',
    'Client approaching age 73 (RMD age) - proactive planning for required minimum distributions',
    'VIDEO_CALL',
    45,
    3000.00,
    0.20,
    FALSE,
    '{
        "email_subject": "Important: Planning for Your Required Minimum Distributions",
        "email_body": "Hi {client_first_name},\n\nAs you approach age 73, you''ll need to begin taking Required Minimum Distributions (RMDs) from your retirement accounts. I''d like to schedule a meeting to discuss:\n\n- RMD calculation and timing\n- Tax-efficient withdrawal strategies\n- Qualified Charitable Distributions (QCDs)\n- Impact on your overall financial plan\n\nWould next week work for a 45-minute video call?\n\nBest,\n{advisor_name}",
        "meeting_agenda": "1. Review current retirement account balances\n2. Calculate first-year RMD amount\n3. Discuss QCD strategy if philanthropic goals\n4. Review tax impact and withholding options\n5. Create automated distribution schedule"
    }'::jsonb,
    ARRAY['TAX_PLANNING', 'RETIREMENT_PLANNING'],
    '{"success_metric": "rmd_strategy_implemented", "target_value": 1}'::jsonb
);

-- Add comment
COMMENT ON TABLE nba_signals IS 'Detected client/market signals that trigger NBA recommendations';
COMMENT ON TABLE nba_action_catalog IS 'Pre-configured advisor actions with templates and expected outcomes';
COMMENT ON TABLE nba_recommendations IS 'ML-generated recommendations for advisors with urgency and value scores';
COMMENT ON TABLE nba_action_outcomes IS 'Execution outcomes for model training and performance tracking';
