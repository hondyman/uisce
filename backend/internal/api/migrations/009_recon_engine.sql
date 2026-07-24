-- Migration 009: Reconciliation Engine ("Immune System")

-- 1. Recon Exceptions Table
-- Stores "Breaks" found between Internal Ledger and External Custodian
CREATE TABLE IF NOT EXISTS recon_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- The Break Details
    recon_date DATE NOT NULL,
    account_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    
    internal_quantity DECIMAL(20, 8),
    external_quantity DECIMAL(20, 8),
    diff_quantity DECIMAL(20, 8) GENERATED ALWAYS AS (internal_quantity - external_quantity) STORED,
    
    status TEXT DEFAULT 'Open', -- Open, Investigating, Resolved, Ignored
    resolution_notes TEXT,
    assigned_user_id UUID,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, recon_date, account_id, asset_id)
);

-- Index for finding open breaks
CREATE INDEX idx_recon_exceptions_status ON recon_exceptions (status) WHERE status != 'Resolved';
