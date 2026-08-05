-- ============================================================================
-- 0001_init_schemas.sql
-- Creates oms, mds, ref schemas for the Soul Trader trade order system.
-- Run with: psql ... -v ON_ERROR_STOP=1 -1 -f 0001_init_schemas.sql ...
-- ============================================================================

BEGIN;

CREATE SCHEMA IF NOT EXISTS ref;
CREATE SCHEMA IF NOT EXISTS mds;
CREATE SCHEMA IF NOT EXISTS oms;

COMMENT ON SCHEMA ref  IS 'Reference / lookup data: currencies, exchanges, asset classes, order types, TIF, sides, event types.';
COMMENT ON SCHEMA mds  IS 'Master data: counterparties, accounts, portfolios, securities.';
COMMENT ON SCHEMA oms  IS 'Order management: orders, slices, links, events, executions, allocations, settlements, positions.';

-- Single-tenant default: this UUID must be back-filled from the actual tenant_id
-- once the uisce tenant row is created. Using a placeholder sentinel.
DO $$
BEGIN
  -- nothing to back-fill yet; seed-soul-trader will patch this post-insert
END $$;

COMMIT;
