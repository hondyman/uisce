-- backend/migrations/024_position_master_schema.sql
-- Position Master Gold Copy Schema
-- The derived "Book of Record" where Portfolio, Security, and Pricing converge.
-- Entities: position_master, position_lot_master, cash_position_master,
--           position_snapshot_master, position_gold_trace

-- ============================================================
-- 1. POSITION MASTER (Root holdings entity)
--    Clustered by (portfolio_id, security_id, position_date)
--    Survivorship: Custodian > Accounting > TradingSystem > InternalModel
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.position_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,       -- NULL = this IS the core record
    -- Business cluster key
    portfolio_id             UUID         NOT NULL, -- FK → portfolio_master
    security_id              UUID         NOT NULL, -- FK → security_master
    position_date            DATE         NOT NULL,
    -- Position details
    position_quantity        NUMERIC(28,8) NOT NULL, -- signed: + long, - short
    position_side            VARCHAR(10)  NOT NULL DEFAULT 'Long'
                                 CHECK (position_side IN ('Long','Short','Net')),
    position_currency        VARCHAR(3)   NOT NULL,  -- ISO-4217
    -- Valuation
    price_id                 UUID,        -- FK → price_master (used for this valuation)
    market_value_local       NUMERIC(28,8),
    market_value_base        NUMERIC(28,8),
    valuation_fx_rate        NUMERIC(18,8) DEFAULT 1.0,
    -- Cost & P&L
    cost_basis_local         NUMERIC(28,8),
    unrealized_pl_local      NUMERIC(28,8), -- market_value_local - cost_basis_local
    unrealized_pl_pct        NUMERIC(10,6), -- pct gain/loss
    -- Portfolio weight
    position_weight_pct      NUMERIC(10,6),
    -- Provenance & quality
    position_source          VARCHAR(100) NOT NULL DEFAULT 'Custodian'
                                 CHECK (position_source IN ('Custodian','Accounting','TradingSystem','InternalModel','Manual')),
    position_confidence      INT          NOT NULL DEFAULT 80
                                 CHECK (position_confidence BETWEEN 0 AND 100),
    -- Reconciliation
    is_reconciled            BOOLEAN      NOT NULL DEFAULT false,
    reconciliation_diff      NUMERIC(28,8),         -- vs. custodian quantity
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit / bi-temporal
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_position_master_cluster
        UNIQUE (portfolio_id, security_id, position_date, position_source, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_position_master_tenant      ON edm.position_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_position_master_portfolio   ON edm.position_master(portfolio_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_position_master_security    ON edm.position_master(security_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_position_master_date        ON edm.position_master(position_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_position_master_valid       ON edm.position_master(valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_position_master_unrec       ON edm.position_master(is_reconciled) WHERE is_reconciled = false;

ALTER TABLE edm.position_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS position_master_tenant_isolation ON edm.position_master;
CREATE POLICY position_master_tenant_isolation ON edm.position_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 2. POSITION LOT MASTER (Tax lot tracking)
--    Linked to position_master. Supports FIFO, LIFO, HIFO, Specific.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.position_lot_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Link to parent position
    position_id              UUID         NOT NULL REFERENCES edm.position_master(id) ON DELETE CASCADE,
    -- Lot identity
    lot_reference            VARCHAR(100),          -- source system lot ID
    acquisition_date         DATE         NOT NULL,
    settlement_date          DATE,
    -- Lot quantities & cost
    lot_quantity             NUMERIC(28,8) NOT NULL CHECK (lot_quantity > 0),
    cost_per_unit            NUMERIC(28,10) NOT NULL CHECK (cost_per_unit >= 0),
    total_cost_basis         NUMERIC(28,8) NOT NULL, -- lot_quantity * cost_per_unit
    -- Method
    lot_method               VARCHAR(20)  NOT NULL DEFAULT 'FIFO'
                                 CHECK (lot_method IN ('FIFO','LIFO','HIFO','Specific','AverageCost')),
    -- Status
    is_closed                BOOLEAN      NOT NULL DEFAULT false,
    closed_date              DATE,
    realized_pl              NUMERIC(28,8),          -- filled on close
    -- Audit
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_position_lot_position    ON edm.position_lot_master(position_id);
CREATE INDEX IF NOT EXISTS idx_position_lot_tenant      ON edm.position_lot_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_position_lot_acq_date    ON edm.position_lot_master(acquisition_date);
CREATE INDEX IF NOT EXISTS idx_position_lot_open        ON edm.position_lot_master(is_closed) WHERE is_closed = false;

ALTER TABLE edm.position_lot_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS position_lot_tenant_isolation ON edm.position_lot_master;
CREATE POLICY position_lot_tenant_isolation ON edm.position_lot_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 3. CASH POSITION MASTER
--    Cash balances by portfolio, currency, and value date.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.cash_position_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Cluster key
    portfolio_id             UUID         NOT NULL,
    cash_currency            VARCHAR(3)   NOT NULL, -- ISO-4217
    account_id               UUID,                  -- custodian account
    value_date               DATE         NOT NULL,
    -- Balances
    balance_amount           NUMERIC(28,8) NOT NULL,
    available_balance        NUMERIC(28,8),          -- balance_amount - pending settlements
    pending_settlements      JSONB        NOT NULL DEFAULT '[]',
    interest_accrued         NUMERIC(28,8) NOT NULL DEFAULT 0,
    -- Provenance
    cash_source              VARCHAR(100) NOT NULL DEFAULT 'Custodian',
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_cash_position_cluster
        UNIQUE (portfolio_id, cash_currency, value_date, cash_source, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_cash_position_tenant     ON edm.cash_position_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cash_position_portfolio  ON edm.cash_position_master(portfolio_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_cash_position_date       ON edm.cash_position_master(value_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_cash_position_valid      ON edm.cash_position_master(valid_from, valid_to);

ALTER TABLE edm.cash_position_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS cash_position_tenant_isolation ON edm.cash_position_master;
CREATE POLICY cash_position_tenant_isolation ON edm.cash_position_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 4. POSITION SNAPSHOT MASTER
--    Append-only historical snapshots for time-series analysis.
--    Created automatically on each gold copy run.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.position_snapshot_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    position_id              UUID         NOT NULL REFERENCES edm.position_master(id) ON DELETE CASCADE,
    snapshot_date            DATE         NOT NULL,
    -- Snapshot values (point-in-time copies)
    snapshot_quantity        NUMERIC(28,8),
    snapshot_market_value    NUMERIC(28,8),
    snapshot_price_used      NUMERIC(28,10),
    snapshot_fx_rate         NUMERIC(18,8),
    portfolio_composition    JSONB,          -- JSON snapshot of portfolio weights at this date
    snapshot_source          VARCHAR(100)   DEFAULT 'GoldCopyRun',
    created_at               TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pos_snapshot_position   ON edm.position_snapshot_master(position_id);
CREATE INDEX IF NOT EXISTS idx_pos_snapshot_date       ON edm.position_snapshot_master(snapshot_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_pos_snapshot_tenant     ON edm.position_snapshot_master(tenant_id);
-- Prevent duplicate snapshots per position per date
CREATE UNIQUE INDEX IF NOT EXISTS uq_pos_snapshot_position_date
    ON edm.position_snapshot_master(position_id, snapshot_date);

ALTER TABLE edm.position_snapshot_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS position_snapshot_tenant_isolation ON edm.position_snapshot_master;
CREATE POLICY position_snapshot_tenant_isolation ON edm.position_snapshot_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 5. POSITION GOLD TRACE
--    Per-field lineage for every Position survivorship run.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.position_gold_trace (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    entity_type              VARCHAR(50)  NOT NULL DEFAULT 'position'
                                 CHECK (entity_type IN ('position','lot','cash','snapshot')),
    entity_id                UUID         NOT NULL,
    run_id                   UUID         NOT NULL,
    field_name               VARCHAR(100) NOT NULL,
    chosen_value             TEXT,
    chosen_source            VARCHAR(100),
    survivorship_rule        VARCHAR(100),
    rejected_sources         JSONB        NOT NULL DEFAULT '[]',
    dq_rules_passed          TEXT[]       NOT NULL DEFAULT '{}',
    dq_rules_failed          TEXT[]       NOT NULL DEFAULT '{}',
    confidence_contribution  INT          NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pos_gold_trace_entity   ON edm.position_gold_trace(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_pos_gold_trace_run      ON edm.position_gold_trace(run_id);
CREATE INDEX IF NOT EXISTS idx_pos_gold_trace_tenant   ON edm.position_gold_trace(tenant_id);

ALTER TABLE edm.position_gold_trace ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS position_gold_trace_isolation ON edm.position_gold_trace;
CREATE POLICY position_gold_trace_isolation ON edm.position_gold_trace
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));
