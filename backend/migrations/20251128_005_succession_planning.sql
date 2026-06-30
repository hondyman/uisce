-- Advisor practice valuation
CREATE TABLE IF NOT EXISTS advisor_practice_metrics (
    advisor_id UUID PRIMARY KEY,
    evaluation_date DATE NOT NULL,
    
    -- Book of business metrics
    total_aum DECIMAL(15,2),
    client_count INTEGER,
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
    succession_readiness_score INTEGER,
    key_person_dependency_score INTEGER, -- Lower is better
    
    -- Documentation completeness
    has_client_service_manual BOOLEAN DEFAULT FALSE,
    has_investment_philosophy_doc BOOLEAN DEFAULT FALSE,
    crm_hygiene_score DECIMAL(3,2), -- 0.0-1.0
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Successor mapping
CREATE TABLE IF NOT EXISTS succession_plans (
    plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    departing_advisor_id UUID NOT NULL, -- REFERENCES users(id)
    successor_advisor_id UUID, -- REFERENCES users(id), nullable if external buyer
    
    plan_type VARCHAR(50) NOT NULL, -- 'RETIREMENT', 'INTERNAL_PROMOTION', 'EXTERNAL_BUYER', 'EMERGENCY'
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
    
    -- Status
    status VARCHAR(50) DEFAULT 'PLANNING', -- 'PLANNING', 'ANNOUNCED', 'IN_PROGRESS', 'COMPLETE', 'CANCELLED'
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_succession_plans_departing ON succession_plans(departing_advisor_id);
CREATE INDEX IF NOT EXISTS idx_succession_plans_successor ON succession_plans(successor_advisor_id);

-- Client transition tracking
CREATE TABLE IF NOT EXISTS client_transitions (
    transition_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    from_advisor_id UUID NOT NULL, -- REFERENCES users(id)
    to_advisor_id UUID NOT NULL, -- REFERENCES users(id)
    succession_plan_id UUID REFERENCES succession_plans(plan_id),
    
    transition_status VARCHAR(50) DEFAULT 'PLANNED', -- 'PLANNED', 'ANNOUNCED', 'IN_PROGRESS', 'COMPLETE', 'DECLINED'
    
    -- Transition milestones
    announcement_date DATE,
    first_joint_meeting_date DATE,
    handoff_complete_date DATE,
    
    -- Client sentiment tracking
    client_satisfaction_before DECIMAL(3,2),
    client_satisfaction_after DECIMAL(3,2),
    client_retained BOOLEAN,
    
    notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transitions_client ON client_transitions(client_id);
CREATE INDEX IF NOT EXISTS idx_transitions_from ON client_transitions(from_advisor_id);
CREATE INDEX IF NOT EXISTS idx_transitions_to ON client_transitions(to_advisor_id);

-- Successor compatibility scores (for ML matching)
CREATE TABLE IF NOT EXISTS successor_compatibility_scores (
    score_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    departing_advisor_id UUID NOT NULL,
    candidate_advisor_id UUID NOT NULL,
    
    -- Compatibility dimensions
    client_demographic_match DECIMAL(3,2), -- 0.0-1.0
    service_style_match DECIMAL(3,2),
    specialization_overlap DECIMAL(3,2),
    capacity_match DECIMAL(3,2),
    geographic_match DECIMAL(3,2),
    
    -- Overall score
    overall_compatibility_score DECIMAL(3,2),
    
    -- Metadata
    reasoning TEXT,
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_successor_scores_departing ON successor_compatibility_scores(departing_advisor_id);
CREATE INDEX IF NOT EXISTS idx_successor_scores_overall ON successor_compatibility_scores(overall_compatibility_score DESC);
