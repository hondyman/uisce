-- ============================================================================
-- Portfolio Management Database Schema
-- ============================================================================

-- Portfolios Table
CREATE TABLE IF NOT EXISTS portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    currency VARCHAR(3) DEFAULT 'USD',
    total_value NUMERIC(18,2) DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT portfolios_user_id_fk FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_portfolios_user_id ON portfolios(user_id);
CREATE INDEX idx_portfolios_created_at ON portfolios(created_at);

-- Holdings Table
CREATE TABLE IF NOT EXISTS holdings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(255),
    asset_class VARCHAR(50),
    quantity NUMERIC(18,4) NOT NULL,
    average_cost NUMERIC(18,2) NOT NULL,
    current_price NUMERIC(18,2) NOT NULL,
    current_value NUMERIC(18,2) GENERATED ALWAYS AS (quantity * current_price) STORED,
    sector VARCHAR(100),
    geography VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT holdings_portfolio_id_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
);

CREATE INDEX idx_holdings_portfolio_id ON holdings(portfolio_id);
CREATE INDEX idx_holdings_symbol ON holdings(symbol);

-- Recommendations Table
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    created_by UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50), -- rebalance, tactical, strategic
    status VARCHAR(50) DEFAULT 'draft', -- draft, proposed, accepted, rejected, implemented
    priority VARCHAR(20) DEFAULT 'medium', -- high, medium, low
    target_allocations JSONB,
    recommended_actions JSONB,
    rationale TEXT,
    risk_score NUMERIC(5,2) DEFAULT 0,
    expected_return NUMERIC(6,3) DEFAULT 0,
    time_horizon INTEGER, -- days
    backtest_id UUID,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT recommendations_portfolio_id_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE,
    CONSTRAINT recommendations_created_by_fk FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX idx_recommendations_portfolio_id ON recommendations(portfolio_id);
CREATE INDEX idx_recommendations_status ON recommendations(status);
CREATE INDEX idx_recommendations_created_at ON recommendations(created_at DESC);

-- Backtest Results Table
CREATE TABLE IF NOT EXISTS backtest_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id UUID NOT NULL,
    portfolio_id UUID NOT NULL,
    simulation_type VARCHAR(50), -- historical, monte_carlo, stress_test
    start_date DATE,
    end_date DATE,
    baseline_return NUMERIC(8,5),
    recommendation_return NUMERIC(8,5),
    alpha_generated NUMERIC(8,5),
    beta_adjusted_return NUMERIC(8,5),
    sharpe_ratio_baseline NUMERIC(8,4),
    sharpe_ratio_recommended NUMERIC(8,4),
    max_drawdown_baseline NUMERIC(6,4),
    max_drawdown_recommended NUMERIC(6,4),
    tax_savings_accumulated NUMERIC(15,2),
    transaction_costs NUMERIC(15,2),
    net_benefit NUMERIC(15,2),
    confidence NUMERIC(5,3),
    simulation_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT backtest_results_rec_id_fk FOREIGN KEY (recommendation_id) REFERENCES recommendations(id),
    CONSTRAINT backtest_results_port_id_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id)
);

CREATE INDEX idx_backtest_results_rec_id ON backtest_results(recommendation_id);
CREATE INDEX idx_backtest_results_port_id ON backtest_results(portfolio_id);
CREATE INDEX idx_backtest_results_created_at ON backtest_results(created_at DESC);

-- Historical Prices Table (for backtesting)
CREATE TABLE IF NOT EXISTS historical_prices (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    open_price NUMERIC(18,2),
    high_price NUMERIC(18,2),
    low_price NUMERIC(18,2),
    close_price NUMERIC(18,2) NOT NULL,
    volume BIGINT,
    adjusted_close NUMERIC(18,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(ticker, date)
);

CREATE INDEX idx_historical_prices_ticker_date ON historical_prices(ticker, date DESC);
CREATE INDEX idx_historical_prices_date ON historical_prices(date DESC);

-- Monte Carlo Results Table
CREATE TABLE IF NOT EXISTS monte_carlo_results (
    id BIGSERIAL PRIMARY KEY,
    backtest_id UUID NOT NULL,
    path_id INTEGER NOT NULL,
    day INTEGER,
    portfolio_value NUMERIC(18,2),
    daily_return NUMERIC(8,5),
    cumulative_return NUMERIC(8,5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT monte_carlo_results_backtest_fk FOREIGN KEY (backtest_id) REFERENCES backtest_results(id) ON DELETE CASCADE
);

CREATE INDEX idx_monte_carlo_backtest_id ON monte_carlo_results(backtest_id);
CREATE INDEX idx_monte_carlo_path_id ON monte_carlo_results(path_id);

-- Backtest Comparisons Table
CREATE TABLE IF NOT EXISTS backtest_comparisons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    recommendation_id_1 UUID NOT NULL,
    recommendation_id_2 UUID NOT NULL,
    winner VARCHAR(50), -- rec1, rec2, tie
    performance_diff NUMERIC(8,5),
    risk_diff NUMERIC(6,4),
    sharpe_ratio_diff NUMERIC(8,4),
    drawdown_diff NUMERIC(6,4),
    tax_diff NUMERIC(15,2),
    cost_diff NUMERIC(15,2),
    reasoning TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT comp_portfolio_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id),
    CONSTRAINT comp_rec1_fk FOREIGN KEY (recommendation_id_1) REFERENCES recommendations(id),
    CONSTRAINT comp_rec2_fk FOREIGN KEY (recommendation_id_2) REFERENCES recommendations(id)
);

CREATE INDEX idx_backtest_comp_portfolio ON backtest_comparisons(portfolio_id);
CREATE INDEX idx_backtest_comp_created_at ON backtest_comparisons(created_at DESC);

-- Portfolio Risk Metrics Table
CREATE TABLE IF NOT EXISTS portfolio_risk_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    as_of_date DATE NOT NULL,
    expected_return NUMERIC(6,3),
    volatility NUMERIC(6,3),
    sharpe_ratio NUMERIC(8,4),
    sortino_ratio NUMERIC(8,4),
    beta NUMERIC(8,4),
    alpha NUMERIC(8,5),
    max_drawdown NUMERIC(6,4),
    var_95 NUMERIC(8,5),
    cvar_95 NUMERIC(8,5),
    diversification_ratio NUMERIC(8,4),
    herfindahl_index NUMERIC(8,4),
    top_10_holdings NUMERIC(5,3),
    top_5_holdings NUMERIC(5,3),
    top_1_holding NUMERIC(5,3),
    correlation_matrix JSONB,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT risk_metrics_portfolio_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
);

CREATE INDEX idx_risk_metrics_portfolio ON portfolio_risk_metrics(portfolio_id);
CREATE INDEX idx_risk_metrics_as_of_date ON portfolio_risk_metrics(as_of_date DESC);

-- Risk Factors Table
CREATE TABLE IF NOT EXISTS risk_factors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    factor_name VARCHAR(100),
    exposure NUMERIC(8,4),
    sensitivity NUMERIC(8,4),
    contribution NUMERIC(8,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT risk_factors_portfolio_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
);

CREATE INDEX idx_risk_factors_portfolio ON risk_factors(portfolio_id);

-- Rebalancing Plans Table
CREATE TABLE IF NOT EXISTS rebalancing_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL,
    created_by UUID NOT NULL,
    status VARCHAR(50) DEFAULT 'draft', -- draft, proposed, approved, executed, canceled
    rebalancing_type VARCHAR(50), -- threshold, tactical, strategic, rehedging
    target_deviation_pct NUMERIC(5,3),
    proposed_transactions JSONB,
    estimated_cost NUMERIC(15,2),
    estimated_tax_impact NUMERIC(15,2),
    approved_at TIMESTAMP WITH TIME ZONE,
    executed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT rebalancing_plans_portfolio_fk FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE,
    CONSTRAINT rebalancing_plans_created_by_fk FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX idx_rebalancing_plans_portfolio ON rebalancing_plans(portfolio_id);
CREATE INDEX idx_rebalancing_plans_status ON rebalancing_plans(status);
CREATE INDEX idx_rebalancing_plans_created_at ON rebalancing_plans(created_at DESC);

-- ============================================================================
-- Views for Analytics
-- ============================================================================

-- Best performing recommendations by portfolio
CREATE OR REPLACE VIEW best_recommendations_by_portfolio AS
SELECT
    br.portfolio_id,
    br.recommendation_id,
    br.net_benefit,
    br.sharpe_ratio_recommended,
    br.alpha_generated,
    br.created_at,
    ROW_NUMBER() OVER (PARTITION BY br.portfolio_id ORDER BY br.net_benefit DESC) as rank
FROM backtest_results br
WHERE br.created_at >= NOW() - INTERVAL '90 days';

-- Portfolio performance summary
CREATE OR REPLACE VIEW portfolio_performance_summary AS
SELECT
    p.id as portfolio_id,
    p.name,
    p.user_id,
    p.total_value,
    COUNT(DISTINCT h.id) as holding_count,
    COUNT(DISTINCT r.id) as recommendation_count,
    MAX(CASE WHEN br.net_benefit IS NOT NULL THEN br.net_benefit END) as best_recommendation_benefit,
    AVG(br.sharpe_ratio_recommended) as avg_sharpe_ratio,
    p.created_at
FROM portfolios p
LEFT JOIN holdings h ON p.id = h.portfolio_id
LEFT JOIN recommendations r ON p.id = r.portfolio_id
LEFT JOIN backtest_results br ON r.id = br.recommendation_id
GROUP BY p.id, p.name, p.user_id, p.total_value, p.created_at;

-- Risk metrics trend
CREATE OR REPLACE VIEW risk_metrics_trend AS
SELECT
    portfolio_id,
    as_of_date,
    expected_return,
    volatility,
    sharpe_ratio,
    max_drawdown,
    LAG(expected_return) OVER (PARTITION BY portfolio_id ORDER BY as_of_date) as prev_expected_return,
    LAG(volatility) OVER (PARTITION BY portfolio_id ORDER BY as_of_date) as prev_volatility
FROM portfolio_risk_metrics
ORDER BY portfolio_id, as_of_date DESC;

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function to update portfolio total value
CREATE OR REPLACE FUNCTION update_portfolio_total_value()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE portfolios
    SET total_value = (
        SELECT COALESCE(SUM(current_value), 0)
        FROM holdings
        WHERE portfolio_id = NEW.portfolio_id
    ),
    updated_at = NOW()
    WHERE id = NEW.portfolio_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for holdings update
DROP TRIGGER IF EXISTS trigger_update_portfolio_total_value ON holdings;
CREATE TRIGGER trigger_update_portfolio_total_value
AFTER INSERT OR UPDATE OR DELETE ON holdings
FOR EACH ROW
EXECUTE FUNCTION update_portfolio_total_value();

-- ============================================================================
-- Initial Data Setup
-- ============================================================================

-- You can add sample data here if needed
-- INSERT INTO portfolios (id, user_id, name, description, currency) 
-- VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'Sample Portfolio', 'For testing', 'USD');
