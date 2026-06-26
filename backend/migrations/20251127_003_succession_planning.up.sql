-- Advisor Succession & Continuity Planning Schema
-- Phase 3: Succession Planning

-- ===========================
-- ADVISOR PRACTICE METRICS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS advisor_practice_metrics (
    advisor_id UUID PRIMARY KEY,
    evaluation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    
    -- Book of business metrics
    total_aum DECIMAL(15,2) NOT NULL DEFAULT 0,
    client_count INTEGER NOT NULL DEFAULT 0,
    average_client_age DECIMAL(5,2),
    average_account_size DECIMAL(12,2),
    
    -- Revenue metrics
    trailing_12mo_revenue DECIMAL(12,2),
    revenue_growth_rate DECIMAL(5,2),
    client_retention_rate DECIMAL(5,2),
    
    -- Client concentration risk
    top_10_clients_aum_pct DECIMAL(5,2), -- Should be <50%
    
    -- Practice valuation (typically 2-3x revenue for RIAs)
    estimated_valuation DECIMAL(15,2),
    valuation_multiple DECIMAL(4,2),
    
    -- Succession readiness score (0-100)
    succession_readiness_score INTEGER CHECK (succession_readiness_score BETWEEN 0 AND 100),
    key_person_dependency_score INTEGER CHECK (key_person_dependency_score BETWEEN 0 AND 100),
    
    -- Documentation completeness
    has_client_service_manual BOOLEAN DEFAULT FALSE,
    has_investment_philosophy_doc BOOLEAN DEFAULT FALSE,
    crm_hygiene_score DECIMAL(3,2) CHECK (crm_hygiene_score BETWEEN 0 AND 1),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ===========================
-- SUCCESSION PLANS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS succession_plans (
    plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    departing_advisor_id UUID NOT NULL,
    successor_advisor_id UUID,
    
    plan_type VARCHAR(50) NOT NULL CHECK (plan_type IN (
        'RETIREMENT',
        'INTERNAL_PROMOTION',
        'EXTERNAL_BUYER',
        'EMERGENCY'
    )),
    target_transition_date DATE,
    
    -- Transition structure
    transition_period_months INTEGER,
    revenue_split_structure JSONB, -- [{"month": 1, "departing_pct": 80, "successor_pct": 20}, ...]
    
    -- Client assignment strategy
    clients_to_transition UUID[], -- Array of client IDs
    transition_complete BOOLEAN DEFAULT FALSE,
    
    -- Financial terms
    purchase_price DECIMAL(12,2),
    payment_terms TEXT,
    earnout_structure JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_succession_plans_departing ON succession_plans(departing_advisor_id);
CREATE INDEX idx_succession_plans_successor ON succession_plans(successor_advisor_id);

-- ===========================
-- CLIENT TRANSITIONS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS client_transitions (
    transition_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    from_advisor_id UUID NOT NULL,
    to_advisor_id UUID NOT NULL,
    succession_plan_id UUID REFERENCES succession_plans(plan_id),
    
    transition_status VARCHAR(50) NOT NULL DEFAULT 'PLANNED' CHECK (transition_status IN (
        'PLANNED',
        'ANNOUNCED',
        'IN_PROGRESS',
        'COMPLETE',
        'CANCELLED'
    )),
    
    -- Transition milestones
    announcement_date DATE,
    first_joint_meeting_date DATE,
    handoff_complete_date DATE,
    
    -- Client sentiment tracking
    client_satisfaction_before DECIMAL(3,2) CHECK (client_satisfaction_before BETWEEN 0 AND 1),
    client_satisfaction_after DECIMAL(3,2) CHECK (client_satisfaction_after BETWEEN 0 AND 1),
    client_retained BOOLEAN,
    
    notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_client_transitions_client ON client_transitions(client_id);
CREATE INDEX idx_client_transitions_from_advisor ON client_transitions(from_advisor_id);
CREATE INDEX idx_client_transitions_to_advisor ON client_transitions(to_advisor_id);
CREATE INDEX idx_client_transitions_status ON client_transitions(transition_status);

COMMENT ON TABLE advisor_practice_metrics IS 'Advisor practice valuations and succession readiness metrics';
COMMENT ON TABLE succession_plans IS 'Succession and continuity plans with financial terms';
COMMENT ON TABLE client_transitions IS 'Individual client transition tracking with sentiment analysis';
