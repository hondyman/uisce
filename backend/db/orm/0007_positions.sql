-- ============================================================================
-- 0007_positions.sql
-- Position lots table: lot-level record of every fill that opens/closes a
-- position.  Current position = SUM(qty_signed) per (account, security).
-- Supports FIFO / LIFO / average-cost cost-basis methods.
-- Run after 0006_oms_fills.sql
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- oms.position_lots
-- One row per fill that changes a position (open or close).
-- qty_signed: positive = long, negative = short.
-- --------------------------------------------------------------------------
CREATE TABLE oms.position_lots (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',

    account_id          UUID        NOT NULL REFERENCES mds.account(id),
    security_id         UUID        NOT NULL REFERENCES mds.security_master(id),

    -- lot identification
    source_execution_id UUID        NOT NULL REFERENCES oms.execution(id),
    source_order_id     UUID        REFERENCES oms.orders(id),

    -- quantity
    qty_signed          NUMERIC(20,6) NOT NULL,  -- +long / -short
    remaining_qty       NUMERIC(20,6) NOT NULL DEFAULT 0,  -- not yet closed by subsequent fills

    -- cost basis (in account base currency)
    cost_basis_ccy      CHAR(3)     REFERENCES ref.currency(iso3_code),
    cost_basis_amount   NUMERIC(20,4),
    cost_basis_per_unit NUMERIC(20,8),

    -- lot metadata
    open_date           DATE        NOT NULL,
    open_price          NUMERIC(20,8),
    close_date          DATE,
    close_price         NUMERIC(20,8),

    -- FIFO/LIFO tracking
    lot_sequence        INTEGER     NOT NULL,  -- monotonically increasing per account+security
    cost_method         VARCHAR(10) DEFAULT 'FIFO',  -- 'FIFO','LIFO','AVCO'

    -- unrealised PnL (recalculated periodically)
    unrealised_pnl_ccy  CHAR(3)    REFERENCES ref.currency(iso3_code),
    unrealised_pnl      NUMERIC(20,4),

    is_open             BOOLEAN     NOT NULL DEFAULT true,
    is_active           BOOLEAN     NOT NULL DEFAULT true,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_lot_sequence UNIQUE (account_id, security_id, lot_sequence)
);

CREATE INDEX idx_lots_account   ON oms.position_lots (account_id);
CREATE INDEX idx_lots_security  ON oms.position_lots (security_id);
CREATE INDEX idx_lots_tenant    ON oms.position_lots (tenant_id);
CREATE INDEX idx_lots_is_open  ON oms.position_lots (account_id, security_id) WHERE is_open = true;
CREATE INDEX idx_lots_exec     ON oms.position_lots (source_execution_id);

-- --------------------------------------------------------------------------
-- Materialized view: current position snapshot per account × security
-- Refreshed on-demand via: REFRESH MATERIALIZED VIEW oms.current_positions;
-- --------------------------------------------------------------------------
CREATE MATERIALIZED VIEW oms.current_positions AS
SELECT
    tenant_id,
    account_id,
    security_id,
    SUM(qty_signed)          AS net_qty,
    SUM(cost_basis_amount)   AS total_cost_basis,
    CASE WHEN SUM(qty_signed) != 0
         THEN SUM(cost_basis_amount) / SUM(qty_signed)
         ELSE 0
    END                      AS avg_cost_per_unit,
    MIN(open_date)           AS first_open_date,
    MAX(open_date)           AS last_open_date,
    BOOL_OR(is_open)         AS has_open_lot
FROM oms.position_lots
WHERE is_active = true
GROUP BY tenant_id, account_id, security_id;

CREATE UNIQUE INDEX idx_current_pos_key
    ON oms.current_positions (tenant_id, account_id, security_id);

COMMENT ON MATERIALIZED VIEW oms.current_positions IS
    'Current net position per account × security. Refresh: REFRESH MATERIALIZED VIEW oms.current_positions;';

COMMIT;
