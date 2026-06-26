-- Migration 030: Cash Master Schema
-- Bi-temporal, RLS-enabled, multi-tenant cash ledger + balance system
-- Aligns with Whitepaper §3: Semantic Graph Architecture

CREATE SCHEMA IF NOT EXISTS edm;

-- ============================================
-- ROOT: Cash Balance Master (Point-in-Time Balances)
-- ============================================
CREATE TABLE IF NOT EXISTS edm.cash_balance_master (
    cash_balance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Core Identity (cluster key)
    portfolio_id UUID NOT NULL,
    cash_account_id TEXT,
    currency TEXT NOT NULL,
    valuation_date DATE NOT NULL,
    
    -- Balance Components (Roll-Forward)
    opening_balance NUMERIC(28,10),
    cash_inflows NUMERIC(28,10),
    cash_outflows NUMERIC(28,10),
    interest_accrual NUMERIC(28,10),
    fx_effect NUMERIC(28,10),
    closing_balance NUMERIC(28,10),
    
    -- Audit
    source_system TEXT NOT NULL,
    is_closed BOOLEAN DEFAULT FALSE,
    
    -- Bi-temporal Versioning (Semantic Design §6)
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    system_from TIMESTAMPTZ DEFAULT NOW(),
    system_to TIMESTAMPTZ DEFAULT 'infinity',
    
    -- Multi-tenant Lineage (Usice Architecture §6.2)
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE (portfolio_id, cash_account_id, currency, valuation_date, valid_from),
    CONSTRAINT chk_currency_format CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_balance_consistency CHECK (
        closing_balance = opening_balance + cash_inflows - cash_outflows + interest_accrual + fx_effect
    )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_cash_balance_cluster ON edm.cash_balance_master (portfolio_id, currency, valuation_date);
CREATE INDEX IF NOT EXISTS idx_cash_balance_tenant ON edm.cash_balance_master (tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_cash_balance_valid ON edm.cash_balance_master (valid_from, valid_to) WHERE valid_to = 'infinity';

-- RLS Policies (Usice Architecture §6.2: Enforcement Layers)
ALTER TABLE edm.cash_balance_master ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    CREATE POLICY cash_balance_tenant_isolation ON edm.cash_balance_master
        FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE POLICY cash_balance_core_read ON edm.cash_balance_master
        FOR SELECT USING (
            current_setting('app.tenant_scope', TRUE) IN ('multi', 'all') 
            OR tenant_id = current_setting('app.current_tenant', TRUE)::UUID
        );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ============================================
-- LEDGER: Cash Ledger Entry (Atomic Cash Events)
-- ============================================
CREATE TABLE IF NOT EXISTS edm.cash_ledger (
    cash_ledger_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Keys
    portfolio_id UUID NOT NULL,
    cash_account_id TEXT,
    currency TEXT NOT NULL,
    
    -- Dates
    value_date DATE NOT NULL,
    booking_date DATE,
    
    -- Classification
    cash_event_type TEXT NOT NULL, -- SETTLEMENT, INCOME, FEE, FX, CONTRIBUTION, WITHDRAWAL
    cash_event_subtype TEXT,
    
    -- Amounts
    amount NUMERIC(28,10) NOT NULL,
    amount_sign TEXT CHECK (amount_sign IN ('POSITIVE', 'NEGATIVE')),
    
    -- Links to Other Domains (Semantic Graph Edges)
    transaction_id UUID REFERENCES edm.transaction_master(transaction_id),
    security_id UUID REFERENCES edm.security_master(security_id),
    
    -- Counterparty / Account
    counterparty_id TEXT,
    
    -- Status / Audit
    status TEXT DEFAULT 'PENDING', -- PENDING, POSTED, CANCELLED
    source_system TEXT NOT NULL,
    external_reference TEXT,
    
    -- Bi-temporal + Tenant
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE (portfolio_id, value_date, external_reference, valid_from),
    CONSTRAINT chk_ledger_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_amount_nonzero CHECK (amount != 0)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_cash_ledger_portfolio ON edm.cash_ledger (portfolio_id, value_date);
CREATE INDEX IF NOT EXISTS idx_cash_ledger_currency ON edm.cash_ledger (currency, value_date);
CREATE INDEX IF NOT EXISTS idx_cash_ledger_tx ON edm.cash_ledger (transaction_id);
CREATE INDEX IF NOT EXISTS idx_cash_ledger_tenant ON edm.cash_ledger (tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_cash_ledger_valid ON edm.cash_ledger (valid_from, valid_to) WHERE valid_to = 'infinity';

-- RLS
ALTER TABLE edm.cash_ledger ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    CREATE POLICY cash_ledger_tenant_isolation ON edm.cash_ledger
        FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ============================================
-- TRACE: Cash Ledger → Balance Lineage
-- ============================================
CREATE TABLE IF NOT EXISTS edm.cash_flow_trace (
    trace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cash_balance_id UUID NOT NULL REFERENCES edm.cash_balance_master(cash_balance_id),
    cash_ledger_id UUID REFERENCES edm.cash_ledger(cash_ledger_id),
    contribution_amount NUMERIC(28,10),
    contribution_type TEXT, -- INFLOW, OUTFLOW, INTEREST, FX
    processed_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_flow_trace_balance ON edm.cash_flow_trace (cash_balance_id);
CREATE INDEX IF NOT EXISTS idx_flow_trace_ledger ON edm.cash_flow_trace (cash_ledger_id);

-- ============================================
-- GOLD TRACE: Survivorship Lineage (Whitepaper §7)
-- ============================================
CREATE TABLE IF NOT EXISTS edm.cash_gold_trace (
    trace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cash_ledger_id UUID NOT NULL,
    field_name TEXT NOT NULL,
    source_system TEXT NOT NULL,
    source_value JSONB,
    was_selected BOOLEAN,
    selection_reason TEXT,
    survivorship_run_id UUID,
    traced_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cash_gold_trace ON edm.cash_gold_trace (cash_ledger_id, field_name);
