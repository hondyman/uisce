-- Portfolio Management System - PostgreSQL Schema
-- Run migrations in order

-- ============================================================================
-- 1. Enable Extensions
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";


-- ============================================================================
-- 2. Users & Authentication
-- ============================================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user', -- user, advisor, admin
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);


-- ============================================================================
-- 3. Portfolio Management
-- ============================================================================
CREATE TABLE portfolios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    total_value DECIMAL(15,2) NOT NULL,
    target_allocation JSONB DEFAULT '{"stocks": 0.6, "bonds": 0.3, "cash": 0.1}',
    current_allocation JSONB,
    drift_threshold DECIMAL(5,4) DEFAULT 0.05,
    last_rebalance TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active', -- active, archived
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_portfolios_user_id ON portfolios(user_id);
CREATE INDEX idx_portfolios_status ON portfolios(status);


-- ============================================================================
-- 4. Holdings & Positions
-- ============================================================================
CREATE TABLE holdings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    ticker VARCHAR(20) NOT NULL,
    shares DECIMAL(20,8) NOT NULL,
    current_price DECIMAL(15,4) NOT NULL,
    cost_basis DECIMAL(15,4) NOT NULL,
    acquired_at TIMESTAMP NOT NULL,
    tax_lot_id VARCHAR(100),
    allocation_pct DECIMAL(5,4),
    target_pct DECIMAL(5,4),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_holdings_portfolio_id ON holdings(portfolio_id);
CREATE INDEX idx_holdings_ticker ON holdings(ticker);


-- ============================================================================
-- 5. Market Data Cache
-- ============================================================================
CREATE TABLE market_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticker VARCHAR(20) NOT NULL UNIQUE,
    price DECIMAL(15,4) NOT NULL,
    price_change DECIMAL(8,4),
    price_change_pct DECIMAL(8,4),
    volume BIGINT,
    market_cap DECIMAL(18,2),
    pe_ratio DECIMAL(8,2),
    dividend_yield DECIMAL(8,4),
    vix DECIMAL(8,2),
    yield_curve VARCHAR(50), -- NORMAL, FLAT, INVERTED
    market_trend VARCHAR(20), -- BULL, BEAR, SIDEWAYS
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_market_data_ticker ON market_data(ticker);
CREATE INDEX idx_market_data_updated_at ON market_data(updated_at);


-- ============================================================================
-- 6. Recommendations Engine
-- ============================================================================
CREATE TABLE recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- TAX_LOSS_HARVEST, DIVERSIFY, REDUCE_RISK, REBALANCE
    priority VARCHAR(20) NOT NULL, -- HIGH, MEDIUM, LOW
    title VARCHAR(255) NOT NULL,
    description TEXT,
    recommended_actions JSONB NOT NULL, -- Array of {ticker, action, quantity, reason}
    expected_benefit JSONB DEFAULT '{}', -- {tax_savings, performance_gain, risk_reduction, ...}
    risk_score DECIMAL(3,2),
    status VARCHAR(50) DEFAULT 'pending', -- pending, approved, executed, rejected, expired
    rejection_reason TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    executed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_recommendations_portfolio_id ON recommendations(portfolio_id);
CREATE INDEX idx_recommendations_status ON recommendations(status);
CREATE INDEX idx_recommendations_priority ON recommendations(priority);
CREATE INDEX idx_recommendations_type ON recommendations(type);
CREATE INDEX idx_recommendations_expires_at ON recommendations(expires_at);


-- ============================================================================
-- 7. Rebalancing & Execution
-- ============================================================================
CREATE TABLE rebalance_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    order_id VARCHAR(100) UNIQUE NOT NULL,
    recommendation_id UUID REFERENCES recommendations(id),
    status VARCHAR(50) DEFAULT 'pending', -- pending, executing, executed, failed, cancelled
    orders_json JSONB NOT NULL, -- Array of orders: {ticker, action, quantity, target_pct, tax}
    total_tax_savings DECIMAL(15,2) DEFAULT 0,
    execution_time_ms INT,
    workflow_id VARCHAR(100), -- Temporal workflow ID
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    executed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rebalance_orders_portfolio_id ON rebalance_orders(portfolio_id);
CREATE INDEX idx_rebalance_orders_status ON rebalance_orders(status);
CREATE INDEX idx_rebalance_orders_order_id ON rebalance_orders(order_id);


-- ============================================================================
-- 8. Portfolio Metrics & Analytics
-- ============================================================================
CREATE TABLE portfolio_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    beta DECIMAL(8,4),
    sharpe_ratio DECIMAL(8,4),
    max_drawdown DECIMAL(8,4),
    concentration DECIMAL(8,4), -- Herfindahl index
    dividend_yield DECIMAL(8,4),
    effective_tax_rate DECIMAL(8,4),
    unrealized_gains DECIMAL(15,2),
    unrealized_losses DECIMAL(15,2),
    harvestable_losses DECIMAL(15,2),
    calculated_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_portfolio_metrics_portfolio_id ON portfolio_metrics(portfolio_id);
CREATE INDEX idx_portfolio_metrics_calculated_at ON portfolio_metrics(calculated_at);


-- ============================================================================
-- 9. Notifications
-- ============================================================================
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE CASCADE,
    recommendation_id UUID REFERENCES recommendations(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- HIGH_PRIORITY_REC, EXECUTION_COMPLETE, TAX_OPPORTUNITY, REBALANCE_ALERT, MARKET_ALERT
    priority VARCHAR(20) DEFAULT 'normal', -- critical, high, normal, low
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    channels JSONB DEFAULT '["in_app"]', -- Array: in_app, email, sms, push
    metadata JSONB DEFAULT '{}',
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_portfolio_id ON notifications(portfolio_id);
CREATE INDEX idx_notifications_read_at ON notifications(read_at);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);


-- ============================================================================
-- 10. Notification Preferences
-- ============================================================================
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    email_high_priority BOOLEAN DEFAULT true,
    email_recommendations BOOLEAN DEFAULT true,
    email_execution_summary BOOLEAN DEFAULT true,
    email_daily_digest BOOLEAN DEFAULT false,
    sms_critical_alerts BOOLEAN DEFAULT true,
    push_notifications BOOLEAN DEFAULT true,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);


-- ============================================================================
-- 11. Notification Delivery Log
-- ============================================================================
CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL, -- email, sms, push, in_app
    recipient VARCHAR(255), -- email, phone, device_id, user_id
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, failed, bounced
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    error_message TEXT,
    sent_at TIMESTAMP,
    failed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notification_deliveries_notification_id ON notification_deliveries(notification_id);
CREATE INDEX idx_notification_deliveries_channel ON notification_deliveries(channel);
CREATE INDEX idx_notification_deliveries_status ON notification_deliveries(status);


-- ============================================================================
-- 12. Audit & Compliance Logging
-- ============================================================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    portfolio_id UUID REFERENCES portfolios(id),
    action VARCHAR(100) NOT NULL, -- CREATE, UPDATE, EXECUTE, REJECT, etc
    entity_type VARCHAR(50), -- portfolio, recommendation, order, etc
    entity_id VARCHAR(100),
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_portfolio_id ON audit_logs(portfolio_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);


-- ============================================================================
-- 13. Views for Common Queries (Used by Hasura)
-- ============================================================================

-- Portfolio summary with current metrics
CREATE VIEW portfolio_summary AS
SELECT 
    p.id,
    p.user_id,
    p.name,
    p.total_value,
    p.target_allocation,
    p.current_allocation,
    p.last_rebalance,
    COUNT(DISTINCT h.id) as holding_count,
    COUNT(DISTINCT r.id) as recommendation_count,
    COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'pending') as pending_recommendations,
    COUNT(DISTINCT ro.id) FILTER (WHERE ro.status = 'executed') as total_rebalances,
    COALESCE(SUM(pm.unrealized_gains), 0) as total_unrealized_gains,
    COALESCE(SUM(pm.harvestable_losses), 0) as total_harvestable_losses,
    p.created_at,
    p.updated_at
FROM portfolios p
LEFT JOIN holdings h ON p.id = h.portfolio_id
LEFT JOIN recommendations r ON p.id = r.portfolio_id
LEFT JOIN rebalance_orders ro ON p.id = ro.portfolio_id
LEFT JOIN portfolio_metrics pm ON p.id = pm.portfolio_id
GROUP BY p.id;

-- Unread notifications count
CREATE VIEW user_notification_summary AS
SELECT 
    u.id as user_id,
    COUNT(DISTINCT n.id) as total_notifications,
    COUNT(DISTINCT n.id) FILTER (WHERE n.read_at IS NULL) as unread_count,
    COUNT(DISTINCT n.id) FILTER (WHERE n.priority = 'critical') as critical_count,
    COUNT(DISTINCT n.id) FILTER (WHERE n.created_at > NOW() - INTERVAL '24 hours') as recent_24h
FROM users u
LEFT JOIN notifications n ON u.id = n.user_id
GROUP BY u.id;

-- Recommendation execution rate
CREATE VIEW recommendation_execution_rate AS
SELECT 
    p.id as portfolio_id,
    p.user_id,
    COUNT(*) as total_recommendations,
    COUNT(*) FILTER (WHERE r.status = 'executed') as executed_count,
    COUNT(*) FILTER (WHERE r.status = 'rejected') as rejected_count,
    ROUND(100.0 * COUNT(*) FILTER (WHERE r.status = 'executed') / NULLIF(COUNT(*), 0), 2) as execution_rate_pct,
    COALESCE(SUM((r.expected_benefit->>'tax_savings')::DECIMAL), 0) as total_tax_savings
FROM portfolios p
LEFT JOIN recommendations r ON p.id = r.portfolio_id
GROUP BY p.id, p.user_id;


-- ============================================================================
-- 14. Functions for Business Logic
-- ============================================================================

-- Function to calculate portfolio drift
CREATE OR REPLACE FUNCTION calculate_portfolio_drift(portfolio_id UUID)
RETURNS TABLE(asset_class VARCHAR, current_pct DECIMAL, target_pct DECIMAL, drift DECIMAL) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        key as asset_class,
        ((value::TEXT)::DECIMAL / 100) as current_pct,
        ((p.target_allocation->key)::TEXT::DECIMAL / 100) as target_pct,
        (((value::TEXT)::DECIMAL - (p.target_allocation->key)::TEXT::DECIMAL) / 100) as drift
    FROM jsonb_each(p.current_allocation)
    CROSS JOIN (SELECT * FROM portfolios WHERE id = portfolio_id) p;
END;
$$ LANGUAGE plpgsql;

-- Function to mark old recommendations as expired
CREATE OR REPLACE FUNCTION expire_old_recommendations()
RETURNS INT AS $$
DECLARE
    count INT;
BEGIN
    UPDATE recommendations
    SET status = 'expired'
    WHERE status = 'pending' 
    AND expires_at < NOW()
    AND status != 'expired';
    
    GET DIAGNOSTICS count = ROW_COUNT;
    RETURN count;
END;
$$ LANGUAGE plpgsql;

-- Function to create audit log
CREATE OR REPLACE FUNCTION create_audit_log(
    p_user_id UUID,
    p_portfolio_id UUID,
    p_action VARCHAR,
    p_entity_type VARCHAR,
    p_entity_id VARCHAR,
    p_old_values JSONB,
    p_new_values JSONB
)
RETURNS UUID AS $$
DECLARE
    log_id UUID;
BEGIN
    INSERT INTO audit_logs (
        user_id, portfolio_id, action, entity_type, entity_id,
        old_values, new_values, ip_address, user_agent
    ) VALUES (
        p_user_id, p_portfolio_id, p_action, p_entity_type, p_entity_id,
        p_old_values, p_new_values, 
        inet_client_addr(), current_setting('application_name')
    )
    RETURNING id INTO log_id;
    
    RETURN log_id;
END;
$$ LANGUAGE plpgsql;


-- ============================================================================
-- 15. Triggers for Automatic Updates
-- ============================================================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_timestamp BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_portfolios_timestamp BEFORE UPDATE ON portfolios
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_holdings_timestamp BEFORE UPDATE ON holdings
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_recommendations_timestamp BEFORE UPDATE ON recommendations
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_rebalance_orders_timestamp BEFORE UPDATE ON rebalance_orders
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_notifications_timestamp BEFORE UPDATE ON notifications
FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 16. Sample Data (For Testing)
-- ============================================================================

INSERT INTO users (email, password_hash, full_name, role, verified_at) 
VALUES 
    ('user@example.com', crypt('password123', gen_salt('bf')), 'John Investor', 'user', NOW()),
    ('advisor@example.com', crypt('advisor123', gen_salt('bf')), 'Jane Advisor', 'advisor', NOW());

INSERT INTO portfolios (user_id, name, description, total_value, current_allocation, last_rebalance)
SELECT id, 'Main Portfolio', 'Diversified long-term investment portfolio', 1000000.00,
       '{"stocks": 0.65, "bonds": 0.28, "cash": 0.07}'::jsonb, NOW()
FROM users WHERE email = 'user@example.com';

INSERT INTO notification_preferences (user_id, email_high_priority, sms_critical_alerts)
SELECT id, true, true FROM users WHERE email = 'user@example.com';


-- ============================================================================
-- 17. Backtest Engine Tables
-- ============================================================================

-- Backtest Results Table
CREATE TABLE backtest_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recommendation_id UUID NOT NULL REFERENCES recommendations(id) ON DELETE CASCADE,
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    simulation_type VARCHAR(50) NOT NULL, -- HISTORICAL, FORWARD, MONTE_CARLO
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    
    -- Return Metrics
    baseline_return DECIMAL(8,6) NOT NULL,
    recommendation_return DECIMAL(8,6) NOT NULL,
    alpha_generated DECIMAL(8,6),
    beta_adjusted_return DECIMAL(8,6),
    
    -- Risk-Adjusted Metrics
    sharpe_ratio_baseline DECIMAL(8,4),
    sharpe_ratio_recommended DECIMAL(8,4),
    max_drawdown_baseline DECIMAL(8,6),
    max_drawdown_recommended DECIMAL(8,6),
    
    -- Cost & Benefit Analysis
    tax_savings_accumulated DECIMAL(15,2),
    transaction_costs DECIMAL(15,2),
    net_benefit DECIMAL(15,2),
    
    -- Model Quality
    confidence DECIMAL(3,2),
    
    -- Daily simulation data (JSON array)
    simulation_data JSONB,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_backtest_results_recommendation_id ON backtest_results(recommendation_id);
CREATE INDEX idx_backtest_results_portfolio_id ON backtest_results(portfolio_id);
CREATE INDEX idx_backtest_results_start_date ON backtest_results(start_date);
CREATE INDEX idx_backtest_results_net_benefit ON backtest_results(net_benefit DESC);


-- Historical prices cache (for faster backtests)
CREATE TABLE historical_prices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticker VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    open_price DECIMAL(15,4),
    high_price DECIMAL(15,4),
    low_price DECIMAL(15,4),
    close_price DECIMAL(15,4),
    volume BIGINT,
    adjusted_close DECIMAL(15,4),
    fetched_at TIMESTAMP DEFAULT NOW(),
    source VARCHAR(50)
);

CREATE UNIQUE INDEX idx_historical_prices_ticker_date ON historical_prices(ticker, date);
CREATE INDEX idx_historical_prices_date ON historical_prices(date DESC);


-- Monte Carlo simulation results
CREATE TABLE monte_carlo_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    backtest_result_id UUID NOT NULL REFERENCES backtest_results(id) ON DELETE CASCADE,
    simulation_day INT NOT NULL,
    path_id INT NOT NULL,
    portfolio_value DECIMAL(15,2),
    daily_return DECIMAL(8,6),
    cumulative_return DECIMAL(8,6),
    max_drawdown_to_date DECIMAL(8,6),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_monte_carlo_backtest_id ON monte_carlo_results(backtest_result_id);
CREATE INDEX idx_monte_carlo_path_id ON monte_carlo_results(path_id);


-- Backtest comparison results
CREATE TABLE backtest_comparisons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    recommendation_id_1 UUID REFERENCES recommendations(id),
    recommendation_id_2 UUID REFERENCES recommendations(id),
    backtest_id_1 UUID NOT NULL REFERENCES backtest_results(id),
    backtest_id_2 UUID NOT NULL REFERENCES backtest_results(id),
    winner VARCHAR(50),
    winner_confidence DECIMAL(3,2),
    performance_diff DECIMAL(8,6),
    risk_adjusted_diff DECIMAL(8,6),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_backtest_comparisons_portfolio_id ON backtest_comparisons(portfolio_id);


-- ============================================================================
-- 18. Backtest Analytics Views
-- ============================================================================

-- Best performing recommendations
CREATE VIEW best_recommendations_by_backtest AS
SELECT 
    r.id as recommendation_id,
    r.type,
    r.priority,
    p.id as portfolio_id,
    COUNT(*) as backtest_count,
    AVG(br.alpha_generated) as avg_alpha,
    AVG(br.sharpe_ratio_recommended) as avg_sharpe,
    AVG(br.net_benefit) as avg_net_benefit,
    MAX(br.net_benefit) as max_net_benefit,
    COUNT(CASE WHEN br.net_benefit > 0 THEN 1 END) as positive_outcomes,
    ROUND(100.0 * COUNT(CASE WHEN br.net_benefit > 0 THEN 1 END) / COUNT(*), 2) as success_rate
FROM backtest_results br
JOIN recommendations r ON br.recommendation_id = r.id
JOIN portfolios p ON br.portfolio_id = p.id
GROUP BY r.id, r.type, r.priority, p.id
ORDER BY avg_net_benefit DESC;


-- Backtest performance summary by user
CREATE VIEW user_backtest_summary AS
SELECT 
    u.id as user_id,
    u.email,
    COUNT(DISTINCT br.id) as total_backtests,
    COUNT(DISTINCT br.portfolio_id) as portfolios_tested,
    AVG(br.alpha_generated) as avg_alpha_generated,
    SUM(br.net_benefit) as total_net_benefit,
    AVG(br.sharpe_ratio_recommended - br.sharpe_ratio_baseline) as avg_sharpe_improvement,
    COUNT(CASE WHEN br.net_benefit > 0 THEN 1 END) as successful_backtests,
    MAX(br.created_at) as last_backtest_date
FROM backtest_results br
JOIN recommendations r ON br.recommendation_id = r.id
JOIN portfolios p ON br.portfolio_id = p.id
JOIN users u ON p.user_id = u.id
GROUP BY u.id, u.email;


-- Recommendation ranking by historical performance
CREATE VIEW recommendation_performance_ranking AS
SELECT 
    r.id,
    r.type,
    r.title,
    COUNT(DISTINCT br.portfolio_id) as applied_to_portfolios,
    AVG(br.alpha_generated) as avg_alpha,
    STDDEV_POP(br.alpha_generated) as alpha_stddev,
    AVG(br.sharpe_ratio_recommended) as avg_sharpe,
    AVG(br.net_benefit) as avg_benefit,
    MIN(br.net_benefit) as worst_case,
    MAX(br.net_benefit) as best_case,
    COUNT(CASE WHEN br.net_benefit > 0 THEN 1 END)::FLOAT / COUNT(*) as success_rate,
    AVG(br.confidence) as model_confidence,
    RANK() OVER (ORDER BY AVG(br.net_benefit) DESC) as performance_rank
FROM backtest_results br
JOIN recommendations r ON br.recommendation_id = r.id
GROUP BY r.id, r.type, r.title;


-- ============================================================================
-- 19. Backtest Functions
-- ============================================================================

-- Store backtest result and create notification
CREATE OR REPLACE FUNCTION store_backtest_result(
    p_recommendation_id UUID,
    p_portfolio_id UUID,
    p_simulation_type VARCHAR,
    p_start_date TIMESTAMP,
    p_end_date TIMESTAMP,
    p_baseline_return DECIMAL,
    p_recommendation_return DECIMAL,
    p_alpha_generated DECIMAL,
    p_net_benefit DECIMAL,
    p_confidence DECIMAL,
    p_simulation_data JSONB
)
RETURNS UUID AS $$
DECLARE
    backtest_id UUID;
    user_id UUID;
BEGIN
    INSERT INTO backtest_results (
        recommendation_id, portfolio_id, simulation_type, start_date, end_date,
        baseline_return, recommendation_return, alpha_generated, net_benefit,
        confidence, simulation_data, created_at
    ) VALUES (
        p_recommendation_id, p_portfolio_id, p_simulation_type, p_start_date, p_end_date,
        p_baseline_return, p_recommendation_return, p_alpha_generated, p_net_benefit,
        p_confidence, p_simulation_data, NOW()
    )
    RETURNING id INTO backtest_id;

    SELECT p.user_id INTO user_id FROM portfolios p WHERE p.id = p_portfolio_id;

    IF p_net_benefit > 1000 THEN
        INSERT INTO notifications (
            user_id, portfolio_id, recommendation_id, type, priority, subject, message, channels
        ) VALUES (
            user_id, p_portfolio_id, p_recommendation_id, 'BACKTEST_COMPLETE', 'high',
            'Backtest Complete: Strong Results',
            'Backtest shows ' || ROUND(p_alpha_generated * 100, 2) || '% alpha generation with $' || ROUND(p_net_benefit, 0) || ' net benefit',
            '["in_app", "email"]'::jsonb
        );
    END IF;

    RETURN backtest_id;
END;
$$ LANGUAGE plpgsql;


-- Calculate recommendation win rate
CREATE OR REPLACE FUNCTION get_recommendation_win_rate(rec_id UUID)
RETURNS TABLE(
    total_backtests INT,
    winning_backtests INT,
    win_rate DECIMAL,
    avg_alpha DECIMAL,
    avg_net_benefit DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INT as total_backtests,
        COUNT(CASE WHEN net_benefit > 0 THEN 1 END)::INT as winning_backtests,
        ROUND(100.0 * COUNT(CASE WHEN net_benefit > 0 THEN 1 END) / COUNT(*), 2)::DECIMAL as win_rate,
        ROUND(AVG(alpha_generated), 6)::DECIMAL as avg_alpha,
        ROUND(AVG(net_benefit), 2)::DECIMAL as avg_net_benefit
    FROM backtest_results
    WHERE recommendation_id = rec_id;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER update_backtest_results_timestamp BEFORE UPDATE ON backtest_results
FOR EACH ROW EXECUTE FUNCTION update_timestamp();
