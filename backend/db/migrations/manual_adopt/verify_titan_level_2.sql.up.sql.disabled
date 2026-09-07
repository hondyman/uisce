-- Verification Script for Titan Level 2 (Recon Engine)

BEGIN;

-- 1. Verify Recon Exceptions Table
-- Insert a break
INSERT INTO recon_exceptions (
    tenant_id, 
    recon_date, 
    account_id, 
    asset_id, 
    internal_quantity, 
    external_quantity
)
VALUES (
    '11111111-1111-1111-1111-111111111111', 
    NOW()::DATE, 
    'ACC-001', 
    'AAPL', 
    100.00, 
    90.00
);

-- 2. Verify Generated Column (diff_quantity)
DO $$
DECLARE
    diff DECIMAL;
BEGIN
    SELECT diff_quantity INTO diff 
    FROM recon_exceptions 
    WHERE account_id = 'ACC-001' AND asset_id = 'AAPL';
    
    IF diff != 10.00 THEN
        RAISE EXCEPTION 'Generated Column diff_quantity failed: expected 10.00, got %', diff;
    END IF;
    RAISE NOTICE 'Recon Exception verification passed: Diff = %', diff;
END $$;

ROLLBACK; -- Clean up
