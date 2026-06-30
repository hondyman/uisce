-- ============================================================================
-- Advisor Workflows for Alternative Investment Allocations
-- Phase 1: Complete Database Schema
-- ============================================================================
-- This migration extends the existing alternative_investments schema with
-- comprehensive pipeline management, due diligence workflows, portfolio
-- construction, monitoring, and compliance automation.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.tables 
        WHERE table_name = 'capital_events'
    ) AND NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'capital_events' AND column_name = 'investment_id'
    ) THEN
        DROP TABLE capital_events CASCADE;
    END IF;
END $$;

-- ============================================================================
-- SECTION 1: INVESTMENT OPPORTUNITY PIPELINE
-- ============================================================================

-- Centralized deal pipeline tracking
CREATE TABLE IF NOT EXISTS investment_opportunities (
    opportunity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    advisor_id UUID,
    
    -- Opportunity classification
    opportunity_type VARCHAR(50) NOT NULL CHECK (opportunity_type IN (
        'PRIVATE_EQUITY',
        'VENTURE_CAPITAL',
        'REAL_ESTATE',
        'HEDGE_FUND',
        'PRIVATE_CREDIT',
        'INFRASTRUCTURE',
        'NATURAL_RESOURCES',
        'SECONDARIES',
        'CO_INVESTMENT',
        'DIRECT_INVESTMENT'
    )),
    
    -- Fund/Investment details
    fund_name TEXT NOT NULL,
    general_partner TEXT,
    strategy VARCHAR(100),
    sub_strategy VARCHAR(100),
    vintage_year INTEGER,
    fund_size DECIMAL(15,2),
    minimum_commitment DECIMAL(15,2),
    target_commitment DECIMAL(15,2),
    
    -- Initial screening criteria
    target_irr_min DECIMAL(5,2),
    target_irr_max DECIMAL(5,2),
    target_tvpi_min DECIMAL(5,2),
    target_vintage_year_range JSONB,  -- {"min": 2025, "max": 2028}
    max_leverage_ratio DECIMAL(5,2),
    manager_aum_min DECIMAL(15,2),
    track_record_years_min INTEGER,
    
    -- Screening results
    screening_passed BOOLEAN,
    screening_reasons TEXT[],
    screening_score DECIMAL(5,2),
    screening_completed_at TIMESTAMPTZ,
    
    -- Stage gates
    current_stage VARCHAR(50) NOT NULL DEFAULT 'INTAKE' CHECK (current_stage IN (
        'INTAKE',
        'INITIAL_SCREEN',
        'DUE_DILIGENCE',
        'INVESTMENT_COMMITTEE',
        'APPROVED',
        'DOCUMENTATION',
        'COMMITTED',
        'FUNDED',
        'CLOSED_WON',
        'CLOSED_LOST',
        'ON_HOLD'
    )),
    stage_updated_at TIMESTAMPTZ DEFAULT NOW(),
    stage_history JSONB DEFAULT '[]'::JSONB,  -- Array of {stage, timestamp, user_id, notes}
    
    -- Advisor notes and attachments
    advisor_notes TEXT,
    investment_thesis TEXT,
    risk_assessment_notes TEXT,
    
    -- Document URLs
    pitch_deck_url TEXT,
    teaser_url TEXT,
    private_placement_memorandum_url TEXT,
    subscription_agreement_url TEXT,
    side_letter_url TEXT,
    due_diligence_report_url TEXT,
    
    -- Temporal workflow tracking
    workflow_instance_id TEXT,
    workflow_status VARCHAR(50),
    
    -- Expected timeline
    expected_close_date DATE,
    expected_first_call_date DATE,
    
    -- Fees
    management_fee_rate DECIMAL(5,4),
    carried_interest_rate DECIMAL(5,4),
    preferred_return_rate DECIMAL(5,4),
    
    -- Metadata
    source VARCHAR(100),  -- How opportunity was sourced
    referral_source TEXT,
    tags TEXT[],
    metadata JSONB,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID
);

CREATE INDEX IF NOT EXISTS idx_opportunities_client ON investment_opportunities(client_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_advisor ON investment_opportunities(advisor_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_stage ON investment_opportunities(current_stage);
CREATE INDEX IF NOT EXISTS idx_opportunities_type ON investment_opportunities(opportunity_type);
CREATE INDEX IF NOT EXISTS idx_opportunities_vintage ON investment_opportunities(vintage_year);
CREATE INDEX IF NOT EXISTS idx_opportunities_workflow ON investment_opportunities(workflow_instance_id);

-- ============================================================================
-- SECTION 2: DUE DILIGENCE TRACKING
-- ============================================================================

-- Due diligence checklist items
CREATE TABLE IF NOT EXISTS due_diligence_items (
    item_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID NOT NULL REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    
    -- Item details
    category VARCHAR(50) NOT NULL CHECK (category IN (
        'LEGAL',
        'FINANCIAL',
        'OPERATIONAL',
        'TAX',
        'ESG',
        'COMPLIANCE',
        'REFERENCES',
        'TRACK_RECORD',
        'TERMS',
        'RISK'
    )),
    item_name TEXT NOT NULL,
    description TEXT,
    required BOOLEAN DEFAULT TRUE,
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'IN_PROGRESS',
        'COMPLETED',
        'FLAGGED',
        'WAIVED',
        'NOT_APPLICABLE'
    )),
    
    -- Assignment
    assigned_to UUID,
    assigned_at TIMESTAMPTZ,
    due_date DATE,
    
    -- Completion
    completed_by UUID,
    completed_at TIMESTAMPTZ,
    completion_notes TEXT,
    
    -- Findings
    risk_level VARCHAR(20) CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    findings_summary TEXT,
    attachments JSONB,  -- Array of {filename, url, uploaded_at}
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dd_items_opportunity ON due_diligence_items(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_dd_items_status ON due_diligence_items(status);
CREATE INDEX IF NOT EXISTS idx_dd_items_assigned ON due_diligence_items(assigned_to);

-- Due diligence templates (reusable checklists)
CREATE TABLE IF NOT EXISTS due_diligence_templates (
    template_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name TEXT NOT NULL,
    opportunity_type VARCHAR(50) NOT NULL,
    
    -- Template items (JSONB array)
    items JSONB NOT NULL,  -- [{category, item_name, description, required}]
    
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_dd_templates_type ON due_diligence_templates(opportunity_type);

-- ============================================================================
-- SECTION 3: INVESTMENT COMMITTEE
-- ============================================================================

-- Investment committee meetings
CREATE TABLE IF NOT EXISTS investment_committee_meetings (
    meeting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    meeting_date DATE NOT NULL,
    meeting_time TIME,
    meeting_type VARCHAR(50) CHECK (meeting_type IN (
        'REGULAR',
        'SPECIAL',
        'EMERGENCY',
        'QUARTERLY_REVIEW'
    )),
    
    -- Agenda and materials
    agenda JSONB,
    meeting_materials_url TEXT,
    
    -- Attendance
    attendees JSONB,  -- [{user_id, role, attended, proxy_for}]
    quorum_met BOOLEAN,
    
    -- Minutes
    minutes_url TEXT,
    minutes_approved_at TIMESTAMPTZ,
    
    -- Status
    status VARCHAR(50) DEFAULT 'SCHEDULED' CHECK (status IN (
        'SCHEDULED',
        'IN_PROGRESS',
        'COMPLETED',
        'CANCELLED'
    )),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ic_meetings_date ON investment_committee_meetings(meeting_date);
CREATE INDEX IF NOT EXISTS idx_ic_meetings_status ON investment_committee_meetings(status);

-- Investment committee reviews (per opportunity)
CREATE TABLE IF NOT EXISTS investment_committee_reviews (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID NOT NULL REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    meeting_id UUID REFERENCES investment_committee_meetings(meeting_id),
    
    -- Review package
    package_prepared_at TIMESTAMPTZ,
    package_prepared_by UUID,
    executive_summary TEXT,
    risk_assessment JSONB,
    financial_projections JSONB,
    portfolio_impact_analysis JSONB,
    
    -- Recommendation
    staff_recommendation VARCHAR(50) CHECK (staff_recommendation IN (
        'APPROVE',
        'APPROVE_WITH_CONDITIONS',
        'DEFER',
        'REJECT'
    )),
    staff_recommendation_notes TEXT,
    recommended_amount DECIMAL(15,2),
    
    -- Committee decision
    decision VARCHAR(50) CHECK (decision IN (
        'APPROVED',
        'APPROVED_WITH_CONDITIONS',
        'DEFERRED',
        'REJECTED',
        'PENDING'
    )),
    decision_date TIMESTAMPTZ,
    decision_conditions TEXT,
    approved_amount DECIMAL(15,2),
    
    -- Voting
    votes JSONB,  -- [{member_id, vote, comments}]
    vote_passed BOOLEAN,
    
    -- Follow-up
    next_review_date DATE,
    action_items JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ic_reviews_opportunity ON investment_committee_reviews(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_ic_reviews_meeting ON investment_committee_reviews(meeting_id);
CREATE INDEX IF NOT EXISTS idx_ic_reviews_decision ON investment_committee_reviews(decision);

-- ============================================================================
-- SECTION 4: PORTFOLIO CONSTRUCTION & ALLOCATION
-- ============================================================================

-- Asset class weights and targets
CREATE TABLE IF NOT EXISTS client_allocation_targets (
    target_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    -- Asset class targets (JSONB for flexibility)
    target_allocations JSONB NOT NULL,  -- {"PRIVATE_EQUITY": 0.15, "VENTURE_CAPITAL": 0.05, ...}
    tolerance_band_pct DECIMAL(5,2) DEFAULT 2.0,  -- +/- 2% tolerance before rebalance
    
    -- Alternative investment limits
    max_alternatives_pct DECIMAL(5,2) DEFAULT 20.0,
    max_single_fund_pct DECIMAL(5,2) DEFAULT 5.0,
    max_single_manager_pct DECIMAL(5,2) DEFAULT 10.0,
    max_vintage_concentration_pct DECIMAL(5,2) DEFAULT 25.0,
    
    -- Liquidity requirements
    min_liquid_assets_pct DECIMAL(5,2) DEFAULT 20.0,
    liquidity_horizon_months INTEGER DEFAULT 12,
    
    -- Risk constraints
    target_risk_score DECIMAL(5,2),
    max_leverage_ratio DECIMAL(5,2),
    
    -- IPS reference
    investment_policy_statement_url TEXT,
    ips_approval_date DATE,
    
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_allocation_targets_client ON client_allocation_targets(client_id);
CREATE INDEX IF NOT EXISTS idx_allocation_targets_effective ON client_allocation_targets(effective_date);

-- Allocation rebalancing triggers and alerts
CREATE TABLE IF NOT EXISTS allocation_rebalance_triggers (
    trigger_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    -- Asset class involved
    asset_class VARCHAR(50) NOT NULL,
    sub_asset_class VARCHAR(50),
    
    -- Current state
    current_allocation_pct DECIMAL(5,2) NOT NULL,
    target_allocation_pct DECIMAL(5,2) NOT NULL,
    tolerance_band_pct DECIMAL(5,2) DEFAULT 2.0,
    deviation_pct DECIMAL(5,2) NOT NULL,
    
    -- Dollar amounts
    current_value DECIMAL(15,2),
    target_value DECIMAL(15,2),
    required_adjustment DECIMAL(15,2),
    
    -- Trigger details
    trigger_type VARCHAR(50) NOT NULL CHECK (trigger_type IN (
        'DRIFT_EXCEEDED',
        'LIQUIDITY_EVENT',
        'MARKET_CONDITION',
        'CAPITAL_CALL',
        'DISTRIBUTION',
        'NEW_COMMITMENT',
        'REVALUATION',
        'CLIENT_REQUEST'
    )),
    trigger_severity VARCHAR(20) CHECK (trigger_severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    trigger_fired_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Recommended action
    recommended_action JSONB,  -- {action, trades[], urgency, notes}
    
    -- Resolution
    status VARCHAR(50) DEFAULT 'OPEN' CHECK (status IN (
        'OPEN',
        'ACKNOWLEDGED',
        'IN_PROGRESS',
        'RESOLVED',
        'DISMISSED'
    )),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,
    
    -- Workflow
    workflow_instance_id TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rebalance_triggers_client ON allocation_rebalance_triggers(client_id);
CREATE INDEX IF NOT EXISTS idx_rebalance_triggers_status ON allocation_rebalance_triggers(status);
CREATE INDEX IF NOT EXISTS idx_rebalance_triggers_fired ON allocation_rebalance_triggers(trigger_fired_at);

-- Allocation recommendations (AI-generated)
CREATE TABLE IF NOT EXISTS allocation_recommendations (
    recommendation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id),
    
    -- Recommendation details
    recommended_amount DECIMAL(15,2),
    recommended_pct_of_portfolio DECIMAL(5,2),
    
    -- Analysis
    current_alt_exposure_pct DECIMAL(5,2),
    post_allocation_alt_exposure_pct DECIMAL(5,2),
    diversification_benefit_score DECIMAL(5,2),
    liquidity_impact_score DECIMAL(5,2),
    risk_budget_utilization_pct DECIMAL(5,2),
    
    -- Rationale (AI-generated)
    rationale TEXT,
    pros JSONB,
    cons JSONB,
    risks JSONB,
    
    -- Scoring
    overall_score DECIMAL(5,2),
    fit_score DECIMAL(5,2),
    timing_score DECIMAL(5,2),
    
    -- Model info
    model_version TEXT,
    model_confidence DECIMAL(5,4),
    
    -- Status
    status VARCHAR(50) DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'ACCEPTED',
        'REJECTED',
        'MODIFIED',
        'EXPIRED'
    )),
    advisor_decision_at TIMESTAMPTZ,
    advisor_decision_by UUID,
    advisor_notes TEXT,
    
    -- Final amount if modified
    final_amount DECIMAL(15,2),
    
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alloc_recs_client ON allocation_recommendations(client_id);
CREATE INDEX IF NOT EXISTS idx_alloc_recs_opportunity ON allocation_recommendations(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_alloc_recs_status ON allocation_recommendations(status);

-- ============================================================================
-- SECTION 5: CAPITAL EVENTS (Enhanced from existing capital_calls table)
-- ============================================================================

-- Unified capital events table
CREATE TABLE IF NOT EXISTS capital_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID,  -- References alternative_investments(investment_id)
    client_id UUID NOT NULL,
    
    -- Event type
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN (
        'CAPITAL_CALL',
        'DISTRIBUTION',
        'REVALUATION',
        'EXIT',
        'RECALLABLE',
        'EQUALIZING_CALL',
        'MANAGEMENT_FEE',
        'CARRIED_INTEREST'
    )),
    
    -- Dates
    notice_date DATE,
    due_date DATE,
    settlement_date DATE,
    
    -- Amounts
    amount DECIMAL(15,2) NOT NULL,
    amount_funded DECIMAL(15,2),
    amount_pending DECIMAL(15,2),
    
    -- Distribution specifics
    distribution_type VARCHAR(50) CHECK (distribution_type IN (
        'INCOME',
        'RETURN_OF_CAPITAL',
        'REALIZED_GAIN',
        'UNREALIZED_GAIN',
        'RECALLABLE'
    )),
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'ACKNOWLEDGED',
        'SCHEDULED',
        'FUNDED',
        'PAID',
        'PARTIAL',
        'OVERDUE',
        'CANCELLED'
    )),
    
    -- Liquidity management
    liquidity_check_passed BOOLEAN,
    funding_source_account UUID,
    funding_notes TEXT,
    
    -- Alerts
    alert_sent_at TIMESTAMPTZ,
    reminder_sent_at TIMESTAMPTZ,
    escalation_sent_at TIMESTAMPTZ,
    
    -- Temporal workflow tracking
    workflow_instance_id TEXT,
    workflow_status VARCHAR(50),
    
    -- Document reference
    notice_document_url TEXT,
    
    -- Processing
    processed_at TIMESTAMPTZ,
    processed_by UUID,
    
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_capital_events_investment ON capital_events(investment_id);
CREATE INDEX IF NOT EXISTS idx_capital_events_client ON capital_events(client_id);
CREATE INDEX IF NOT EXISTS idx_capital_events_type ON capital_events(event_type);
CREATE INDEX IF NOT EXISTS idx_capital_events_status ON capital_events(status);
CREATE INDEX IF NOT EXISTS idx_capital_events_due_date ON capital_events(due_date);
CREATE INDEX IF NOT EXISTS idx_capital_events_workflow ON capital_events(workflow_instance_id);

-- ============================================================================
-- SECTION 6: POST-INVESTMENT MONITORING
-- ============================================================================

-- Quarterly review tracking
CREATE TABLE IF NOT EXISTS quarterly_reviews (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    
    -- Period
    review_period VARCHAR(10) NOT NULL,  -- e.g., '2025-Q1'
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    
    -- Performance summary
    portfolio_return_pct DECIMAL(5,2),
    alt_portfolio_return_pct DECIMAL(5,2),
    benchmark_return_pct DECIMAL(5,2),
    alpha DECIMAL(5,2),
    
    -- Position summary
    total_alt_aum DECIMAL(15,2),
    total_unfunded_commitments DECIMAL(15,2),
    alt_allocation_pct DECIMAL(5,2),
    
    -- Liquidity analysis
    upcoming_capital_calls_90d DECIMAL(15,2),
    available_liquidity DECIMAL(15,2),
    liquidity_coverage_ratio DECIMAL(5,2),
    
    -- Risk flags
    risk_flags JSONB,  -- [{investment_id, flag_type, severity, description}]
    
    -- Report
    report_generated_at TIMESTAMPTZ,
    report_url TEXT,
    report_sent_at TIMESTAMPTZ,
    
    -- Client meeting
    meeting_scheduled_at TIMESTAMPTZ,
    meeting_completed_at TIMESTAMPTZ,
    meeting_notes TEXT,
    action_items JSONB,
    
    -- Workflow
    workflow_instance_id TEXT,
    
    -- Status
    status VARCHAR(50) DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'IN_PROGRESS',
        'REPORT_GENERATED',
        'MEETING_SCHEDULED',
        'COMPLETED'
    )),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quarterly_reviews_client ON quarterly_reviews(client_id);
CREATE INDEX IF NOT EXISTS idx_quarterly_reviews_period ON quarterly_reviews(review_period);
CREATE INDEX IF NOT EXISTS idx_quarterly_reviews_status ON quarterly_reviews(status);

-- Manager updates and communications
CREATE TABLE IF NOT EXISTS manager_updates (
    update_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID,  -- References alternative_investments(investment_id)
    
    -- Update type
    update_type VARCHAR(50) NOT NULL CHECK (update_type IN (
        'QUARTERLY_LETTER',
        'CAPITAL_ACCOUNT_STATEMENT',
        'NAV_UPDATE',
        'PORTFOLIO_UPDATE',
        'PERSONNEL_CHANGE',
        'STRATEGY_CHANGE',
        'REGULATORY_NOTICE',
        'K1_TAX_DOCUMENT',
        'OTHER'
    )),
    
    -- Content
    title TEXT,
    summary TEXT,
    document_url TEXT,
    document_date DATE,
    
    -- Key metrics extracted
    reported_nav DECIMAL(15,2),
    reported_irr DECIMAL(5,2),
    reported_tvpi DECIMAL(5,2),
    
    -- Processing
    processed_at TIMESTAMPTZ,
    processed_by UUID,
    extracted_data JSONB,  -- AI/OCR extracted data
    
    -- Alerts generated
    alert_generated BOOLEAN DEFAULT FALSE,
    alert_reason TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_manager_updates_investment ON manager_updates(investment_id);
CREATE INDEX IF NOT EXISTS idx_manager_updates_type ON manager_updates(update_type);
CREATE INDEX IF NOT EXISTS idx_manager_updates_date ON manager_updates(document_date);

-- ============================================================================
-- SECTION 7: COMPLIANCE & REGULATORY
-- ============================================================================

-- Regulatory filings automation
CREATE TABLE IF NOT EXISTS regulatory_filings (
    filing_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Filing type
    filing_type VARCHAR(50) NOT NULL CHECK (filing_type IN (
        'FORM_ADV',
        'FORM_PF',
        'FORM_13F',
        'FORM_13D',
        'FORM_13G',
        'FORM_D',
        'SCHEDULE_K1_SUMMARY',
        'CUSIP_REPORT',
        'QUALIFIED_CLIENT_CENSUS',
        'OTHER'
    )),
    
    -- Period
    reporting_period VARCHAR(20),  -- e.g., '2025-Q1', '2025'
    period_end_date DATE,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT' CHECK (status IN (
        'DRAFT',
        'IN_PREPARATION',
        'REVIEW',
        'SUBMITTED',
        'ACCEPTED',
        'AMENDED',
        'REJECTED'
    )),
    
    -- Due date
    due_date DATE,
    submitted_date DATE,
    
    -- Content
    filing_data JSONB,
    
    -- Alternative-specific disclosures
    qualified_clients_count INTEGER,
    performance_fee_clients_count INTEGER,
    illiquid_assets_value DECIMAL(15,2),
    side_pocket_value DECIMAL(15,2),
    total_alternative_aum DECIMAL(15,2),
    
    -- Documents
    filing_document_url TEXT,
    confirmation_url TEXT,
    
    -- Workflow
    workflow_instance_id TEXT,
    
    -- Audit
    generated_at TIMESTAMPTZ,
    generated_by UUID,
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID,
    submitted_by UUID,
    
    notes TEXT,
    metadata JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reg_filings_type ON regulatory_filings(filing_type);
CREATE INDEX IF NOT EXISTS idx_reg_filings_period ON regulatory_filings(reporting_period);
CREATE INDEX IF NOT EXISTS idx_reg_filings_status ON regulatory_filings(status);
CREATE INDEX IF NOT EXISTS idx_reg_filings_due ON regulatory_filings(due_date);

-- Compliance checkpoints for opportunities
CREATE TABLE IF NOT EXISTS compliance_checkpoints (
    checkpoint_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id) ON DELETE CASCADE,
    
    -- Checkpoint type
    checkpoint_type VARCHAR(50) NOT NULL CHECK (checkpoint_type IN (
        'ACCREDITED_INVESTOR_VERIFICATION',
        'QP_VERIFICATION',
        'AML_KYC_CHECK',
        'SANCTIONS_SCREENING',
        'CONFLICT_OF_INTEREST_CHECK',
        'CONCENTRATION_LIMIT_CHECK',
        'LIQUIDITY_REQUIREMENT_CHECK',
        'IPS_COMPLIANCE_CHECK',
        'REGULATORY_ELIGIBILITY',
        'FIDUCIARY_REVIEW'
    )),
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'PASSED',
        'FAILED',
        'WAIVED',
        'EXPIRED'
    )),
    
    -- Details
    check_performed_at TIMESTAMPTZ,
    check_performed_by UUID,
    result_details JSONB,
    expiration_date DATE,
    
    -- Override/waiver
    waived_by UUID,
    waived_at TIMESTAMPTZ,
    waiver_reason TEXT,
    waiver_approved_by UUID,
    
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_checkpoints_opportunity ON compliance_checkpoints(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_compliance_checkpoints_type ON compliance_checkpoints(checkpoint_type);
CREATE INDEX IF NOT EXISTS idx_compliance_checkpoints_status ON compliance_checkpoints(status);

-- ============================================================================
-- SECTION 8: E-SIGNATURE & DOCUMENTS
-- ============================================================================

-- E-signature workflows
CREATE TABLE IF NOT EXISTS esignature_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id),
    
    -- Document details
    document_type VARCHAR(50) NOT NULL CHECK (document_type IN (
        'SUBSCRIPTION_AGREEMENT',
        'SIDE_LETTER',
        'INVESTOR_QUESTIONNAIRE',
        'IRS_W9',
        'IRS_W8BEN',
        'ACCREDITED_INVESTOR_CERT',
        'QUALIFIED_PURCHASER_CERT',
        'AML_CERTIFICATION',
        'OTHER'
    )),
    document_name TEXT,
    document_url TEXT,
    
    -- Signers
    signers JSONB NOT NULL,  -- [{email, name, role, order, status, signed_at}]
    
    -- Provider tracking
    provider VARCHAR(50),  -- 'DOCUSIGN', 'ADOBE_SIGN', 'HELLOSIGN'
    external_envelope_id TEXT,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT' CHECK (status IN (
        'DRAFT',
        'SENT',
        'VIEWED',
        'PARTIALLY_SIGNED',
        'COMPLETED',
        'DECLINED',
        'VOIDED',
        'EXPIRED'
    )),
    
    -- Dates
    sent_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Completed document
    signed_document_url TEXT,
    
    -- Workflow
    workflow_instance_id TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX IF NOT EXISTS idx_esig_requests_opportunity ON esignature_requests(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_esig_requests_status ON esignature_requests(status);
CREATE INDEX IF NOT EXISTS idx_esig_requests_envelope ON esignature_requests(external_envelope_id);

-- ============================================================================
-- SECTION 9: ADVISOR NOTIFICATIONS & TASKS
-- ============================================================================

-- Advisor task queue (unified task management)
CREATE TABLE IF NOT EXISTS advisor_tasks (
    task_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    advisor_id UUID NOT NULL,
    client_id UUID,
    opportunity_id UUID REFERENCES investment_opportunities(opportunity_id),
    
    -- Task type
    task_type VARCHAR(50) NOT NULL CHECK (task_type IN (
        'OPPORTUNITY_REVIEW',
        'DUE_DILIGENCE_ITEM',
        'CAPITAL_CALL_FUNDING',
        'REBALANCE_REVIEW',
        'QUARTERLY_REVIEW',
        'DOCUMENT_REVIEW',
        'CLIENT_MEETING',
        'COMMITTEE_PREPARATION',
        'COMPLIANCE_CHECK',
        'ESIGNATURE_FOLLOWUP',
        'CUSTOM'
    )),
    
    -- Task details
    title TEXT NOT NULL,
    description TEXT,
    
    -- Priority
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM' CHECK (priority IN (
        'CRITICAL',
        'HIGH',
        'MEDIUM',
        'LOW'
    )),
    
    -- Dates
    due_date TIMESTAMPTZ,
    reminder_date TIMESTAMPTZ,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'IN_PROGRESS',
        'COMPLETED',
        'DEFERRED',
        'CANCELLED'
    )),
    
    -- Completion
    completed_at TIMESTAMPTZ,
    completed_by UUID,
    completion_notes TEXT,
    
    -- Related entity references
    related_entity_type VARCHAR(50),
    related_entity_id UUID,
    
    -- Workflow
    workflow_instance_id TEXT,
    auto_generated BOOLEAN DEFAULT FALSE,
    
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_advisor_tasks_advisor ON advisor_tasks(advisor_id);
CREATE INDEX IF NOT EXISTS idx_advisor_tasks_client ON advisor_tasks(client_id);
CREATE INDEX IF NOT EXISTS idx_advisor_tasks_status ON advisor_tasks(status);
CREATE INDEX IF NOT EXISTS idx_advisor_tasks_priority ON advisor_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_advisor_tasks_due ON advisor_tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_advisor_tasks_type ON advisor_tasks(task_type);

-- ============================================================================
-- SECTION 10: ANALYTICS & METRICS
-- ============================================================================

-- Pipeline metrics snapshot (materialized for dashboard performance)
CREATE TABLE IF NOT EXISTS pipeline_metrics_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_date DATE NOT NULL,
    advisor_id UUID,
    
    -- Pipeline counts by stage
    intake_count INTEGER DEFAULT 0,
    initial_screen_count INTEGER DEFAULT 0,
    due_diligence_count INTEGER DEFAULT 0,
    committee_count INTEGER DEFAULT 0,
    documentation_count INTEGER DEFAULT 0,
    committed_count INTEGER DEFAULT 0,
    
    -- Dollar amounts
    total_pipeline_amount DECIMAL(15,2),
    committed_amount_ytd DECIMAL(15,2),
    funded_amount_ytd DECIMAL(15,2),
    
    -- Conversion metrics
    avg_days_intake_to_commit INTEGER,
    conversion_rate_pct DECIMAL(5,2),
    win_rate_pct DECIMAL(5,2),
    
    -- Activity metrics
    opportunities_added INTEGER,
    opportunities_closed_won INTEGER,
    opportunities_closed_lost INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_date ON pipeline_metrics_snapshots(snapshot_date);
CREATE INDEX IF NOT EXISTS idx_pipeline_metrics_advisor ON pipeline_metrics_snapshots(advisor_id);

-- Portfolio alternative metrics snapshot
CREATE TABLE IF NOT EXISTS portfolio_alt_metrics_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_date DATE NOT NULL,
    client_id UUID NOT NULL,
    
    -- AUM metrics
    total_alt_aum DECIMAL(15,2),
    total_unfunded_commitments DECIMAL(15,2),
    dry_powder_pct DECIMAL(5,2),
    
    -- Allocation metrics
    alt_allocation_pct DECIMAL(5,2),
    pe_allocation_pct DECIMAL(5,2),
    vc_allocation_pct DECIMAL(5,2),
    re_allocation_pct DECIMAL(5,2),
    hf_allocation_pct DECIMAL(5,2),
    pc_allocation_pct DECIMAL(5,2),
    
    -- Performance metrics
    total_irr DECIMAL(5,2),
    total_tvpi DECIMAL(5,2),
    total_dpi DECIMAL(5,2),
    pme_alpha DECIMAL(5,2),
    
    -- Vintage year distribution
    vintage_distribution JSONB,  -- {"2020": 0.15, "2021": 0.20, ...}
    
    -- Risk metrics
    concentration_score DECIMAL(5,2),
    liquidity_score DECIMAL(5,2),
    
    -- Upcoming events
    capital_calls_30d DECIMAL(15,2),
    capital_calls_90d DECIMAL(15,2),
    expected_distributions_90d DECIMAL(15,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_portfolio_alt_metrics_date ON portfolio_alt_metrics_snapshots(snapshot_date);
CREATE INDEX IF NOT EXISTS idx_portfolio_alt_metrics_client ON portfolio_alt_metrics_snapshots(client_id);

-- ============================================================================
-- SECTION 11: AUDIT TRAIL
-- ============================================================================

-- Comprehensive audit trail for all actions
CREATE TABLE IF NOT EXISTS alt_investment_audit_log (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- What changed
    entity_type VARCHAR(50) NOT NULL,  -- 'OPPORTUNITY', 'CAPITAL_EVENT', etc.
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,  -- 'CREATE', 'UPDATE', 'DELETE', 'STATUS_CHANGE', etc.
    
    -- Change details
    field_changed VARCHAR(100),
    old_value TEXT,
    new_value TEXT,
    change_summary TEXT,
    
    -- Who and when
    performed_by UUID NOT NULL,
    performed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    session_id TEXT,
    
    -- Related entities
    client_id UUID,
    opportunity_id UUID,
    
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON alt_investment_audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_performed ON alt_investment_audit_log(performed_at);
CREATE INDEX IF NOT EXISTS idx_audit_log_user ON alt_investment_audit_log(performed_by);
CREATE INDEX IF NOT EXISTS idx_audit_log_client ON alt_investment_audit_log(client_id);

-- ============================================================================
-- SECTION 12: HELPER FUNCTIONS
-- ============================================================================

-- Function: Automated screening for new opportunities
CREATE OR REPLACE FUNCTION screen_investment_opportunity()
RETURNS TRIGGER AS $$
DECLARE
    screening_passed BOOLEAN := TRUE;
    screening_reasons TEXT[] := ARRAY[]::TEXT[];
    client_liquid_assets DECIMAL(15,2);
    screening_score DECIMAL(5,2) := 100;
BEGIN
    -- Get client's liquid assets
    SELECT COALESCE(SUM(current_nav), 0) INTO client_liquid_assets
    FROM alternative_investments ai
    WHERE ai.client_id = NEW.client_id;
    
    -- Add check for portfolio_summary if it exists
    -- This is a placeholder - adjust based on actual schema
    
    -- Check 1: Minimum commitment vs client capacity (max 10% of AUM per position)
    IF NEW.minimum_commitment > (client_liquid_assets * 0.10) THEN
        screening_passed := FALSE;
        screening_reasons := screening_reasons || 'Minimum commitment exceeds 10% of alternative AUM';
        screening_score := screening_score - 20;
    END IF;
    
    -- Check 2: Vintage year alignment (2025-2028 target)
    IF NEW.vintage_year IS NOT NULL AND (NEW.vintage_year < 2025 OR NEW.vintage_year > 2028) THEN
        screening_reasons := screening_reasons || 'Vintage year outside preferred 2025-2028 range';
        screening_score := screening_score - 10;
    END IF;
    
    -- Check 3: Manager track record
    IF NEW.track_record_years_min IS NOT NULL AND NEW.track_record_years_min < 5 THEN
        screening_reasons := screening_reasons || 'Manager track record less than 5 years';
        screening_score := screening_score - 15;
    END IF;
    
    -- Check 4: Target IRR reasonableness
    IF NEW.target_irr_min IS NOT NULL AND NEW.target_irr_min > 35 THEN
        screening_reasons := screening_reasons || 'Target IRR appears unrealistically high';
        screening_score := screening_score - 10;
    END IF;
    
    -- Update the record
    NEW.screening_passed := screening_passed;
    NEW.screening_reasons := screening_reasons;
    NEW.screening_score := GREATEST(screening_score, 0);
    NEW.screening_completed_at := NOW();
    
    -- Auto-advance stage if passed
    IF screening_passed AND NEW.current_stage = 'INTAKE' THEN
        NEW.current_stage := 'INITIAL_SCREEN';
        NEW.stage_updated_at := NOW();
        NEW.stage_history := NEW.stage_history || jsonb_build_object(
            'stage', 'INITIAL_SCREEN',
            'timestamp', NOW(),
            'notes', 'Automated screening passed'
        );
    ELSIF NOT screening_passed AND NEW.current_stage = 'INTAKE' THEN
        -- Keep in intake but flag for manual review
        NEW.stage_history := NEW.stage_history || jsonb_build_object(
            'stage', 'INTAKE',
            'timestamp', NOW(),
            'notes', 'Automated screening flagged issues: ' || array_to_string(screening_reasons, '; ')
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for new opportunities
DROP TRIGGER IF EXISTS trg_screen_investment_opportunity ON investment_opportunities;
CREATE TRIGGER trg_screen_investment_opportunity
    BEFORE INSERT ON investment_opportunities
    FOR EACH ROW
    EXECUTE FUNCTION screen_investment_opportunity();

-- Function: Check allocation drift
CREATE OR REPLACE FUNCTION check_allocation_drift()
RETURNS TABLE (
    client_id UUID,
    asset_class VARCHAR(50),
    current_allocation_pct DECIMAL(5,2),
    target_allocation_pct DECIMAL(5,2),
    deviation_pct DECIMAL(5,2),
    action_required BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    WITH client_allocations AS (
        SELECT 
            ai.client_id,
            ai.investment_type as asset_class,
            SUM(ai.current_nav) as current_value
        FROM alternative_investments ai
        WHERE ai.current_nav IS NOT NULL
        GROUP BY ai.client_id, ai.investment_type
    ),
    client_totals AS (
        SELECT 
            client_id,
            SUM(current_value) as total_value
        FROM client_allocations
        GROUP BY client_id
    ),
    current_pcts AS (
        SELECT 
            ca.client_id,
            ca.asset_class,
            (ca.current_value / NULLIF(ct.total_value, 0) * 100)::DECIMAL(5,2) as current_pct
        FROM client_allocations ca
        JOIN client_totals ct ON ca.client_id = ct.client_id
    )
    SELECT 
        cp.client_id,
        cp.asset_class::VARCHAR(50),
        cp.current_pct as current_allocation_pct,
        COALESCE((cat.target_allocations->>cp.asset_class)::DECIMAL(5,2) * 100, 15)::DECIMAL(5,2) as target_allocation_pct,
        ABS(cp.current_pct - COALESCE((cat.target_allocations->>cp.asset_class)::DECIMAL(5,2) * 100, 15))::DECIMAL(5,2) as deviation_pct,
        ABS(cp.current_pct - COALESCE((cat.target_allocations->>cp.asset_class)::DECIMAL(5,2) * 100, 15)) > COALESCE(cat.tolerance_band_pct, 2.0) as action_required
    FROM current_pcts cp
    LEFT JOIN client_allocation_targets cat ON cp.client_id = cat.client_id
        AND cat.effective_date <= CURRENT_DATE
        AND (cat.end_date IS NULL OR cat.end_date > CURRENT_DATE)
    ORDER BY deviation_pct DESC;
END;
$$ LANGUAGE plpgsql;

-- Function: Generate pipeline metrics snapshot
CREATE OR REPLACE FUNCTION generate_pipeline_metrics_snapshot(p_advisor_id UUID DEFAULT NULL)
RETURNS UUID AS $$
DECLARE
    v_snapshot_id UUID;
BEGIN
    INSERT INTO pipeline_metrics_snapshots (
        snapshot_date,
        advisor_id,
        intake_count,
        initial_screen_count,
        due_diligence_count,
        committee_count,
        documentation_count,
        committed_count,
        total_pipeline_amount,
        committed_amount_ytd,
        funded_amount_ytd,
        opportunities_added,
        opportunities_closed_won,
        opportunities_closed_lost
    )
    SELECT 
        CURRENT_DATE,
        p_advisor_id,
        COUNT(*) FILTER (WHERE current_stage = 'INTAKE'),
        COUNT(*) FILTER (WHERE current_stage = 'INITIAL_SCREEN'),
        COUNT(*) FILTER (WHERE current_stage = 'DUE_DILIGENCE'),
        COUNT(*) FILTER (WHERE current_stage = 'INVESTMENT_COMMITTEE'),
        COUNT(*) FILTER (WHERE current_stage = 'DOCUMENTATION'),
        COUNT(*) FILTER (WHERE current_stage = 'COMMITTED'),
        SUM(target_commitment) FILTER (WHERE current_stage NOT IN ('CLOSED_WON', 'CLOSED_LOST')),
        SUM(target_commitment) FILTER (WHERE current_stage IN ('COMMITTED', 'FUNDED', 'CLOSED_WON') AND created_at >= DATE_TRUNC('year', CURRENT_DATE)),
        SUM(target_commitment) FILTER (WHERE current_stage IN ('FUNDED', 'CLOSED_WON') AND created_at >= DATE_TRUNC('year', CURRENT_DATE)),
        COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'),
        COUNT(*) FILTER (WHERE current_stage = 'CLOSED_WON' AND stage_updated_at >= CURRENT_DATE - INTERVAL '30 days'),
        COUNT(*) FILTER (WHERE current_stage = 'CLOSED_LOST' AND stage_updated_at >= CURRENT_DATE - INTERVAL '30 days')
    FROM investment_opportunities
    WHERE p_advisor_id IS NULL OR advisor_id = p_advisor_id
    RETURNING snapshot_id INTO v_snapshot_id;
    
    RETURN v_snapshot_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SECTION 13: SEED DATA - Due Diligence Templates
-- ============================================================================

INSERT INTO due_diligence_templates (template_name, opportunity_type, items, is_default) VALUES
('Private Equity Standard', 'PRIVATE_EQUITY', '[
    {"category": "LEGAL", "item_name": "Limited Partnership Agreement Review", "description": "Review LPA terms, governance rights, and fee structure", "required": true},
    {"category": "LEGAL", "item_name": "Side Letter Review", "description": "Review any negotiated side letter terms", "required": true},
    {"category": "LEGAL", "item_name": "Regulatory Filings Check", "description": "Verify Form ADV, Form PF filings", "required": true},
    {"category": "FINANCIAL", "item_name": "Historical Performance Analysis", "description": "Analyze IRR, TVPI, DPI across prior funds", "required": true},
    {"category": "FINANCIAL", "item_name": "Fee Analysis", "description": "Review management fees, carried interest, other expenses", "required": true},
    {"category": "FINANCIAL", "item_name": "Audited Financial Statements", "description": "Review last 3 years of audited financials", "required": true},
    {"category": "OPERATIONAL", "item_name": "Operations Due Diligence", "description": "Assess back-office operations, fund administrator", "required": true},
    {"category": "OPERATIONAL", "item_name": "Cybersecurity Assessment", "description": "Review IT security policies and SOC reports", "required": true},
    {"category": "TAX", "item_name": "UBTI Analysis", "description": "Assess potential UBTI exposure for tax-exempt investors", "required": true},
    {"category": "TAX", "item_name": "FIRPTA Analysis", "description": "Review potential FIRPTA withholding requirements", "required": false},
    {"category": "ESG", "item_name": "ESG Policy Review", "description": "Assess ESG integration in investment process", "required": false},
    {"category": "REFERENCES", "item_name": "Reference Calls", "description": "Conduct reference calls with LPs and portfolio companies", "required": true},
    {"category": "TRACK_RECORD", "item_name": "Attribution Analysis", "description": "Analyze deal-level returns and team attribution", "required": true},
    {"category": "RISK", "item_name": "Key Person Risk Assessment", "description": "Assess key person provisions and team stability", "required": true}
]'::JSONB, TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO due_diligence_templates (template_name, opportunity_type, items, is_default) VALUES
('Venture Capital Standard', 'VENTURE_CAPITAL', '[
    {"category": "LEGAL", "item_name": "Fund Documents Review", "description": "Review LPA and subscription documents", "required": true},
    {"category": "FINANCIAL", "item_name": "Portfolio Company Analysis", "description": "Review current portfolio companies and valuations", "required": true},
    {"category": "FINANCIAL", "item_name": "Historical Performance", "description": "Analyze prior fund performance and realized exits", "required": true},
    {"category": "OPERATIONAL", "item_name": "Investment Process Review", "description": "Understand deal sourcing and selection process", "required": true},
    {"category": "TRACK_RECORD", "item_name": "Deal Attribution", "description": "Review team member contributions to key deals", "required": true},
    {"category": "REFERENCES", "item_name": "Founder References", "description": "Speak with portfolio company founders", "required": true},
    {"category": "RISK", "item_name": "Concentration Risk", "description": "Assess sector and stage concentration", "required": true}
]'::JSONB, TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO due_diligence_templates (template_name, opportunity_type, items, is_default) VALUES
('Hedge Fund Standard', 'HEDGE_FUND', '[
    {"category": "LEGAL", "item_name": "Offering Documents Review", "description": "Review PPM, subscription agreement, side letter", "required": true},
    {"category": "LEGAL", "item_name": "Liquidity Terms", "description": "Analyze lock-up, gates, and redemption terms", "required": true},
    {"category": "FINANCIAL", "item_name": "Performance Analysis", "description": "Review risk-adjusted returns, drawdowns, Sharpe ratio", "required": true},
    {"category": "FINANCIAL", "item_name": "Leverage Analysis", "description": "Assess leverage usage and risk management", "required": true},
    {"category": "OPERATIONAL", "item_name": "ODD Report", "description": "Conduct operational due diligence review", "required": true},
    {"category": "OPERATIONAL", "item_name": "Counterparty Risk", "description": "Review prime broker and counterparty relationships", "required": true},
    {"category": "COMPLIANCE", "item_name": "Regulatory Check", "description": "Verify registration status and regulatory history", "required": true},
    {"category": "RISK", "item_name": "Stress Testing", "description": "Review historical performance in stress scenarios", "required": true}
]'::JSONB, TRUE)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SECTION 14: VIEWS FOR DASHBOARD
-- ============================================================================

-- View: Active pipeline summary
CREATE OR REPLACE VIEW v_pipeline_summary AS
SELECT 
    current_stage,
    opportunity_type,
    COUNT(*) as opportunity_count,
    SUM(target_commitment) as total_commitment,
    AVG(EXTRACT(DAY FROM NOW() - created_at))::INTEGER as avg_days_in_pipeline,
    MIN(created_at) as oldest_opportunity
FROM investment_opportunities
WHERE current_stage NOT IN ('CLOSED_WON', 'CLOSED_LOST')
GROUP BY current_stage, opportunity_type
ORDER BY 
    CASE current_stage 
        WHEN 'INTAKE' THEN 1
        WHEN 'INITIAL_SCREEN' THEN 2
        WHEN 'DUE_DILIGENCE' THEN 3
        WHEN 'INVESTMENT_COMMITTEE' THEN 4
        WHEN 'APPROVED' THEN 5
        WHEN 'DOCUMENTATION' THEN 6
        WHEN 'COMMITTED' THEN 7
        WHEN 'FUNDED' THEN 8
        ELSE 9
    END;

-- View: Upcoming capital events
CREATE OR REPLACE VIEW v_upcoming_capital_events AS
SELECT 
    ce.event_id,
    ce.client_id,
    ai.fund_name,
    ce.event_type,
    ce.due_date,
    ce.amount,
    ce.status,
    ce.liquidity_check_passed,
    (ce.due_date - CURRENT_DATE) as days_until_due
FROM capital_events ce
JOIN alternative_investments ai ON ce.investment_id = ai.investment_id
WHERE ce.due_date >= CURRENT_DATE
  AND ce.status NOT IN ('FUNDED', 'PAID', 'CANCELLED')
ORDER BY ce.due_date;

-- View: Advisor task queue
CREATE OR REPLACE VIEW v_advisor_task_queue AS
SELECT 
    at.task_id,
    at.advisor_id,
    at.client_id,
    at.task_type,
    at.title,
    at.priority,
    at.due_date,
    at.status,
    io.fund_name as related_fund,
    io.current_stage as opportunity_stage,
    CASE 
        WHEN at.due_date < NOW() THEN 'OVERDUE'
        WHEN at.due_date < NOW() + INTERVAL '1 day' THEN 'DUE_TODAY'
        WHEN at.due_date < NOW() + INTERVAL '7 days' THEN 'DUE_THIS_WEEK'
        ELSE 'UPCOMING'
    END as urgency
FROM advisor_tasks at
LEFT JOIN investment_opportunities io ON at.opportunity_id = io.opportunity_id
WHERE at.status IN ('PENDING', 'IN_PROGRESS')
ORDER BY 
    CASE at.priority 
        WHEN 'CRITICAL' THEN 1 
        WHEN 'HIGH' THEN 2 
        WHEN 'MEDIUM' THEN 3 
        ELSE 4 
    END,
    at.due_date NULLS LAST;

-- ============================================================================
-- Grant appropriate permissions
-- ============================================================================
-- Uncomment and modify as needed for your environment:
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO your_app_role;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO your_app_role;

COMMENT ON TABLE investment_opportunities IS 'Centralized deal pipeline tracking for alternative investment opportunities';
COMMENT ON TABLE due_diligence_items IS 'Individual due diligence checklist items for each opportunity';
COMMENT ON TABLE capital_events IS 'Unified capital events (calls, distributions, etc.) across all investments';
COMMENT ON TABLE allocation_recommendations IS 'AI-generated allocation recommendations for clients';
COMMENT ON TABLE regulatory_filings IS 'Regulatory filing tracking and automation';
COMMENT ON TABLE advisor_tasks IS 'Unified task management for advisors';
