-- Tenant Financial Database Schema (Isolated per Tenant)
-- 
-- Each tenant gets their own PostgreSQL instance for:
--   - Financial data with full isolation
--   - Regulatory compliance (data residency)
--   - Tenant-specific schemas and customizations
--
-- This is a TEMPLATE - deployed per tenant

-- =============================================================================
-- ACCOUNTS
-- =============================================================================

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(128) UNIQUE,
    account_name VARCHAR(256) NOT NULL,
    account_type VARCHAR(64) NOT NULL, -- checking, savings, investment, custody
    account_number VARCHAR(64),
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(32) DEFAULT 'active',
    opened_date DATE,
    closed_date DATE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_accounts_external_id ON accounts(external_id);
CREATE INDEX idx_accounts_type ON accounts(account_type);
CREATE INDEX idx_accounts_status ON accounts(status);

-- =============================================================================
-- PORTFOLIOS
-- =============================================================================

CREATE TABLE IF NOT EXISTS portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(128) UNIQUE,
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    portfolio_name VARCHAR(256) NOT NULL,
    portfolio_type VARCHAR(64), -- managed, self-directed, retirement
    benchmark VARCHAR(64),
    inception_date DATE,
    status VARCHAR(32) DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_portfolios_account ON portfolios(account_id);
CREATE INDEX idx_portfolios_type ON portfolios(portfolio_type);

-- =============================================================================
-- SECURITIES (Reference Data)
-- =============================================================================

CREATE TABLE IF NOT EXISTS securities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(32) NOT NULL,
    cusip VARCHAR(9),
    isin VARCHAR(12),
    sedol VARCHAR(7),
    security_name VARCHAR(256) NOT NULL,
    security_type VARCHAR(64), -- equity, bond, etf, mutual_fund, option, etc.
    exchange VARCHAR(32),
    currency VARCHAR(3) DEFAULT 'USD',
    sector VARCHAR(128),
    industry VARCHAR(128),
    country VARCHAR(64),
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(symbol, security_type)
);

CREATE INDEX idx_securities_symbol ON securities(symbol);
CREATE INDEX idx_securities_cusip ON securities(cusip);
CREATE INDEX idx_securities_isin ON securities(isin);
CREATE INDEX idx_securities_type ON securities(security_type);

-- =============================================================================
-- TRANSACTIONS
-- =============================================================================

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(128) UNIQUE,
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE SET NULL,
    security_id UUID REFERENCES securities(id) ON DELETE SET NULL,
    transaction_date DATE NOT NULL,
    settlement_date DATE,
    transaction_type VARCHAR(32) NOT NULL, -- buy, sell, dividend, fee, transfer, etc.
    quantity DECIMAL(18, 8),
    price DECIMAL(18, 8),
    amount DECIMAL(18, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    fees DECIMAL(18, 4) DEFAULT 0,
    tax_lot_id UUID,
    description TEXT,
    status VARCHAR(32) DEFAULT 'settled',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_account ON transactions(account_id);
CREATE INDEX idx_transactions_portfolio ON transactions(portfolio_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date DESC);
CREATE INDEX idx_transactions_type ON transactions(transaction_type);
CREATE INDEX idx_transactions_security ON transactions(security_id);

-- Partitioning for large tables (optional, enable if > 10M rows)
-- CREATE TABLE transactions_partitioned (LIKE transactions INCLUDING ALL)
-- PARTITION BY RANGE (transaction_date);

-- =============================================================================
-- POSITIONS (Current Holdings)
-- =============================================================================

CREATE TABLE IF NOT EXISTS positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    security_id UUID NOT NULL REFERENCES securities(id) ON DELETE CASCADE,
    quantity DECIMAL(18, 8) NOT NULL,
    average_cost DECIMAL(18, 8),
    cost_basis DECIMAL(18, 4),
    market_value DECIMAL(18, 4),
    unrealized_pnl DECIMAL(18, 4),
    unrealized_pnl_pct DECIMAL(12, 6),
    currency VARCHAR(3) DEFAULT 'USD',
    as_of_date DATE NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(portfolio_id, security_id, as_of_date)
);

CREATE INDEX idx_positions_portfolio ON positions(portfolio_id);
CREATE INDEX idx_positions_security ON positions(security_id);
CREATE INDEX idx_positions_date ON positions(as_of_date DESC);

-- =============================================================================
-- TAX LOTS
-- =============================================================================

CREATE TABLE IF NOT EXISTS tax_lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    security_id UUID NOT NULL REFERENCES securities(id) ON DELETE CASCADE,
    acquisition_date DATE NOT NULL,
    acquisition_price DECIMAL(18, 8) NOT NULL,
    original_quantity DECIMAL(18, 8) NOT NULL,
    remaining_quantity DECIMAL(18, 8) NOT NULL,
    cost_basis DECIMAL(18, 4) NOT NULL,
    holding_period VARCHAR(16), -- short_term, long_term
    lot_method VARCHAR(32) DEFAULT 'fifo', -- fifo, lifo, specific, hifo
    is_closed BOOLEAN DEFAULT false,
    closed_date DATE,
    realized_pnl DECIMAL(18, 4),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tax_lots_portfolio ON tax_lots(portfolio_id);
CREATE INDEX idx_tax_lots_security ON tax_lots(security_id);
CREATE INDEX idx_tax_lots_open ON tax_lots(is_closed, remaining_quantity) WHERE NOT is_closed;

-- =============================================================================
-- PERFORMANCE HISTORY
-- =============================================================================

CREATE TABLE IF NOT EXISTS performance_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    period_date DATE NOT NULL,
    period_type VARCHAR(16) NOT NULL, -- daily, monthly, quarterly, yearly
    beginning_value DECIMAL(18, 4),
    ending_value DECIMAL(18, 4),
    net_flows DECIMAL(18, 4),
    time_weighted_return DECIMAL(12, 8),
    money_weighted_return DECIMAL(12, 8),
    benchmark_return DECIMAL(12, 8),
    alpha DECIMAL(12, 8),
    beta DECIMAL(8, 4),
    sharpe_ratio DECIMAL(8, 4),
    sortino_ratio DECIMAL(8, 4),
    max_drawdown DECIMAL(12, 8),
    volatility DECIMAL(12, 8),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(portfolio_id, period_date, period_type)
);

CREATE INDEX idx_performance_portfolio ON performance_history(portfolio_id);
CREATE INDEX idx_performance_date ON performance_history(period_date DESC);

-- =============================================================================
-- AUDIT TRAIL (Regulatory Compliance)
-- =============================================================================

CREATE TABLE IF NOT EXISTS audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(64) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(16) NOT NULL, -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    changed_by VARCHAR(128),
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_trail_table_record ON audit_trail(table_name, record_id);
CREATE INDEX idx_audit_trail_time ON audit_trail(changed_at DESC);

-- Audit trigger function
CREATE OR REPLACE FUNCTION audit_trigger_func()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_trail (table_name, record_id, action, old_values, changed_by)
        VALUES (TG_TABLE_NAME, OLD.id, 'DELETE', to_jsonb(OLD), current_user);
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_trail (table_name, record_id, action, old_values, new_values, changed_by)
        VALUES (TG_TABLE_NAME, NEW.id, 'UPDATE', to_jsonb(OLD), to_jsonb(NEW), current_user);
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_trail (table_name, record_id, action, new_values, changed_by)
        VALUES (TG_TABLE_NAME, NEW.id, 'INSERT', to_jsonb(NEW), current_user);
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Apply audit triggers to sensitive tables
CREATE TRIGGER audit_accounts AFTER INSERT OR UPDATE OR DELETE ON accounts
FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER audit_transactions AFTER INSERT OR UPDATE OR DELETE ON transactions
FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER audit_positions AFTER INSERT OR UPDATE OR DELETE ON positions
FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER audit_tax_lots AFTER INSERT OR UPDATE OR DELETE ON tax_lots
FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

-- =============================================================================
-- VIEWS FOR COMMON QUERIES
-- =============================================================================

-- Current positions with security details
CREATE OR REPLACE VIEW v_current_positions AS
SELECT 
    p.id AS position_id,
    pf.id AS portfolio_id,
    pf.portfolio_name,
    a.id AS account_id,
    a.account_name,
    s.symbol,
    s.security_name,
    s.security_type,
    s.sector,
    p.quantity,
    p.average_cost,
    p.cost_basis,
    p.market_value,
    p.unrealized_pnl,
    p.unrealized_pnl_pct,
    p.currency,
    p.as_of_date
FROM positions p
JOIN portfolios pf ON p.portfolio_id = pf.id
JOIN accounts a ON pf.account_id = a.id
JOIN securities s ON p.security_id = s.id
WHERE p.as_of_date = (SELECT MAX(as_of_date) FROM positions WHERE portfolio_id = p.portfolio_id);

-- Portfolio summary
CREATE OR REPLACE VIEW v_portfolio_summary AS
SELECT 
    pf.id AS portfolio_id,
    pf.portfolio_name,
    a.account_name,
    COUNT(DISTINCT p.security_id) AS position_count,
    SUM(p.market_value) AS total_market_value,
    SUM(p.cost_basis) AS total_cost_basis,
    SUM(p.unrealized_pnl) AS total_unrealized_pnl,
    CASE 
        WHEN SUM(p.cost_basis) > 0 
        THEN SUM(p.unrealized_pnl) / SUM(p.cost_basis) * 100
        ELSE 0 
    END AS unrealized_pnl_pct,
    MAX(p.as_of_date) AS as_of_date
FROM portfolios pf
JOIN accounts a ON pf.account_id = a.id
LEFT JOIN positions p ON pf.id = p.portfolio_id
GROUP BY pf.id, pf.portfolio_name, a.account_name;
