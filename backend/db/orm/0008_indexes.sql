-- ============================================================================
-- 0008_indexes.sql
-- Supplemental / performance indexes not covered by prior migration files.
-- Run last after all table DDL files are applied.
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- Additional indexes on existing tables not covered above
-- --------------------------------------------------------------------------

-- mds.portfolio: add benchmark lookup
CREATE INDEX IF NOT EXISTS idx_portfolio_benchmark ON mds.portfolio (benchmark_id) WHERE benchmark_id IS NOT NULL;

-- mds.security_master: composite for equity screening
CREATE INDEX IF NOT EXISTS idx_security_eq_screen
    ON mds.security_master (tenant_id, eq_sector, eq_industry)
    WHERE asset_class = 'EQUITY' AND is_active = true;

-- mds.security_master: composite for FI screening
CREATE INDEX IF NOT EXISTS idx_security_fi_screen
    ON mds.security_master (tenant_id, fi_rating, fi_maturity_date)
    WHERE asset_class = 'FIXED_INCOME' AND is_active = true;

-- mds.security_master: FX pairs
CREATE INDEX IF NOT EXISTS idx_security_fx_pair
    ON mds.security_master (tenant_id, fx_ccy_base, fx_ccy_quote)
    WHERE asset_class = 'FX' AND is_active = true;

-- oms.orders: stale-orders cleanup sweep
CREATE INDEX IF NOT EXISTS idx_orders_stale
    ON oms.orders (tenant_id, created_at ASC);

-- oms.order_slice: stale slices
CREATE INDEX IF NOT EXISTS idx_slice_stale
    ON oms.order_slice (tenant_id, created_at ASC);

-- oms.execution: daily settlement reconciliation
CREATE INDEX IF NOT EXISTS idx_exec_settlement_date
    ON oms.execution (tenant_id, settlement_date, currency_id)
    WHERE settlement_date IS NOT NULL;

-- oms.execution: p&l aggregation
CREATE INDEX IF NOT EXISTS idx_exec_pnl
    ON oms.execution (tenant_id, executed_at DESC)
    WHERE fee IS NOT NULL AND fee != 0;

-- oms.allocation: confirm pending
CREATE INDEX IF NOT EXISTS idx_alloc_pending
    ON oms.allocation (tenant_id, created_at ASC);

-- oms.settlement: pending settlements
CREATE INDEX IF NOT EXISTS idx_settlement_pending
    ON oms.settlement (tenant_id, expected_date ASC);

-- --------------------------------------------------------------------------
-- Update timestamp trigger helper
-- --------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables with updated_at columns
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT quote_ident(nspname) || '.' || quote_ident(relname)
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE relname IN (
            'counterparty','exchange_membership','account','portfolio',
            'security_master','venue',
            'orders','order_slice','order_link',
            'allocation','settlement','position_lots'
        )
        AND nspname IN ('mds','oms')
    LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_updated_at
             BEFORE UPDATE ON %s
             FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()',
            replace(tbl, '.', '_'), tbl
        );
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- --------------------------------------------------------------------------
-- Final FK from orders.status_id to ref (optional, soft)
-- (status transitions are enforced in application logic, not at DB level)
-- --------------------------------------------------------------------------

COMMIT;
