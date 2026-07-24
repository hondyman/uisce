-- Verification Script for Semantic Financial Core

BEGIN;

-- 1. Setup Tenant (Mock)
INSERT INTO tenants (id, name) VALUES ('11111111-1111-1111-1111-111111111111', 'Test Tenant') ON CONFLICT DO NOTHING;

-- 2. Setup Portfolio
INSERT INTO portfolios (id, tenant_id, name, region, strategy) 
VALUES ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'Alpha Fund', 'US', 'Equity');

-- 3. Setup Market Prices
-- T-1 Price: 150.00
-- T-0 Price: 155.00
INSERT INTO market_prices (tenant_id, asset_id, price, valid_from)
VALUES 
('11111111-1111-1111-1111-111111111111', 'AAPL', 150.00, NOW() - INTERVAL '1 day'),
('11111111-1111-1111-1111-111111111111', 'AAPL', 155.00, NOW());

-- 4. Setup IBOR Data (Snapshot + Events)
-- Snapshot at T-2: 100 shares
INSERT INTO position_snapshots (tenant_id, account_id, asset_id, quantity, snapshot_date)
VALUES ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'AAPL', 100, (NOW() - INTERVAL '2 days')::DATE);

-- Event at T-1: Buy 50 shares
INSERT INTO position_events (tenant_id, account_id, asset_id, quantity_change, event_type, event_date)
VALUES ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'AAPL', 50, 'Trade', NOW() - INTERVAL '1 day');

-- 5. Verify get_position_as_of
-- Should be 100 (Snapshot) + 50 (Event) = 150
DO $$
DECLARE
    pos DECIMAL;
BEGIN
    pos := get_position_as_of('22222222-2222-2222-2222-222222222222', 'AAPL', NOW());
    IF pos != 150 THEN
        RAISE EXCEPTION 'get_position_as_of failed: expected 150, got %', pos;
    END IF;
    RAISE NOTICE 'get_position_as_of passed: %', pos;
END $$;

-- 6. Verify calculate_nav
-- Note: calculate_nav uses position_versions (Bi-Temporal), not snapshots/events.
-- We populate position_versions to match the IBOR state for this test.
INSERT INTO position_versions (entity_id, tenant_id, account_id, asset_id, quantity, valid_time)
VALUES (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'AAPL', 150, tstzrange(NOW() - INTERVAL '1 day', 'infinity'));

-- NAV = 150 * 155.00 (Latest Price) = 23250
DO $$
DECLARE
    nav DECIMAL;
    p_row portfolios%ROWTYPE;
BEGIN
    SELECT * INTO p_row FROM portfolios WHERE id = '22222222-2222-2222-2222-222222222222';
    nav := calculate_nav(p_row);
    IF nav != 23250 THEN
        RAISE EXCEPTION 'calculate_nav failed: expected 23250, got %', nav;
    END IF;
    RAISE NOTICE 'calculate_nav passed: %', nav;
END $$;

ROLLBACK; -- Clean up changes
