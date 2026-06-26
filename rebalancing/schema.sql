CREATE TABLE portfolios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  aum DECIMAL(20,2) NOT NULL,
  drift DECIMAL(5,2) DEFAULT 0,
  risk_score DECIMAL(5,2) DEFAULT 0,
  alpha DECIMAL(5,2) DEFAULT 0,
  sector_attribution JSONB,
  mitigation_action TEXT,
  last_rebalance TIMESTAMP,
  tax_saved DECIMAL(20,2) DEFAULT 0,
  rebalance_status VARCHAR(50) DEFAULT 'idle',
  target_model JSONB NOT NULL,
  constraints JSONB NOT NULL,
  policy_document TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_portfolios_tenant ON portfolios(tenant_id);
CREATE INDEX idx_portfolios_drift ON portfolios(drift) WHERE drift > 5;
CREATE INDEX idx_portfolios_status ON portfolios(rebalance_status) WHERE rebalance_status != 'idle';

-- Holdings
CREATE TABLE holdings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
  symbol VARCHAR(20) NOT NULL,
  shares DECIMAL(20,8) NOT NULL CHECK (shares > 0),
  current_price DECIMAL(20,4) NOT NULL CHECK (current_price > 0),
  cost_basis DECIMAL(20,4) NOT NULL,
  purchase_date DATE NOT NULL,
  tax_lot_id VARCHAR(100) NOT NULL,
  sector VARCHAR(100) NOT NULL,
  asset_class VARCHAR(50),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(portfolio_id, tax_lot_id)
);

CREATE INDEX idx_holdings_portfolio ON holdings(portfolio_id);
CREATE INDEX idx_holdings_symbol ON holdings(symbol);
CREATE INDEX idx_holdings_sector ON holdings(sector);

-- Rebalance Plans
CREATE TABLE rebalance_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  portfolio_id UUID NOT NULL REFERENCES portfolios(id),
  timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
  current_drift DECIMAL(5,2) NOT NULL,
  expected_drift DECIMAL(5,2) NOT NULL,
  tax_savings DECIMAL(20,2) NOT NULL,
  confidence DECIMAL(5,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 100),
  status VARCHAR(50) NOT NULL DEFAULT 'proposed',
  rationale TEXT,
  summary TEXT,
  proposed_trades JSONB NOT NULL,
  tax_analysis JSONB NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_plans_portfolio ON rebalance_plans(portfolio_id);
CREATE INDEX idx_plans_timestamp ON rebalance_plans(timestamp DESC);
CREATE INDEX idx_plans_status ON rebalance_plans(status);

-- Audit Logs
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id VARCHAR(100) NOT NULL,
  tenant_id UUID NOT NULL,
  action VARCHAR(100) NOT NULL,
  resource VARCHAR(100) NOT NULL,
  resource_id VARCHAR(100) NOT NULL,
  allowed BOOLEAN NOT NULL,
  timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_user ON audit_logs(user_id);

-- Sample Data
INSERT INTO portfolios (id, tenant_id, name, aum, drift, risk_score, alpha, target_model, constraints, policy_document) VALUES
('11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001', 'Growth Portfolio', 10000000.00, 8.5, 7.5, 1.2,
  '{"Technology": 40, "Healthcare": 20, "Financial": 15, "Consumer": 15, "Other": 10}',
  '{"max_trade_size": 500000, "min_trade_size": 1000, "max_turnover": 10, "tax_budget": 100000, "drift_tolerance": 5, "restricted_list": [], "esg_preference": "high", "risk_appetite": "aggressive", "forbidden_sectors": ["Tobacco"]}',
  '1. The portfolio must not hold any securities from the Tobacco or Firearms sectors.\n2. The allocation to any single stock must not exceed 15% of the total portfolio value.\n3. Maintain a minimum cash reserve of 2%.'
);

INSERT INTO holdings (portfolio_id, symbol, shares, current_price, cost_basis, purchase_date, tax_lot_id, sector) VALUES
('11111111-1111-1111-1111-111111111111', 'AAPL', 5000, 178.50, 150.00, '2023-01-15', 'lot_aapl_1', 'Technology'),
('11111111-1111-1111-1111-111111111111', 'MSFT', 3000, 420.30, 380.00, '2023-03-20', 'lot_msft_1', 'Technology'),
('11111111-1111-1111-1111-111111111111', 'GOOGL', 2000, 142.80, 125.50, '2023-06-10', 'lot_googl_1', 'Technology'),
('11111111-1111-1111-1111-111111111111', 'JNJ', 4000, 155.20, 160.00, '2023-02-01', 'lot_jnj_1', 'Healthcare'),
('11111111-1111-1111-1111-111111111111', 'JPM', 3500, 185.40, 175.00, '2023-04-15', 'lot_jpm_1', 'Financial');