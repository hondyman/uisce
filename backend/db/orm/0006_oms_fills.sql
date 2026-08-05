-- ============================================================================
-- 0006_oms_fills.sql
-- Execution, allocation, settlement.
-- Run after 0005_oms_events.sql
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- oms.execution
-- Fills / executions as reported by the venue / gateway.
-- One row per venue print. Immutable.
-- --------------------------------------------------------------------------
CREATE TABLE oms.execution (
    id                      UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id               UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',

    -- linkage to order / slice
    order_id                UUID        NOT NULL REFERENCES oms.orders(id),
    slice_id                UUID        REFERENCES oms.order_slice(id),

    -- venue identifiers
    venue_execution_id      VARCHAR(100) NOT NULL UNIQUE,  -- venue's unique print ID
    venue_report_id         VARCHAR(100),                  -- secondary venue ref

    -- fill details
    qty                     NUMERIC(20,6) NOT NULL,
    price                   NUMERIC(20,8) NOT NULL,
    gross_amount            NUMERIC(20,2) NOT NULL,
    fee                     NUMERIC(12,4) DEFAULT 0,
    fee_ccy                 CHAR(3)      REFERENCES ref.currency(iso3_code),
    net_amount              NUMERIC(20,2) NOT NULL,

    -- pricing references
    price_currency_id       INTEGER      REFERENCES ref.currency(id),
    price_type              VARCHAR(20),   -- 'LAST','OPEN','CLOSE','VWAP','NAVW','MID'
    price_override_flag     BOOLEAN     DEFAULT false,

    -- venue metadata
    venue_id                UUID        REFERENCES mds.venue(id),
    liquidity_flag_id       INTEGER     REFERENCES ref.liquidity_flag(id),

    -- settlement
    currency_id             INTEGER      REFERENCES ref.currency(id),
    settlement_ccy          CHAR(3)      REFERENCES ref.currency(iso3_code),
    settlement_date         DATE,
    settlement_amount       NUMERIC(20,2),

    -- timing
    executed_at             TIMESTAMPTZ NOT NULL,  -- venue-reported execution time
    reported_at             TIMESTAMPTZ NOT NULL DEFAULT now(),  -- when venue told us

    -- cross-DB reference
    external_execution_id    TEXT,        -- e.g. broker's execution ID

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_execution_order    ON oms.execution (order_id, executed_at DESC);
CREATE INDEX idx_execution_slice    ON oms.execution (slice_id) WHERE slice_id IS NOT NULL;
CREATE INDEX idx_execution_venue   ON oms.execution (venue_id);
CREATE INDEX idx_execution_tenant  ON oms.execution (tenant_id, executed_at DESC);
CREATE INDEX idx_execution_venue_eid ON oms.execution (venue_execution_id);
CREATE INDEX idx_execution_ext     ON oms.execution (tenant_id, external_execution_id) WHERE external_execution_id IS NOT NULL;

-- --------------------------------------------------------------------------
-- oms.allocation
-- Allocation of an execution to destination accounts (post-trade).
-- --------------------------------------------------------------------------
CREATE TABLE oms.allocation (
    id                    UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id             UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',

    execution_id          UUID        NOT NULL REFERENCES oms.execution(id),
    account_id            UUID        NOT NULL REFERENCES mds.account(id),

    qty                   NUMERIC(20,6) NOT NULL,
    price                 NUMERIC(20,8) NOT NULL,
    net_amount            NUMERIC(20,2) NOT NULL,
    currency_id           INTEGER     REFERENCES ref.currency(id),

    status_id             INTEGER,

    allocated_by          VARCHAR(100),
    allocated_at          TIMESTAMPTZ DEFAULT now(),
    confirmed_at          TIMESTAMPTZ,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_alloc_execution   ON oms.allocation (execution_id);
CREATE INDEX idx_alloc_account     ON oms.allocation (account_id);
CREATE INDEX idx_alloc_status      ON oms.allocation (status_id);
CREATE INDEX idx_alloc_tenant      ON oms.allocation (tenant_id);

-- --------------------------------------------------------------------------
-- oms.settlement
-- Settlement record per allocation or execution.
-- --------------------------------------------------------------------------
CREATE TABLE oms.settlement (
    id                    UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id             UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',

    execution_id          UUID        REFERENCES oms.execution(id),
    allocation_id         UUID        REFERENCES oms.allocation(id),

    counterparty_id       UUID        REFERENCES mds.counterparty(id),

    -- instruction details
    settlement_type       VARCHAR(20),  -- 'DVP','FOP','RVP','DDA'
    deliver_ccy          CHAR(3)     REFERENCES ref.currency(iso3_code),
    deliver_amount        NUMERIC(20,2),
    receive_ccy           CHAR(3)     REFERENCES ref.currency(iso3_code),
    receive_amount        NUMERIC(20,2),

    -- CCP / custodians
    ccp_name              VARCHAR(50),
    custodian_account     VARCHAR(50),
    custodian_ref         VARCHAR(100),

    -- dates
    expected_date         DATE,
    actual_date           DATE,

    -- status
    status_id             INTEGER,

    instructions          JSONB       DEFAULT '{}',  -- full settlement instruction blob
    failure_reason        TEXT,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_settlement_execution  ON oms.settlement (execution_id) WHERE execution_id IS NOT NULL;
CREATE INDEX idx_settlement_alloc      ON oms.settlement (allocation_id) WHERE allocation_id IS NOT NULL;
CREATE INDEX idx_settlement_cp        ON oms.settlement (counterparty_id) WHERE counterparty_id IS NOT NULL;
CREATE INDEX idx_settlement_status   ON oms.settlement (status_id);
CREATE INDEX idx_settlement_expected ON oms.settlement (expected_date) WHERE expected_date IS NOT NULL;
CREATE INDEX idx_settlement_tenant   ON oms.settlement (tenant_id);

-- Attach default status triggers (function defined in 0004_oms_orders.sql)
CREATE TRIGGER trg_allocation_default_status
    BEFORE INSERT ON oms.allocation
    FOR EACH ROW EXECUTE FUNCTION oms.default_order_status();

CREATE TRIGGER trg_settlement_default_status
    BEFORE INSERT ON oms.settlement
    FOR EACH ROW EXECUTE FUNCTION oms.default_order_status();

COMMIT;
