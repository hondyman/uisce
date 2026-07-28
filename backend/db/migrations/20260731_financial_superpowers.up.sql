-- Migration: Financial Superpowers Bedrock (Instrument Master, Pre-Trade Compliance, IBOR/ABOR Posting, Household Graph)
-- Date: 2026-07-31
-- Purpose: Schema for Instrument Symbology Master, Pre-Trade Compliance Rules, IBOR/ABOR Ledger Postings, and Household Tax-Loss Harvesting.

-- 1. Instrument Master & Symbology Resolution Table
CREATE TABLE IF NOT EXISTS public.financial_instrument_master (
    instrument_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    primary_ticker VARCHAR(32) NOT NULL,
    isin VARCHAR(12),
    cusip VARCHAR(9),
    sedol VARCHAR(7),
    figi VARCHAR(12),
    instrument_name VARCHAR(255) NOT NULL,
    asset_class VARCHAR(64) NOT NULL, -- EQUITY, FIXED_INCOME, DERIVATIVE, PRIVATE_CREDIT
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    feed_survivorship_rules JSONB NOT NULL DEFAULT '{"pricing_priority": ["BLOOMBERG", "REUTERS", "SP"]}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT uk_instrument_ticker UNIQUE (tenant_id, primary_ticker)
);

CREATE INDEX IF NOT EXISTS idx_instrument_symbology ON public.financial_instrument_master(tenant_id, isin, cusip, figi);

-- 2. Pre- & Post-Trade Compliance Constraints Table
CREATE TABLE IF NOT EXISTS public.financial_compliance_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    target_entity_type VARCHAR(64) NOT NULL, -- PORTFOLIO, ACCOUNT, ORDER
    rule_expression VARCHAR(512) NOT NULL,   -- e.g. 'Holding.market_value / Portfolio.total_value <= 0.05'
    severity VARCHAR(32) NOT NULL DEFAULT 'BLOCK', -- BLOCK, WARN
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_compliance_rules ON public.financial_compliance_rules(tenant_id, target_entity_type, is_active);

-- 3. IBOR / ABOR Transaction Posting Behavior Rules Table
CREATE TABLE IF NOT EXISTS public.financial_posting_behaviors (
    behavior_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL, -- TRADE_BUY, TRADE_SELL, DIVIDEND, INTEREST_ACCRUAL
    asset_class VARCHAR(64) NOT NULL,
    posting_rules JSONB NOT NULL,    -- Cash movement, position lot updates, IBOR/ABOR ledger mapping
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

-- 4. Household Tax Lots & Optimization Records Table
CREATE TABLE IF NOT EXISTS public.financial_household_tax_lots (
    lot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    household_id VARCHAR(128) NOT NULL,
    account_id VARCHAR(128) NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    quantity NUMERIC(18, 6) NOT NULL,
    cost_basis NUMERIC(18, 4) NOT NULL,
    current_price NUMERIC(18, 4) NOT NULL,
    unrealized_gain_loss NUMERIC(18, 4) NOT NULL,
    acquisition_date DATE NOT NULL,
    tax_status VARCHAR(32) NOT NULL DEFAULT 'TAXABLE' -- TAXABLE, TAX_DEFERRED, TAX_EXEMPT
);

CREATE INDEX IF NOT EXISTS idx_household_tax_lots ON public.financial_household_tax_lots(tenant_id, household_id, tax_status);
