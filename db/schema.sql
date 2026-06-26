-- schema.sql
-- Runnable schema for Investment Front Office Platform

-- Enable UUID extension for primary keys
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table (for multitenancy management)
CREATE TABLE IF NOT EXISTS tenants (
    tenant_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Consolidated Client/Investor table (STI pattern)
CREATE TABLE IF NOT EXISTS client_investors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('ClientInvestor', 'IndividualInvestor', 'InstitutionalInvestor', 'FamilyOffice')),
    name VARCHAR(255) NOT NULL,
    contact_details JSONB,
    risk_tolerance VARCHAR(50) CHECK (risk_tolerance IN ('Low', 'Medium', 'High')),
    investment_objectives TEXT[],
    regulatory_status VARCHAR(100),
    age INTEGER,
    tax_id VARCHAR(50),
    beneficiary VARCHAR(255),
    org_type VARCHAR(100),
    signatories TEXT[],
    custody VARCHAR(255),
    generations INTEGER,
    aggregated_reporting BOOLEAN DEFAULT FALSE,
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Consolidated Portfolio table (STI pattern)
CREATE TABLE IF NOT EXISTS portfolios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES client_investors(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('Portfolio', 'DiscretionaryPortfolio', 'NonDiscretionaryPortfolio', 'ModelPortfolio')),
    benchmark VARCHAR(100),
    asset_allocation_targets JSONB,
    performance_metrics JSONB,
    advisor_discretion BOOLEAN DEFAULT TRUE,
    client_approval_required BOOLEAN DEFAULT TRUE,
    template_id UUID,
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trades table (STI pattern)
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('Trade', 'BlockTrade', 'ProgramTrade')),
    trade_date TIMESTAMPTZ NOT NULL,
    security_ticker VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL CHECK (side IN ('Buy', 'Sell')),
    quantity NUMERIC(15, 4) NOT NULL,
    price NUMERIC(12, 6) NOT NULL,
    total_value NUMERIC(15, 2) GENERATED ALWAYS AS ((quantity * price)) STORED,
    commission NUMERIC(10, 4),
    settlement_date TIMESTAMPTZ,
    -- Subtype-specific fields
    block_size_threshold NUMERIC(10, 2), -- BlockTrade
    basket_composition JSONB, -- ProgramTrade
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    -- Use security_ticker as the 'name' for consistency in all_entities view
    name VARCHAR(255) GENERATED ALWAYS AS (security_ticker) STORED
);

-- Indexes for trades table
CREATE INDEX IF NOT EXISTS idx_trades_tenant ON trades(tenant_id);
CREATE INDEX IF NOT EXISTS idx_trades_portfolio ON trades(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_trades_date ON trades(trade_date);
CREATE INDEX IF NOT EXISTS idx_trades_ticker ON trades(security_ticker);

-- Entity manager tables
CREATE TABLE IF NOT EXISTS entity_registry (
  entity_name TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  default_schema JSONB NOT NULL,
  subtypes JSONB DEFAULT '[]',
  validation_rules JSONB DEFAULT '{}'::jsonb,
  workflow_hooks JSONB DEFAULT '{}'::jsonb,
  policy_ids TEXT[] DEFAULT '{}'::text[],
  event_mappings JSONB DEFAULT '{}'::jsonb,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tenant_entity_customizations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  entity_name TEXT NOT NULL REFERENCES entity_registry(entity_name),
  schema_overrides JSONB DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS migration_jobs (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  job_name TEXT NOT NULL,
  job_spec JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  started_at TIMESTAMP WITH TIME ZONE,
  completed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_client_investors_tenant ON client_investors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_client_investors_type ON client_investors(type);
CREATE INDEX IF NOT EXISTS idx_portfolios_tenant ON portfolios(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_client ON portfolios(client_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_type ON portfolios(type);

-- Partial indexes for subtype-specific queries (saves space / improves performance)
CREATE INDEX IF NOT EXISTS idx_client_investors_individual_tax_id
ON client_investors (tax_id)
WHERE type = 'IndividualInvestor';

CREATE INDEX IF NOT EXISTS idx_portfolios_model_template_id
ON portfolios (template_id)
WHERE type = 'ModelPortfolio';

-- Function and triggers
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_client_investors_ts ON client_investors;
CREATE TRIGGER update_client_investors_ts
BEFORE UPDATE ON client_investors
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

DROP TRIGGER IF EXISTS update_portfolios_ts ON portfolios;
CREATE TRIGGER update_portfolios_ts
BEFORE UPDATE ON portfolios
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

DROP TRIGGER IF EXISTS update_trades_ts ON trades;
CREATE TRIGGER update_trades_ts
BEFORE UPDATE ON trades
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

DROP TRIGGER IF EXISTS update_entity_registry_ts ON entity_registry;
CREATE TRIGGER update_entity_registry_ts
BEFORE UPDATE ON entity_registry
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- View for unified querying across entities (for Hasura)
CREATE OR REPLACE VIEW all_entities AS
SELECT 
    'client_investors' AS entity_type, 
    id, 
    tenant_id, 
    name, 
    custom_fields, 
    created_at, 
    row_to_json(t.*) AS full_data
FROM client_investors t
UNION ALL
SELECT 
    'portfolios' AS entity_type, 
    id, 
    tenant_id, 
    benchmark AS name,  
    custom_fields, 
    created_at, 
    row_to_json(t1.*) AS full_data
FROM portfolios t1
UNION ALL
SELECT
    'trades' AS entity_type,
    id,
    tenant_id,
    name,
    custom_fields,
    created_at,
    row_to_json(t2.*) AS full_data
FROM trades t2;

-- SECURITY: Enable Row Level Security (RLS) to enforce tenant isolation at the DB level.
-- Hasura will set session variables from JWT claims like 'hasura.jwt.claims.X-Hasura-Tenant-Id'
ALTER TABLE client_investors ENABLE ROW LEVEL SECURITY;
ALTER TABLE portfolios ENABLE ROW LEVEL SECURITY;

-- Policy: only allow access to rows that belong to the tenant from JWT claims
CREATE POLICY IF NOT EXISTS tenant_isolation_policy_client_investors ON client_investors
  FOR ALL
  USING (tenant_id = current_setting('hasura.jwt.claims.X-Hasura-Tenant-Id')::uuid);

CREATE POLICY IF NOT EXISTS tenant_isolation_policy_portfolios ON portfolios
  FOR ALL
  USING (tenant_id = current_setting('hasura.jwt.claims.X-Hasura-Tenant-Id')::uuid);

ALTER TABLE trades ENABLE ROW LEVEL SECURITY;
CREATE POLICY IF NOT EXISTS tenant_isolation_policy_trades ON trades
  FOR ALL
  USING (tenant_id = current_setting('hasura.jwt.claims.X-Hasura-Tenant-Id')::uuid);

-- JSONB validation helpers
-- Example: ensure contact_details JSONB contains an 'email' key
CREATE OR REPLACE FUNCTION has_email(data JSONB)
RETURNS BOOLEAN AS $$
BEGIN
  RETURN data ? 'email';
END;
$$ LANGUAGE plpgsql IMMUTABLE;

ALTER TABLE client_investors
  ADD CONSTRAINT IF NOT EXISTS check_contact_has_email
  CHECK (contact_details IS NULL OR has_email(contact_details));

-- Computed field: display label for client_investors (used by Hasura as a computed field)
CREATE OR REPLACE FUNCTION client_investor_display_label(investor_row client_investors)
RETURNS TEXT AS $
BEGIN
  RETURN investor_row.name || ' (Risk: ' || COALESCE(investor_row.risk_tolerance, 'N/A') || ')';
END;
$ LANGUAGE plpgsql STABLE;

--
-- Business Object Event Tracking
--

CREATE TABLE IF NOT EXISTS bo_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    bo_type VARCHAR(50),           -- "client_investors"
    bo_id UUID NOT NULL,           -- John Smith's UUID
    changed_by UUID,               -- Trader Jane's user_id
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    field_name VARCHAR(50),        -- "risk_tolerance"
    old_value JSONB,               -- "Medium"
    new_value JSONB,               -- "High"
    bp_step VARCHAR(50),           -- "APPROVED"
    custom_data JSONB              -- {"compliance_alert": true}
);

CREATE INDEX IF NOT EXISTS idx_bo_events_bo ON bo_events(bo_type, bo_id);
CREATE INDEX IF NOT EXISTS idx_bo_events_time ON bo_events(changed_at);
CREATE INDEX IF NOT EXISTS idx_bo_events_tenant ON bo_events(tenant_id);


CREATE OR REPLACE FUNCTION log_field_change()
RETURNS TRIGGER AS $
DECLARE
    user_id UUID;
BEGIN
    BEGIN
        user_id := current_setting('hasura.jwt.claims.X-Hasura-User-Id')::uuid;
    EXCEPTION WHEN OTHERS THEN
        user_id := '00000000-0000-0000-0000-000000000000'; -- System user
    END;

    IF TG_OP = 'UPDATE' THEN
        INSERT INTO bo_events (tenant_id, bo_type, bo_id, changed_by, field_name, old_value, new_value)
        VALUES (NEW.tenant_id, TG_TABLE_NAME, NEW.id, user_id, TG_OP, row_to_json(OLD), row_to_json(NEW));
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO bo_events (tenant_id, bo_type, bo_id, changed_by, field_name, new_value)
        VALUES (NEW.tenant_id, TG_TABLE_NAME, NEW.id, user_id, TG_OP, row_to_json(NEW));
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO bo_events (tenant_id, bo_type, bo_id, changed_by, field_name, old_value)
        VALUES (OLD.tenant_id, TG_TABLE_NAME, OLD.id, user_id, TG_OP, row_to_json(OLD));
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS client_investors_changes ON client_investors;
CREATE TRIGGER client_investors_changes
AFTER INSERT OR UPDATE OR DELETE ON client_investors
FOR EACH ROW EXECUTE FUNCTION log_field_change();
