-- Phase 1: Rebalancing Ecosystem Database Schema

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Households: Top-level entity for wash sale tracking across accounts
CREATE TABLE IF NOT EXISTS households (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    tax_id_encrypted TEXT, -- Encrypted SSN/Tax ID
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Accounts: Individual brokerage accounts linked to a household
CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY, -- Brokerage Account Number
    household_id UUID NOT NULL REFERENCES households(id),
    name TEXT NOT NULL,
    account_type TEXT NOT NULL, -- 'TAXABLE', 'IRA', 'ROTH', '401K'
    custodian TEXT NOT NULL, -- 'SCHWAB', 'FIDELITY', 'IBKR'
    status TEXT NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'CLOSED', 'PENDING'
    config JSONB DEFAULT '{}'::jsonb, -- Rebalancing config (drift thresholds, etc.)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Market Data: Pricing and reference data
CREATE TABLE IF NOT EXISTS market_data (
    ticker TEXT NOT NULL,
    date DATE NOT NULL,
    close_price NUMERIC NOT NULL,
    volume BIGINT,
    currency TEXT DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (ticker, date)
);

-- 4. Tax Lots: The atomic unit of inventory
CREATE TABLE IF NOT EXISTS tax_lots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id TEXT NOT NULL REFERENCES accounts(id),
    ticker TEXT NOT NULL,
    quantity NUMERIC NOT NULL,
    acquired_date DATE NOT NULL,
    cost_basis NUMERIC NOT NULL, -- Per share basis
    unit_cost NUMERIC NOT NULL, -- Total cost / quantity
    currency TEXT DEFAULT 'USD',
    disposition_date DATE, -- NULL if still held
    status TEXT NOT NULL DEFAULT 'OPEN', -- 'OPEN', 'CLOSED', 'WASH_SALE'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Trades: Immutable log of all executions
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id TEXT NOT NULL REFERENCES accounts(id),
    ticker TEXT NOT NULL,
    side TEXT NOT NULL, -- 'BUY', 'SELL'
    quantity NUMERIC NOT NULL,
    price NUMERIC NOT NULL,
    trade_date TIMESTAMPTZ NOT NULL,
    settlement_date DATE,
    commission NUMERIC DEFAULT 0,
    lot_ids UUID[], -- Array of tax_lot IDs affected (for sells)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Wash Sales: Tracking disallowed losses
CREATE TABLE IF NOT EXISTS wash_sales (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_lot_id UUID NOT NULL REFERENCES tax_lots(id),
    replacement_lot_id UUID REFERENCES tax_lots(id), -- NULL if replacement not yet identified (or strictly disallowed)
    disallowed_loss NUMERIC NOT NULL,
    wash_date DATE NOT NULL,
    expiration_date DATE NOT NULL, -- When the wash sale window closes
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_accounts_household ON accounts(household_id);
CREATE INDEX IF NOT EXISTS idx_tax_lots_account_ticker ON tax_lots(account_id, ticker) WHERE status = 'OPEN';
CREATE INDEX IF NOT EXISTS idx_tax_lots_acquired ON tax_lots(acquired_date);
CREATE INDEX IF NOT EXISTS idx_trades_account_date ON trades(account_id, trade_date DESC);
CREATE INDEX IF NOT EXISTS idx_wash_sales_expiration ON wash_sales(expiration_date);
