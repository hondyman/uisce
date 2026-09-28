-- 20261026_014_orm_fx_exposure.up.sql

CREATE TABLE IF NOT EXISTS orm.fx_exposure (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    portfolio_id uuid,
    currency varchar(3) NOT NULL,
    as_of_date date NOT NULL,
    gross_exposure numeric(24,4),
    net_exposure numeric(24,4),
    hedged_exposure numeric(24,4),
    unhedged_exposure numeric(24,4),
    hedge_ratio_pct numeric(7,4),
    base_currency varchar(3),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fx_exposure_pkey PRIMARY KEY (id),
    CONSTRAINT uq_fxe UNIQUE (tenant_id, account_id, currency, as_of_date)
);
CREATE INDEX IF NOT EXISTS idx_fxe_acct ON orm.fx_exposure (account_id, as_of_date);

CREATE TABLE IF NOT EXISTS orm.fx_hedge (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    portfolio_id uuid,
    currency varchar(3) NOT NULL,
    hedge_instrument_type varchar(30),
    hedge_instrument_id uuid,
    notional_amount numeric(24,4) NOT NULL,
    notional_currency varchar(3) NOT NULL,
    hedge_ratio_pct numeric(7,4),
    effective_date date NOT NULL,
    maturity_date date,
    is_active bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT fx_hedge_pkey PRIMARY KEY (id),
    CONSTRAINT chk_fxh_type CHECK (hedge_instrument_type IS NULL OR hedge_instrument_type IN (
        'FORWARD','FUTURE','OPTION','SWAP','NDF'))
);
CREATE INDEX IF NOT EXISTS idx_fxh_acct ON orm.fx_hedge (account_id, is_active);
