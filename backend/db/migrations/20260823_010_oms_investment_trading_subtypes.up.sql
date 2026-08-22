-- 20260823_010_oms_investment_trading_subtypes.up.sql
-- Single-Table Inheritance for OMS Accounts, Positions, Securities, and Trade Orders

-- 1. OMS ACCOUNT
CREATE TABLE IF NOT EXISTS oms.account (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_number TEXT NOT NULL,
    account_name TEXT NOT NULL,
    base_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    subtype_code TEXT NOT NULL,

    -- Subtype-Specific Fields (Nullable)
    sponsor_id UUID,
    mandate_type TEXT,
    erisa_flag BOOLEAN,
    fee_schedule_code TEXT,
    tax_id_type TEXT,
    citizenship VARCHAR(2),
    margin_agreement_flag BOOLEAN,
    accredited_investor_status BOOLEAN,
    sponsor_firm TEXT,
    model_strategy_id UUID,
    overlay_manager_id UUID,
    rebalance_frequency TEXT,
    trust_type TEXT,
    grantor_name TEXT,
    trustee_signatory_id UUID,
    dissolution_date DATE,
    plan_type TEXT,
    vesting_schedule_code TEXT,
    rmd_eligible_flag BOOLEAN,
    custodian_bank_id UUID,
    corporate_entity_id UUID,
    treasury_signatory_group TEXT,
    wire_limit_daily NUMERIC(18,2),

    -- Bitemporal & Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,

    CONSTRAINT uq_oms_account_number UNIQUE (tenant_id, account_number),
    CONSTRAINT ck_oms_account_subtype CHECK (
        subtype_code IN ('institutional', 'retail_wealth', 'sma', 'trust_estate', 'qualified_retirement', 'corporate_treasury')
    )
);

CREATE INDEX IF NOT EXISTS idx_account_inst ON oms.account (tenant_id) WHERE subtype_code = 'institutional';
CREATE INDEX IF NOT EXISTS idx_account_retail ON oms.account (tenant_id) WHERE subtype_code = 'retail_wealth';
CREATE INDEX IF NOT EXISTS idx_account_sma ON oms.account (tenant_id) WHERE subtype_code = 'sma';
CREATE INDEX IF NOT EXISTS idx_account_trust ON oms.account (tenant_id) WHERE subtype_code = 'trust_estate';
CREATE INDEX IF NOT EXISTS idx_account_qualret ON oms.account (tenant_id) WHERE subtype_code = 'qualified_retirement';
CREATE INDEX IF NOT EXISTS idx_account_treasury ON oms.account (tenant_id) WHERE subtype_code = 'corporate_treasury';

-- 2. OMS POSITION
CREATE TABLE IF NOT EXISTS oms.position (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,  -- FK resolved after oms.account exists; soft-ref
    security_id UUID NOT NULL,
    quantity NUMERIC(28,8) NOT NULL,
    market_value NUMERIC(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    subtype_code TEXT NOT NULL,

    -- Subtype Fields
    custody_account_id UUID,
    settled_shares NUMERIC(28,8),
    cost_basis_method TEXT,
    held_to_maturity_flag BOOLEAN,
    prime_broker_id UUID,
    borrow_rate_bps NUMERIC(10,4),
    locate_id TEXT,
    hard_to_borrow_flag BOOLEAN,
    underlying_security_id UUID,
    notional_amount NUMERIC(18,2),
    unrealized_pnl NUMERIC(18,2),
    expiration_date DATE,
    pledged_to_party TEXT,
    haircut_pct NUMERIC(6,4),
    rehypothecation_allowed_flag BOOLEAN,
    trade_date_shares NUMERIC(28,8),
    pending_settlement_cash NUMERIC(18,2),
    fails_to_deliver_flag BOOLEAN,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,

    CONSTRAINT ck_oms_position_subtype CHECK (
        subtype_code IN ('settled_long', 'short_borrowed', 'derivative_exposure', 'pledged_collateral', 'unsettled_pipeline')
    )
);

CREATE INDEX IF NOT EXISTS idx_pos_settled ON oms.position (tenant_id, account_id) WHERE subtype_code = 'settled_long';
CREATE INDEX IF NOT EXISTS idx_pos_short ON oms.position (tenant_id, account_id) WHERE subtype_code = 'short_borrowed';
CREATE INDEX IF NOT EXISTS idx_pos_deriv ON oms.position (tenant_id, account_id) WHERE subtype_code = 'derivative_exposure';
CREATE INDEX IF NOT EXISTS idx_pos_pledged ON oms.position (tenant_id, account_id) WHERE subtype_code = 'pledged_collateral';
CREATE INDEX IF NOT EXISTS idx_pos_unsettled ON oms.position (tenant_id, account_id) WHERE subtype_code = 'unsettled_pipeline';

-- 3. OMS SECURITY
CREATE TABLE IF NOT EXISTS oms.security (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    security_name TEXT NOT NULL,
    identifier_type TEXT NOT NULL,
    identifier_value TEXT NOT NULL,
    subtype_code TEXT NOT NULL,

    -- Subtype Fields
    ticker TEXT,
    isin VARCHAR(12),
    voting_rights_type TEXT,
    dividend_currency VARCHAR(3),
    coupon_rate NUMERIC(8,4),
    maturity_date DATE,
    day_count_convention TEXT,
    inflation_protected_flag BOOLEAN,
    credit_rating_sp TEXT,
    call_date DATE,
    conversion_ratio NUMERIC(12,4),
    seniority_level TEXT,
    pool_number TEXT,
    factor_current NUMERIC(10,8),
    prepayment_speed_cpr NUMERIC(6,3),
    tranche_tier TEXT,
    contract_size NUMERIC(18,4),
    strike_price NUMERIC(18,4),
    put_call_indicator VARCHAR(4),
    exchange_mic VARCHAR(4),
    isda_agreement_id UUID,
    fixed_rate NUMERIC(8,4),
    floating_index_name TEXT,
    counterparty_lei VARCHAR(20),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,

    CONSTRAINT uq_oms_security_id UNIQUE (tenant_id, identifier_type, identifier_value),
    CONSTRAINT ck_oms_security_subtype CHECK (
        subtype_code IN ('equity', 'sovereign_debt', 'corporate_debt', 'structured_abs_mbs', 'etd_derivative', 'otc_derivative')
    )
);

CREATE INDEX IF NOT EXISTS idx_sec_equity ON oms.security (tenant_id) WHERE subtype_code = 'equity';
CREATE INDEX IF NOT EXISTS idx_sec_debt ON oms.security (tenant_id) WHERE subtype_code IN ('sovereign_debt', 'corporate_debt');
CREATE INDEX IF NOT EXISTS idx_sec_structured ON oms.security (tenant_id) WHERE subtype_code = 'structured_abs_mbs';
CREATE INDEX IF NOT EXISTS idx_sec_etd ON oms.security (tenant_id) WHERE subtype_code = 'etd_derivative';
CREATE INDEX IF NOT EXISTS idx_sec_otc ON oms.security (tenant_id) WHERE subtype_code = 'otc_derivative';

-- 4. OMS TRADE ORDER
CREATE TABLE IF NOT EXISTS oms.trade_order (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    security_id UUID NOT NULL,
    order_side TEXT NOT NULL,
    ordered_quantity NUMERIC(28,8) NOT NULL,
    execution_price NUMERIC(18,4),
    order_status TEXT NOT NULL DEFAULT 'NEW',
    subtype_code TEXT NOT NULL,

    -- Subtype Fields
    allocation_profile_id UUID,
    total_requested_quantity NUMERIC(28,8),
    average_price NUMERIC(18,4),
    execution_algo_id TEXT,
    venue_id TEXT,
    liquidity_flag TEXT,
    route_time_micros BIGINT,
    counterparty_dealer_id UUID,
    confirmation_status TEXT,
    isda_schedule_version TEXT,
    base_currency VARCHAR(3),
    quote_currency VARCHAR(3),
    fx_rate NUMERIC(18,8),
    value_date DATE,
    syndicate_manager_id UUID,
    concession_amount NUMERIC(18,2),
    allotment_shares NUMERIC(28,8),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,

    CONSTRAINT ck_oms_trade_subtype CHECK (
        subtype_code IN ('block_parent', 'dma_execution', 'otc_bilateral', 'fx_spot_forward', 'primary_auction')
    )
);

CREATE INDEX IF NOT EXISTS idx_trade_block ON oms.trade_order (tenant_id, account_id) WHERE subtype_code = 'block_parent';
CREATE INDEX IF NOT EXISTS idx_trade_dma ON oms.trade_order (tenant_id, account_id) WHERE subtype_code = 'dma_execution';
CREATE INDEX IF NOT EXISTS idx_trade_otc ON oms.trade_order (tenant_id, account_id) WHERE subtype_code = 'otc_bilateral';
CREATE INDEX IF NOT EXISTS idx_trade_fx ON oms.trade_order (tenant_id, account_id) WHERE subtype_code = 'fx_spot_forward';
CREATE INDEX IF NOT EXISTS idx_trade_auction ON oms.trade_order (tenant_id, account_id) WHERE subtype_code = 'primary_auction';
