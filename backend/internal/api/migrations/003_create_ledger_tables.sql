-- Bi-Temporal Ledger Table
CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    
    -- Bi-Temporal Columns
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_to TIMESTAMP WITH TIME ZONE NOT NULL, -- 'infinity' for current
    system_from TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    system_to TIMESTAMP WITH TIME ZONE DEFAULT 'infinity',

    status TEXT NOT NULL CHECK (status IN ('Pending', 'Committed', 'Deleted')),
    transaction_ref TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for "As Of" queries (Time Travel)
CREATE INDEX idx_ledger_valid_time ON ledger_entries (account_id, valid_from, valid_to);
CREATE INDEX idx_ledger_system_time ON ledger_entries (account_id, system_from, system_to);

-- CQRS Command Table
CREATE TABLE IF NOT EXISTS insert_trade_requests (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    trade_details JSONB NOT NULL,
    status TEXT DEFAULT 'Pending', -- Pending, Processing, Completed, Failed
    workflow_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
