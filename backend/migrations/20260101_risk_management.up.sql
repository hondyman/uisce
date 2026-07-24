-- Risk Management Database Schema
-- Migration: Add tables for options strategies, tail risk analysis, and drawdown tracking

-- ==============================================================================
-- OPTIONS OVERLAY STRATEGIES
-- ==============================================================================

CREATE TABLE IF NOT EXISTS options_overlay_strategies (
    strategy_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    family_id UUID NOT NULL,
    strategy_type VARCHAR(50) NOT NULL, -- PROTECTIVE_PUT, COLLAR, COVERED_CALL, IRON_CONDOR
    underlying_position VARCHAR(50) NOT NULL,
    position_value NUMERIC(18,2) NOT NULL,
    protection_level NUMERIC(5,2), -- % downside protection
    cost_of_protection NUMERIC(18,2) NOT NULL,
    max_loss NUMERIC(18,2),
    max_gain NUMERIC(18,2),
    break_even_price NUMERIC(18,2),
    expiration TIMESTAMP WITH TIME ZONE NOT NULL,
    implied_volatility NUMERIC(5,2),
    status VARCHAR(20) DEFAULT 'PROPOSED', -- PROPOSED, ACTIVE, EXPIRED, CLOSED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_options_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_options_portfolio ON options_overlay_strategies(portfolio_id);
CREATE INDEX idx_options_family ON options_overlay_strategies(family_id);
CREATE INDEX idx_options_type ON options_overlay_strategies(strategy_type);
CREATE INDEX idx_options_status ON options_overlay_strategies(status);
CREATE INDEX idx_options_expiration ON options_overlay_strategies(expiration);

-- Option Legs (individual options in a strategy)
CREATE TABLE IF NOT EXISTS option_legs (
    leg_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL,
    leg_type VARCHAR(50) NOT NULL, -- LONG_PUT, SHORT_PUT, LONG_CALL, SHORT_CALL
    strike_price NUMERIC(18,2) NOT NULL,
    quantity INT NOT NULL,
    premium NUMERIC(18,4) NOT NULL, -- Per share
    expiration TIMESTAMP WITH TIME ZONE NOT NULL,
    option_symbol VARCHAR(100),
    delta NUMERIC(6,4),
    gamma NUMERIC(6,4),
    theta NUMERIC(6,4),
    vega NUMERIC(6,4),
    rho NUMERIC(6,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_leg_strategy FOREIGN KEY (strategy_id) 
        REFERENCES options_overlay_strategies(strategy_id) ON DELETE CASCADE
);

CREATE INDEX idx_leg_strategy ON option_legs(strategy_id);
CREATE INDEX idx_leg_type ON option_legs(leg_type);

-- ==============================================================================
-- TAIL RISK ANALYSIS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS tail_risk_analyses (
    analysis_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    family_id UUID NOT NULL,
    analysis_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    portfolio_value NUMERIC(18,2) NOT NULL,
    
    -- Value at Risk metrics
    value_at_risk_95 NUMERIC(18,2), -- 95% confidence VaR
    value_at_risk_99 NUMERIC(18,2), -- 99% confidence VaR
    conditional_var NUMERIC(18,2), -- CVaR/Expected Shortfall
    
    -- Historical metrics
    max_drawdown_historical NUMERIC(5,2),
    tail_risk_exposure NUMERIC(5,2), -- % portfolio at risk
    
    -- Stress test scenarios (JSONB)
    stress_test_scenarios JSONB,
    
    -- Hedge recommendations (JSONB)
    recommended_hedges JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_tail_risk_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_tail_risk_portfolio ON tail_risk_analyses(portfolio_id);
CREATE INDEX idx_tail_risk_family ON tail_risk_analyses(family_id);
CREATE INDEX idx_tail_risk_date ON tail_risk_analyses(analysis_date DESC);

-- ==============================================================================
-- DRAWDOWN ANALYSIS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS drawdown_analyses (
    analysis_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    family_id UUID NOT NULL,
    analysis_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Current drawdown metrics
    current_drawdown NUMERIC(5,2), -- % below peak
    max_drawdown NUMERIC(5,2), -- Worst historical drawdown
    average_drawdown NUMERIC(5,2), -- Average of all drawdowns
    
    -- Duration metrics
    drawdown_duration INT, -- Days in current drawdown
    recovery_time_estimate INT, -- Estimated days to recover
    
    -- Historical drawdown events (JSONB)
    drawdown_events JSONB,
    
    -- Probability estimates
    prob_10pct_drawdown_1yr NUMERIC(5,2),
    prob_20pct_drawdown_1yr NUMERIC(5,2),
    prob_30pct_drawdown_1yr NUMERIC(5,2),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_drawdown_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_drawdown_portfolio ON drawdown_analyses(portfolio_id);
CREATE INDEX idx_drawdown_family ON drawdown_analyses(family_id);
CREATE INDEX idx_drawdown_date ON drawdown_analyses(analysis_date DESC);

-- ==============================================================================
-- RISK ALERTS
-- ==============================================================================

CREATE TABLE IF NOT EXISTS risk_alerts (
    alert_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    family_id UUID NOT NULL,
    alert_type VARCHAR(50) NOT NULL, -- VaR_BREACH, DRAWDOWN_THRESHOLD, VOLATILITY_SPIKE
    severity VARCHAR(20) NOT NULL, -- LOW, MEDIUM, HIGH, CRITICAL
    alert_message TEXT NOT NULL,
    metric_value NUMERIC(18,2),
    threshold_value NUMERIC(18,2),
    recommended_action TEXT,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_alert_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_alert_portfolio ON risk_alerts(portfolio_id);
CREATE INDEX idx_alert_family ON risk_alerts(family_id);
CREATE INDEX idx_alert_type ON risk_alerts(alert_type);
CREATE INDEX idx_alert_severity ON risk_alerts(severity);
CREATE INDEX idx_alert_acknowledged ON risk_alerts(acknowledged, created_at DESC);

-- ==============================================================================
-- COMMENTS
-- ==============================================================================

COMMENT ON TABLE options_overlay_strategies IS 'Options overlay strategies for portfolio protection';
COMMENT ON TABLE option_legs IS 'Individual option positions within a strategy';
COMMENT ON TABLE tail_risk_analyses IS 'VaR, CVaR, and stress testing analyses';
COMMENT ON TABLE drawdown_analyses IS 'Portfolio drawdown tracking and recovery estimates';
COMMENT ON TABLE risk_alerts IS 'Automated risk threshold alerts';
