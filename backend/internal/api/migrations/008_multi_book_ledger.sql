-- Migration 008: Multi-Book Accounting (Prism Pattern)

-- 1. Ledger Basis Reference Table
CREATE TABLE IF NOT EXISTS ref_ledger_basis (
    id TEXT PRIMARY KEY, -- 'IBOR', 'ABOR_GAAP', 'ABOR_IFRS', 'PBOR'
    description TEXT,
    requires_settlement BOOLEAN DEFAULT false
);

-- Seed default bases
INSERT INTO ref_ledger_basis (id, description, requires_settlement) VALUES
('IBOR', 'Investment Book of Record - Trade Date, Live Positions', false),
('ABOR', 'Accounting Book of Record - Settlement Date, GAAP/IFRS', true),
('PBOR', 'Performance Book of Record - Historical, Restated', false)
ON CONFLICT (id) DO NOTHING;

-- 2. Posting Rules Engine (Metadata)
CREATE TABLE IF NOT EXISTS meta_posting_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL, -- e.g., 'Trade', 'CorporateAction'
    asset_class TEXT, -- e.g., 'Equity', 'Bond' (Optional filter)
    rules_json JSONB NOT NULL, -- The "Fan-Out" logic
    -- Example JSON:
    -- [
    --   { "basis": "IBOR", "timing": "TradeDate", "account_role": "Inventory" },
    --   { "basis": "ABOR", "timing": "SettlementDate", "account_role": "Inventory" }
    -- ]
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, event_type, asset_class)
);

-- 3. Update Ledger Entries for Multi-Book Support
ALTER TABLE ledger_entries 
ADD COLUMN IF NOT EXISTS basis_id TEXT REFERENCES ref_ledger_basis(id) DEFAULT 'IBOR';

-- Update existing entries to be IBOR (Migration safety)
UPDATE ledger_entries SET basis_id = 'IBOR' WHERE basis_id IS NULL;

-- Add index for Basis filtering (The "Lens")
CREATE INDEX IF NOT EXISTS idx_ledger_basis ON ledger_entries (basis_id);

-- 4. RLS for Basis Views (The "Lens" Implementation)
-- Ensure RLS is enabled
ALTER TABLE ledger_entries ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see rows for bases they are allowed to see.
-- We assume 'hasura.user.allowed_bases' is a session variable (e.g., "{IBOR,ABOR}")
-- If not present, default to IBOR only or deny all depending on security posture.
-- Here we allow IBOR by default if variable is missing for backward compatibility/safety.
CREATE POLICY basis_access_policy ON ledger_entries
    FOR SELECT
    USING (
        basis_id = ANY(
            string_to_array(
                COALESCE(current_setting('hasura.user.allowed_bases', true), 'IBOR'), 
                ','
            )
        )
        OR
        current_setting('hasura.user.role', true) = 'admin'
    );
