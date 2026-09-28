-- 20261026_006_orm_position_keeping.up.sql
-- position_id FK → oms.position(id) (cross-schema).

CREATE TABLE IF NOT EXISTS orm.position_lot (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    position_id uuid NOT NULL,
    account_id uuid NOT NULL,
    security_id uuid,
    lot_open_date date NOT NULL,
    lot_acquisition_type varchar(20),
    quantity_open numeric(18,4) NOT NULL,
    quantity_closed numeric(18,4) DEFAULT 0 NOT NULL,
    quantity_remaining numeric(18,4) NOT NULL,
    cost_basis_per_unit numeric(18,9),
    cost_basis_total numeric(18,4),
    adjusted_cost_basis_total numeric(18,4),
    tax_lot_method varchar(20),
    is_wash_sale bool DEFAULT false NOT NULL,
    wash_sale_disallowed_loss numeric(18,4),
    is_open bool DEFAULT true NOT NULL,
    closed_date date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT position_lot_pkey PRIMARY KEY (id),
    CONSTRAINT chk_pl_acq CHECK (lot_acquisition_type IS NULL OR lot_acquisition_type IN (
        'PURCHASE','TRANSFER','GIFT','INHERITANCE','SPINOFF','MERGER')),
    CONSTRAINT chk_pl_method CHECK (tax_lot_method IS NULL OR tax_lot_method IN (
        'FIFO','LIFO','HIFO','AVG_COST','SPEC_ID')),
    CONSTRAINT fk_pl_position FOREIGN KEY (position_id) REFERENCES oms.position(id)
);
CREATE INDEX IF NOT EXISTS idx_pl_position ON orm.position_lot (position_id, is_open);
CREATE INDEX IF NOT EXISTS idx_pl_acct_sec ON orm.position_lot (account_id, security_id, is_open);

CREATE TABLE IF NOT EXISTS orm.position_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    security_id uuid NOT NULL,
    as_of_date date NOT NULL,
    opening_qty numeric(18,4),
    closing_qty numeric(18,4) NOT NULL,
    long_short_cd varchar(1),
    market_price numeric(18,9),
    market_value numeric(18,4),
    base_cost numeric(18,4),
    unrealized_pnl numeric(18,4),
    realized_pnl_ytd numeric(18,4),
    income_ytd numeric(18,4),
    currency varchar(3),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT position_history_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ph UNIQUE (tenant_id, account_id, security_id, as_of_date),
    CONSTRAINT chk_ph_ls CHECK (long_short_cd IS NULL OR long_short_cd IN ('L','S'))
);
CREATE INDEX IF NOT EXISTS idx_ph_acct_date ON orm.position_history (account_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_ph_sec ON orm.position_history (security_id, as_of_date);

CREATE TABLE IF NOT EXISTS orm.cash_balance (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    currency varchar(3) NOT NULL,
    as_of_date date NOT NULL,
    as_of_time timestamptz,
    balance_type varchar(20) NOT NULL,
    settled_cash numeric(24,4) DEFAULT 0 NOT NULL,
    unsettled_cash numeric(24,4) DEFAULT 0 NOT NULL,
    projected_cash numeric(24,4),
    pending_deposits numeric(24,4) DEFAULT 0,
    pending_withdrawals numeric(24,4) DEFAULT 0,
    margin_requirement numeric(24,4) DEFAULT 0,
    buying_power numeric(24,4),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT cash_balance_pkey PRIMARY KEY (id),
    CONSTRAINT uq_cb UNIQUE (tenant_id, account_id, currency, as_of_date, balance_type),
    CONSTRAINT chk_cb_type CHECK (balance_type IN (
        'OPENING','CLOSING','INTRADAY','PROJECTED','SETTLED','UNSETTLED','AVAILABLE'))
);
CREATE INDEX IF NOT EXISTS idx_cb_acct ON orm.cash_balance (account_id, as_of_date);
