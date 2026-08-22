-- 20260823_011_oms_alternatives_and_cash_flow_subtypes.up.sql
-- STI Tables for Alternative Investments and Cash Settlements

-- 1. ALTERNATIVE INVESTMENT MASTER
CREATE TABLE IF NOT EXISTS altinv.alternative_investment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    investment_name TEXT NOT NULL,
    sponsor_name TEXT NOT NULL,
    asset_class TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    subtype_code TEXT NOT NULL,

    -- Subtype Fields
    vintage_year INT,
    committed_capital NUMERIC(18,2),
    called_capital NUMERIC(18,2),
    unfunded_commitment NUMERIC(18,2),
    dpi NUMERIC(8,4),
    rvpi NUMERIC(8,4),
    round_series TEXT,
    pro_rata_rights_flag BOOLEAN,
    lead_investor_name TEXT,
    post_money_valuation NUMERIC(18,2),
    property_type TEXT,
    occupancy_rate_pct NUMERIC(6,3),
    gross_asset_value NUMERIC(18,2),
    loan_to_value_pct NUMERIC(6,3),
    sofr_spread_bps NUMERIC(10,4),
    pik_interest_pct NUMERIC(6,3),
    warrant_coverage_pct NUMERIC(6,3),
    covenant_type TEXT,
    hurdle_rate_pct NUMERIC(6,3),
    high_water_mark_nav NUMERIC(18,2),
    lockup_period_months INT,
    redemption_notice_days INT,
    project_phase TEXT,
    concession_expiry_year INT,
    esg_carbon_offset_tons NUMERIC(12,2),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,

    CONSTRAINT ck_altinv_subtype CHECK (
        subtype_code IN ('private_equity', 'venture_capital', 'real_estate', 'private_credit', 'hedge_fund', 'infrastructure')
    )
);

CREATE INDEX IF NOT EXISTS idx_altinv_pe ON altinv.alternative_investment (tenant_id) WHERE subtype_code = 'private_equity';
CREATE INDEX IF NOT EXISTS idx_altinv_vc ON altinv.alternative_investment (tenant_id) WHERE subtype_code = 'venture_capital';
CREATE INDEX IF NOT EXISTS idx_altinv_re ON altinv.alternative_investment (tenant_id) WHERE subtype_code = 'real_estate';
CREATE INDEX IF NOT EXISTS idx_altinv_credit ON altinv.alternative_investment (tenant_id) WHERE subtype_code = 'private_credit';
CREATE INDEX IF NOT EXISTS idx_altinv_hf ON altinv.alternative_investment (tenant_id) WHERE subtype_code = 'hedge_fund';
CREATE INDEX IF NOT EXISTS idx_altinv_infra ON altinv.alternative_investment (tenant_id) WHERE subtype_code = 'infrastructure';

-- 2. CASH FLOW & SETTLEMENT
CREATE TABLE IF NOT EXISTS cash_flow.settlement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    amount NUMERIC(18,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    settlement_date DATE NOT NULL,
    settlement_status TEXT NOT NULL DEFAULT 'SETTLED',
    subtype_code TEXT NOT NULL,

    -- Subtype Fields
    ex_date DATE,
    record_date DATE,
    drip_reinvest_flag BOOLEAN,
    tax_withholding_amount NUMERIC(18,2),
    coupon_period_start DATE,
    accrued_interest NUMERIC(18,2),
    payment_frequency TEXT,
    call_notice_id TEXT,
    due_date DATE,
    management_fee_portion NUMERIC(18,2),
    investment_portion NUMERIC(18,2),
    return_of_capital NUMERIC(18,2),
    preferred_return NUMERIC(18,2),
    carried_interest_retained NUMERIC(18,2),
    action_type_code TEXT,
    cash_in_lieu_amount NUMERIC(18,2),
    mandatory_flag BOOLEAN,
    fee_category TEXT,
    invoice_reference_id TEXT,
    vat_amount NUMERIC(18,2),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,

    CONSTRAINT ck_cash_flow_subtype CHECK (
        subtype_code IN ('dividend', 'coupon_fixed_income', 'capital_call', 'lp_distribution', 'corporate_action', 'expense_fee')
    )
);

CREATE INDEX IF NOT EXISTS idx_cash_div ON cash_flow.settlement (tenant_id, account_id) WHERE subtype_code = 'dividend';
CREATE INDEX IF NOT EXISTS idx_cash_coupon ON cash_flow.settlement (tenant_id, account_id) WHERE subtype_code = 'coupon_fixed_income';
CREATE INDEX IF NOT EXISTS idx_cash_call ON cash_flow.settlement (tenant_id, account_id) WHERE subtype_code = 'capital_call';
CREATE INDEX IF NOT EXISTS idx_cash_lp ON cash_flow.settlement (tenant_id, account_id) WHERE subtype_code = 'lp_distribution';
CREATE INDEX IF NOT EXISTS idx_cash_action ON cash_flow.settlement (tenant_id, account_id) WHERE subtype_code = 'corporate_action';
CREATE INDEX IF NOT EXISTS idx_cash_expense ON cash_flow.settlement (tenant_id, account_id) WHERE subtype_code = 'expense_fee';
