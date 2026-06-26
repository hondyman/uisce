-- ============================================================================
-- REBALANCING SCHEMA: Portfolio Rebalancing Tables
-- ============================================================================

-- 1. Proposed Trades Table
CREATE TABLE proposed_trades (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  portfolio_id UUID NOT NULL,
  workflow_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('buy', 'sell')), -- buy or sell
  shares DECIMAL(12, 4) NOT NULL,
  price DECIMAL(12, 2) NOT NULL,
  unrealized_gain DECIMAL(14, 2),
  days_held INT,
  is_tax_harvest BOOLEAN DEFAULT false,
  proposed_at TIMESTAMP NOT NULL DEFAULT NOW(),
  status TEXT DEFAULT 'proposed' CHECK (status IN ('proposed', 'approved', 'executed', 'cancelled')),
  execution_price DECIMAL(12, 2),
  executed_at TIMESTAMP,
  execution_error TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- 2. Rebalance Audit Table (Immutable)
CREATE TABLE rebalance_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  portfolio_id UUID NOT NULL,
  workflow_id TEXT NOT NULL UNIQUE,
  triggered_by TEXT NOT NULL,
  drift_before DECIMAL(10, 4) NOT NULL,
  drift_after DECIMAL(10, 4),
  tax_saved DECIMAL(14, 2) NOT NULL DEFAULT 0,
  estimated_tax_debt DECIMAL(14, 2) NOT NULL DEFAULT 0,
  trades_proposed INT NOT NULL DEFAULT 0,
  trades_executed INT NOT NULL DEFAULT 0,
  trades_failed INT NOT NULL DEFAULT 0,
  total_commission DECIMAL(14, 2) NOT NULL DEFAULT 0,
  policy_version INT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('proposed', 'approved', 'executing', 'completed', 'failed')),
  error_message TEXT,
  metadata JSONB,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT immutable_check CHECK (created_at IS NOT NULL)
);

-- 3. Trade Execution Log (Immutable)
CREATE TABLE trade_execution_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  proposed_trade_id UUID NOT NULL,
  custodian TEXT NOT NULL, -- schwab, fidelity, pershing, etc.
  order_id TEXT,
  symbol TEXT NOT NULL,
  action TEXT NOT NULL,
  shares DECIMAL(12, 4) NOT NULL,
  price DECIMAL(12, 2) NOT NULL,
  gross_amount DECIMAL(14, 2) NOT NULL,
  commission DECIMAL(12, 2) NOT NULL DEFAULT 0,
  net_amount DECIMAL(14, 2) NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'submitted', 'filled', 'partial', 'cancelled', 'failed')),
  settlement_date DATE,
  executed_at TIMESTAMP,
  error_message TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_proposed_trade FOREIGN KEY (proposed_trade_id) REFERENCES proposed_trades(id) ON DELETE CASCADE
);

-- 4. Allocation Models Table (for Hasura GraphQL auto-gen)
CREATE TABLE allocation_models (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  model_type TEXT NOT NULL CHECK (model_type IN ('balanced', '60-40', '70-30', '80-20', 'aggressive', 'conservative', 'custom')),
  created_by TEXT NOT NULL,
  is_active BOOLEAN DEFAULT true,
  allocations JSONB NOT NULL, -- Array of {asset_class, target_percent, min_percent, max_percent, benchmark}
  metadata JSONB,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- 5. Rebalance Execution History (for Hasura subscriptions)
CREATE TABLE rebalance_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  portfolio_id UUID NOT NULL,
  workflow_id TEXT NOT NULL,
  audit_id UUID NOT NULL,
  step INT NOT NULL,
  step_name TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
  drift_at_step DECIMAL(10, 4),
  trades_at_step INT,
  tax_impact_at_step DECIMAL(14, 2),
  duration_ms INT,
  error_message TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_audit FOREIGN KEY (audit_id) REFERENCES rebalance_audit(id) ON DELETE CASCADE
);

-- ============================================================================
-- INDEXES for Query Performance
-- ============================================================================

CREATE INDEX idx_proposed_trades_portfolio ON proposed_trades(tenant_id, portfolio_id);
CREATE INDEX idx_proposed_trades_status ON proposed_trades(status);
CREATE INDEX idx_proposed_trades_workflow ON proposed_trades(workflow_id);
CREATE INDEX idx_proposed_trades_tax_harvest ON proposed_trades(is_tax_harvest, unrealized_gain);

CREATE INDEX idx_rebalance_audit_portfolio ON rebalance_audit(tenant_id, portfolio_id);
CREATE INDEX idx_rebalance_audit_workflow ON rebalance_audit(workflow_id);
CREATE INDEX idx_rebalance_audit_status ON rebalance_audit(status, created_at);
CREATE INDEX idx_rebalance_audit_triggered_by ON rebalance_audit(triggered_by);

CREATE INDEX idx_trade_execution_log_status ON trade_execution_log(status);
CREATE INDEX idx_trade_execution_log_custodian ON trade_execution_log(custodian);
CREATE INDEX idx_trade_execution_log_symbol ON trade_execution_log(symbol);

CREATE INDEX idx_allocation_models_tenant_active ON allocation_models(tenant_id, is_active);

CREATE INDEX idx_rebalance_executions_workflow ON rebalance_executions(workflow_id);
CREATE INDEX idx_rebalance_executions_status ON rebalance_executions(status);

-- ============================================================================
-- MATERIALIZED VIEW: Rebalance Summary (for Hasura subscriptions + real-time dashboard)
-- ============================================================================

CREATE MATERIALIZED VIEW v_rebalance_summary AS
SELECT
  ra.tenant_id,
  ra.portfolio_id,
  ra.id AS audit_id,
  ra.workflow_id,
  ra.status,
  ra.created_at::DATE AS rebalance_date,
  EXTRACT(EPOCH FROM (NOW() - ra.created_at)) / 3600 AS hours_ago,
  ra.drift_before,
  ra.drift_after,
  ra.tax_saved,
  ra.estimated_tax_debt,
  ra.trades_proposed,
  ra.trades_executed,
  ra.trades_failed,
  ra.total_commission,
  (SELECT COUNT(*) FROM proposed_trades pt WHERE pt.workflow_id = ra.workflow_id AND pt.status = 'executed') AS trades_actually_executed,
  (SELECT SUM(gross_amount) FROM trade_execution_log tel WHERE tel.tenant_id = ra.tenant_id AND tel.proposed_trade_id IN (SELECT id FROM proposed_trades WHERE workflow_id = ra.workflow_id)) AS gross_trade_value,
  (SELECT SUM(commission) FROM trade_execution_log tel WHERE tel.tenant_id = ra.tenant_id AND tel.proposed_trade_id IN (SELECT id FROM proposed_trades WHERE workflow_id = ra.workflow_id)) AS total_commissions,
  ra.triggered_by,
  ra.policy_version
FROM rebalance_audit ra
ORDER BY ra.created_at DESC;

CREATE INDEX idx_v_rebalance_summary_portfolio ON v_rebalance_summary(tenant_id, portfolio_id);
CREATE INDEX idx_v_rebalance_summary_status ON v_rebalance_summary(status);

-- ============================================================================
-- ROW LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

ALTER TABLE proposed_trades ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_proposed_trades ON proposed_trades
  USING (tenant_id = CURRENT_SETTING('app.current_tenant_id')::uuid);

ALTER TABLE rebalance_audit ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_rebalance_audit ON rebalance_audit
  USING (tenant_id = CURRENT_SETTING('app.current_tenant_id')::uuid);

ALTER TABLE trade_execution_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_trade_execution ON trade_execution_log
  USING (tenant_id = CURRENT_SETTING('app.current_tenant_id')::uuid);

ALTER TABLE allocation_models ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_allocation_models ON allocation_models
  USING (tenant_id = CURRENT_SETTING('app.current_tenant_id')::uuid);

ALTER TABLE rebalance_executions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_rebalance_executions ON rebalance_executions
  USING (tenant_id = CURRENT_SETTING('app.current_tenant_id')::uuid);

-- ============================================================================
-- TRIGGER: Update rebalance_audit.updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION update_rebalance_audit_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_rebalance_audit_timestamp
  BEFORE UPDATE ON rebalance_audit
  FOR EACH ROW
  EXECUTE FUNCTION update_rebalance_audit_timestamp();

-- ============================================================================
-- TRIGGER: Auto-update materialized view
-- ============================================================================

CREATE OR REPLACE FUNCTION refresh_rebalance_summary()
RETURNS TRIGGER AS $$
BEGIN
  REFRESH MATERIALIZED VIEW CONCURRENTLY v_rebalance_summary;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_refresh_rebalance_summary_after_audit
  AFTER INSERT ON rebalance_audit
  FOR EACH ROW
  EXECUTE FUNCTION refresh_rebalance_summary();

-- ============================================================================
-- SAMPLE DATA (for development/testing)
-- ============================================================================

INSERT INTO allocation_models (tenant_id, name, description, model_type, created_by, allocations)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  'Classic 60/40',
  'Traditional balanced portfolio: 60% stocks, 40% bonds',
  '60-40',
  'system',
  '[
    {
      "asset_class": "US Equities",
      "target_percent": 0.60,
      "min_percent": 0.55,
      "max_percent": 0.65,
      "benchmark": "SPY"
    },
    {
      "asset_class": "Bonds",
      "target_percent": 0.30,
      "min_percent": 0.25,
      "max_percent": 0.35,
      "benchmark": "BND"
    },
    {
      "asset_class": "Intl Equities",
      "target_percent": 0.07,
      "min_percent": 0.05,
      "max_percent": 0.10,
      "benchmark": "VXUS"
    },
    {
      "asset_class": "Real Estate",
      "target_percent": 0.03,
      "min_percent": 0.00,
      "max_percent": 0.05,
      "benchmark": "VNQ"
    }
  ]'
);

INSERT INTO allocation_models (tenant_id, name, description, model_type, created_by, allocations)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  'Aggressive Growth',
  'Growth-focused: 80% stocks, 20% alternatives',
  '80-20',
  'system',
  '[
    {
      "asset_class": "US Equities",
      "target_percent": 0.65,
      "min_percent": 0.60,
      "max_percent": 0.70,
      "benchmark": "QQQ"
    },
    {
      "asset_class": "Intl Equities",
      "target_percent": 0.15,
      "min_percent": 0.10,
      "max_percent": 0.20,
      "benchmark": "VXUS"
    },
    {
      "asset_class": "Alternatives",
      "target_percent": 0.20,
      "min_percent": 0.15,
      "max_percent": 0.25,
      "benchmark": "PDBC"
    }
  ]'
);
