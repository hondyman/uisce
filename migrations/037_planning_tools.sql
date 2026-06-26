-- Migration 037: Interactive Planning Tools
-- Goal simulators, scenario analysis, and cash flow monitoring

-- =============================================================================
-- 1. FINANCIAL GOALS (Enhanced from basic version)
-- =============================================================================

CREATE TABLE IF NOT EXISTS financial_goals (
    goal_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Goal details
    goal_name TEXT NOT NULL,
    goal_type VARCHAR(50) NOT NULL, -- 'RETIREMENT', 'EDUCATION', 'HOME', 'WEALTH_BUILDING', 'OTHER'
    
    -- Financial targets
    target_amount DECIMAL(15,2) NOT NULL,
    current_amount DECIMAL(15,2) DEFAULT 0,
    target_date DATE NOT NULL,
    
    -- Funding plan
    monthly_contribution DECIMAL(10,2) DEFAULT 0,
    expected_return_rate DECIMAL(5,2) DEFAULT 7.00, -- Percentage
    
    -- Progress
    progress_percentage DECIMAL(5,2) DEFAULT 0,
    on_track BOOLEAN DEFAULT TRUE,
    projected_shortfall DECIMAL(15,2),
    
    -- Priority
    priority_rank INTEGER DEFAULT 1,
    
    -- Status
    goal_status VARCHAR(20) DEFAULT 'ACTIVE', -- 'ACTIVE', 'COMPLETED', 'ABANDONED'
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_goals_client (client_id, goal_status)
);

-- =============================================================================
-- 2. GOAL SCENARIOS (What-If Analysis)
-- =============================================================================

CREATE TABLE goal_scenarios (
    scenario_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES financial_goals(goal_id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    scenario_name TEXT NOT NULL,
    scenario_type VARCHAR(50), -- 'OPTIMISTIC', 'PESSIMISTIC', 'MARKET_CRASH', 'JOB_LOSS', 'CUSTOM'
    
    -- Assumptions
    assumptions JSONB NOT NULL,
    /* Example:
    {
        "return_rate": 0.05,
        "volatility": 0.20,
        "monthly_contribution": 3000,
        "time_horizon_years": 25,
        "inflation_rate": 0.03,
        "special_events": [
            {"year": 5, "type": "WITHDRAWAL", "amount": -50000, "reason": "Down payment"},
            {"year": 10, "type": "CONTRIBUTION_CHANGE", "new_amount": 5000, "reason": "Promotion"}
        ]
    }
    */
    
    -- Projection results (cached from Monte Carlo simulation)
    projection_data JSONB,
    /* Example:
    {
        "final_value_50th_percentile": 1500000,
        "final_value_25th_percentile": 1200000,
        "final_value_75th_percentile": 1800000,
        "success_probability": 0.75,
        "years_to_goal": [
            {"year": 1, "median": 50000, "p25": 45000, "p75": 55000},
            ...
        ]
    }
    */
    
    success_probability DECIMAL(5,2),
    expected_final_value DECIMAL(15,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_calculated_at TIMESTAMPTZ,
    
    INDEX idx_scenarios_goal (goal_id)
);

-- =============================================================================
-- 3. CASH FLOW TRACKING
-- =============================================================================

CREATE TABLE cash_flow_tracking (
    tracking_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Month being tracked
    tracking_month DATE NOT NULL, -- First day of month
    
    -- Income
    total_income DECIMAL(12,2) DEFAULT 0,
    salary_income DECIMAL(12,2) DEFAULT 0,
    investment_income DECIMAL(12,2) DEFAULT 0,
    other_income DECIMAL(12,2) DEFAULT 0,
    
    -- Expenses
    total_expenses DECIMAL(12,2) DEFAULT 0,
    fixed_expenses DECIMAL(12,2) DEFAULT 0,
    variable_expenses DECIMAL(12,2) DEFAULT 0,
    discretionary_expenses DECIMAL(12,2) DEFAULT 0,
    
    -- Tax withholding
    federal_tax_withheld DECIMAL(12,2) DEFAULT 0,
    state_tax_withheld DECIMAL(12,2) DEFAULT 0,
    estimated_tax_liability DECIMAL(12,2) DEFAULT 0,
    
    -- Net cash flow
    net_cash_flow DECIMAL(12,2) GENERATED ALWAYS AS (total_income - total_expenses) STORED,
    
    -- Savings rate
    savings_rate_pct DECIMAL(5,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (client_id, tracking_month),
    INDEX idx_cashflow_client_month (client_id, tracking_month DESC)
);

-- =============================================================================
-- 4. PLANNING MILESTONES
-- =============================================================================

CREATE TABLE planning_milestones (
    milestone_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES financial_goals(goal_id) ON DELETE CASCADE,
    
    milestone_name TEXT NOT NULL,
    milestone_date DATE NOT NULL,
    target_amount DECIMAL(15,2) NOT NULL,
    
    -- Status
    achieved BOOLEAN DEFAULT FALSE,
    achieved_date DATE,
    actual_amount DECIMAL(15,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_milestones_goal (goal_id, milestone_date)
);

-- =============================================================================
-- 5. HELPER FUNCTIONS
-- =============================================================================

-- Calculate goal progress
CREATE OR REPLACE FUNCTION calculate_goal_progress(p_goal_id UUID)
RETURNS VOID AS $$
DECLARE
    v_goal RECORD;
    v_current_value DECIMAL;
    v_projected_value DECIMAL;
    v_shortfall DECIMAL;
BEGIN
    SELECT * INTO v_goal FROM financial_goals WHERE goal_id = p_goal_id;
    
    IF NOT FOUND THEN
        RETURN;
    END IF;
    
    -- Simple future value calculation (can be replaced with Monte Carlo)
    -- FV = PV * (1 + r)^n + PMT * [((1 + r)^n - 1) / r]
    DECLARE
        v_years_remaining DECIMAL;
        v_monthly_rate DECIMAL;
        v_months_remaining INTEGER;
    BEGIN
        v_years_remaining := EXTRACT(YEAR FROM AGE(v_goal.target_date, CURRENT_DATE));
        v_monthly_rate := v_goal.expected_return_rate / 100 / 12;
        v_months_remaining := v_years_remaining * 12;
        
        -- Future value formula
        v_projected_value := 
            v_goal.current_amount * POWER(1 + v_monthly_rate, v_months_remaining) +
            v_goal.monthly_contribution * ((POWER(1 + v_monthly_rate, v_months_remaining) - 1) / v_monthly_rate);
        
        -- Calculate shortfall
        v_shortfall := v_goal.target_amount - v_projected_value;
        
        -- Update goal
        UPDATE financial_goals
        SET progress_percentage = LEAST(100, (v_projected_value / NULLIF(v_goal.target_amount, 0)) * 100),
            on_track = (v_projected_value >= v_goal.target_amount),
            projected_shortfall = CASE WHEN v_shortfall > 0 THEN v_shortfall ELSE 0 END,
            updated_at = NOW()
        WHERE goal_id = p_goal_id;
    END;
END;
$$ LANGUAGE plpgsql;

-- Run Monte Carlo simulation for scenario
CREATE OR REPLACE FUNCTION run_monte_carlo_simulation(p_scenario_id UUID)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    -- Placeholder: In production, this would call Python service for Monte Carlo
    -- For now, return simple projection
    v_result := '{
        "final_value_50th_percentile": 1500000,
        "final_value_25th_percentile": 1200000,
        "final_value_75th_percentile": 1800000,
        "success_probability": 0.75
    }'::JSONB;
    
    UPDATE goal_scenarios
    SET projection_data = v_result,
        last_calculated_at = NOW()
    WHERE scenario_id = p_scenario_id;
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 6. TRIGGERS
-- =============================================================================

CREATE OR REPLACE FUNCTION update_goal_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER goal_update_trigger
BEFORE UPDATE ON financial_goals
FOR EACH ROW
EXECUTE FUNCTION update_goal_timestamp();

-- =============================================================================
-- 7. RLS POLICIES
-- =============================================================================

ALTER TABLE financial_goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE goal_scenarios ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_flow_tracking ENABLE ROW LEVEL SECURITY;

CREATE POLICY goals_tenant_isolation ON financial_goals
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY cashflow_tenant_isolation ON cash_flow_tracking
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 8. COMMENTS
-- =============================================================================

COMMENT ON TABLE financial_goals IS 'Client financial goals with progress tracking and projections';
COMMENT ON TABLE goal_scenarios IS 'What-if scenario analysis with Monte Carlo simulations';
COMMENT ON TABLE cash_flow_tracking IS 'Monthly income and expense tracking for cash flow analysis';
COMMENT ON FUNCTION calculate_goal_progress IS 'Calculate goal progress and projected shortfall using future value formula';
COMMENT ON FUNCTION run_monte_carlo_simulation IS 'Run Monte Carlo simulation for scenario analysis (placeholder for Python service)';
