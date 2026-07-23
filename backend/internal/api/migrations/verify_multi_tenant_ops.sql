-- Verification Script for Multi-Tenant Ops Architecture
-- Run this on the Central Ops Postgres DB.

BEGIN;

-- 1. Verify Tables Exist
SELECT * FROM tenants LIMIT 1;
SELECT * FROM exceptions LIMIT 1;
SELECT * FROM workflows LIMIT 1;
SELECT * FROM audit_records LIMIT 1;

-- 2. Verify RLS Policies Exist
SELECT * FROM pg_policies WHERE tablename = 'exceptions';
SELECT * FROM pg_policies WHERE tablename = 'workflows';

-- 3. Test RLS Isolation (Mock)
-- Set Tenant Context
SET app.tenant_id = '11111111-1111-1111-1111-111111111111';

-- Insert Exception for Tenant A
INSERT INTO exceptions (tenant_id, source_system, description)
VALUES (current_setting('app.tenant_id')::uuid, 'TradeSystem', 'Test Exception A');

-- Verify Visibility
DO $$
DECLARE
    cnt INT;
BEGIN
    SELECT count(*) INTO cnt FROM exceptions;
    IF cnt != 1 THEN
        RAISE EXCEPTION 'RLS Failed: Expected 1 row, got %', cnt;
    END IF;
END $$;

-- Switch Context to Tenant B
SET app.tenant_id = '22222222-2222-2222-2222-222222222222';

-- Verify Isolation (Should see 0 rows from Tenant A)
DO $$
DECLARE
    cnt INT;
BEGIN
    SELECT count(*) INTO cnt FROM exceptions;
    IF cnt != 0 THEN
        RAISE EXCEPTION 'RLS Failed: Expected 0 rows for Tenant B, got %', cnt;
    END IF;
END $$;

ROLLBACK; -- Clean up
