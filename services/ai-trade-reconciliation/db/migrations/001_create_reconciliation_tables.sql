-- 001_create_reconciliation_tables.sql
-- Create all tables for AI Trade Reconciliation module

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Trades table
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    portfolio_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('buy', 'sell')),
    shares NUMERIC NOT NULL,
    price NUMERIC NOT NULL,
    trade_date TIMESTAMPTZ NOT NULL,
    settle_date TIMESTAMPTZ,
    custodian TEXT,
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'discrepancy')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB,
    created_by UUID,
    updated_by UUID
);

CREATE INDEX idx_trades_portfolio_id ON trades(portfolio_id);
CREATE INDEX idx_trades_trade_date ON trades(trade_date);
CREATE INDEX idx_trades_status ON trades(status);
CREATE INDEX idx_trades_symbol ON trades(symbol);

-- Trade confirmations table
CREATE TABLE IF NOT EXISTS trade_confirms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source TEXT NOT NULL CHECK (source IN ('email', 'sftp', 'api', 'manual')),
    raw_data JSONB,
    parsed JSONB,
    received_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_trade_confirms_received_at ON trade_confirms(received_at);
CREATE INDEX idx_trade_confirms_source ON trade_confirms(source);

-- Reconciliation results table
CREATE TABLE IF NOT EXISTS reconciliation_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_date DATE NOT NULL,
    match_rate NUMERIC NOT NULL,
    matched_count INTEGER NOT NULL,
    unmatched_count INTEGER NOT NULL,
    discrepancies JSONB,
    model_version INTEGER,
    status TEXT DEFAULT 'completed' CHECK (status IN ('in_progress', 'completed', 'failed')),
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX idx_reconciliation_results_run_date ON reconciliation_results(run_date);
CREATE INDEX idx_reconciliation_results_status ON reconciliation_results(status);

-- Discrepancies table
CREATE TABLE IF NOT EXISTS discrepancies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    result_id UUID NOT NULL REFERENCES reconciliation_results(id) ON DELETE CASCADE,
    trade_id UUID REFERENCES trades(id) ON DELETE SET NULL,
    confirm_id UUID REFERENCES trade_confirms(id) ON DELETE SET NULL,
    discrepancy_type TEXT NOT NULL CHECK (discrepancy_type IN ('unmatched_trade', 'unmatched_confirm', 'mismatch')),
    field TEXT,
    trade_value JSONB,
    confirm_value JSONB,
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    suggested_fix TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_discrepancies_result_id ON discrepancies(result_id);
CREATE INDEX idx_discrepancies_severity ON discrepancies(severity);
CREATE INDEX idx_discrepancies_type ON discrepancies(discrepancy_type);

-- Reconciliation tasks (for ops team)
CREATE TABLE IF NOT EXISTS reconciliation_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    result_id UUID NOT NULL REFERENCES reconciliation_results(id) ON DELETE CASCADE,
    discrepancy_id UUID REFERENCES discrepancies(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'escalated')),
    assigned_to UUID,
    priority TEXT NOT NULL CHECK (priority IN ('low', 'medium', 'high')),
    notes TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX idx_reconciliation_tasks_result_id ON reconciliation_tasks(result_id);
CREATE INDEX idx_reconciliation_tasks_status ON reconciliation_tasks(status);
CREATE INDEX idx_reconciliation_tasks_assigned_to ON reconciliation_tasks(assigned_to);
CREATE INDEX idx_reconciliation_tasks_priority ON reconciliation_tasks(priority);

-- Reconciliation rules table (low-code tolerance rules)
CREATE TABLE IF NOT EXISTS reconciliation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    rule_type TEXT NOT NULL CHECK (rule_type IN ('share_tolerance', 'price_tolerance', 'date_tolerance', 'custom')),
    enabled BOOLEAN DEFAULT TRUE,
    rule_expr TEXT NOT NULL,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID
);

CREATE INDEX idx_reconciliation_rules_enabled ON reconciliation_rules(enabled);
CREATE INDEX idx_reconciliation_rules_type ON reconciliation_rules(rule_type);

-- Audit log table
CREATE TABLE IF NOT EXISTS reconciliation_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    result_id UUID NOT NULL REFERENCES reconciliation_results(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    actor UUID,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reconciliation_audit_logs_result_id ON reconciliation_audit_logs(result_id);
CREATE INDEX idx_reconciliation_audit_logs_action ON reconciliation_audit_logs(action);
CREATE INDEX idx_reconciliation_audit_logs_created_at ON reconciliation_audit_logs(created_at);

-- Insert default tolerance rules
INSERT INTO reconciliation_rules (name, description, rule_type, rule_expr) VALUES
('share_tolerance', 'Allow ±0.1% difference in shares', 'share_tolerance', '$abs(($trade.shares - $confirm.shares) / $trade.shares) <= 0.001'),
('price_tolerance', 'Allow ±0.5% or $0.01 difference in price', 'price_tolerance', '$max($abs($trade.price - $confirm.price) / $trade.price, 0.005) <= 0.005 or $abs($trade.price - $confirm.price) <= 0.01'),
('date_tolerance', 'Allow ±1 business day difference', 'date_tolerance', '$abs($dateDayOfWeek($trade.trade_date) - $dateDayOfWeek($confirm.received_at)) <= 1')
ON CONFLICT (name) DO NOTHING;
