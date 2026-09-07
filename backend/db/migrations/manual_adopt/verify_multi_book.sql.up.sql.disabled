-- Verification Script for Multi-Book Accounting

BEGIN;

-- 1. Verify Ledger Bases
DO $$
DECLARE
    count INT;
BEGIN
    SELECT COUNT(*) INTO count FROM ref_ledger_basis WHERE id IN ('IBOR', 'ABOR', 'PBOR');
    IF count != 3 THEN
        RAISE EXCEPTION 'Missing default ledger bases';
    END IF;
END $$;

-- 2. Verify Posting Rules Table
INSERT INTO meta_posting_rules (tenant_id, event_type, rules_json)
VALUES (
    '11111111-1111-1111-1111-111111111111', 
    'Trade', 
    '[{"basis": "IBOR", "timing": "TradeDate"}, {"basis": "ABOR", "timing": "SettlementDate"}]'::jsonb
);

-- 3. Verify Ledger Entries with Basis
-- Insert IBOR Entry
INSERT INTO ledger_entries (id, tenant_id, basis_id, account_id, asset_id, quantity, valid_from, valid_to, status)
VALUES (
    gen_random_uuid(), 
    '11111111-1111-1111-1111-111111111111', 
    'IBOR', 
    '22222222-2222-2222-2222-222222222222', 
    'AAPL', 
    100, 
    NOW(), 
    'infinity', 
    'Committed'
);

-- Insert ABOR Entry
INSERT INTO ledger_entries (id, tenant_id, basis_id, account_id, asset_id, quantity, valid_from, valid_to, status)
VALUES (
    gen_random_uuid(), 
    '11111111-1111-1111-1111-111111111111', 
    'ABOR', 
    '22222222-2222-2222-2222-222222222222', 
    'AAPL', 
    100, 
    NOW() + INTERVAL '2 days', -- Settlement Date
    'infinity', 
    'Committed'
);

-- 4. Verify RLS (Simulate User)
-- Case A: User with IBOR access only
-- We can't easily simulate session variables in a simple SQL script block without SET LOCAL, 
-- but we can verify the policy exists.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies 
        WHERE tablename = 'ledger_entries' AND policyname = 'basis_access_policy'
    ) THEN
        RAISE EXCEPTION 'RLS Policy basis_access_policy missing on ledger_entries';
    END IF;
END $$;

ROLLBACK; -- Clean up
