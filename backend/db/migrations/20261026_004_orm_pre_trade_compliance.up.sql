-- 20261026_004_orm_pre_trade_compliance.up.sql
-- rule_id left as plain UUID; no orm.rule exists.

CREATE TABLE IF NOT EXISTS orm.pre_trade_check (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    check_time timestamptz NOT NULL,
    check_type varchar(50) NOT NULL,
    check_status varchar(20) NOT NULL,
    rule_id uuid,
    rule_cd varchar(50),
    breach_amount numeric(24,6),
    limit_amount numeric(24,6),
    breach_pct numeric(7,4),
    details jsonb,
    is_blocking bool DEFAULT false NOT NULL,
    override_reason varchar(500),
    overridden_by uuid,
    overridden_at timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT pre_trade_check_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ptc_status CHECK (check_status IN (
        'PASS','WARN','FAIL','BYPASS','ERROR')),
    CONSTRAINT chk_ptc_type CHECK (check_type IN (
        'RESTRICTED_LIST','POSITION_LIMIT','CONCENTRATION','MANDATE',
        'SHORT_SELL_LOCATE','REG_T','FAT_FINGER','LEVERAGE','LIQUIDITY',
        'CASH_AVAILABILITY','PRICE_TOLERANCE','COUNTERPARTY_LIMIT',
        'WASH_SALE','RESTRICTED_SECURITY')),
    CONSTRAINT fk_ptc_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ptc_order ON orm.pre_trade_check (order_id, check_time);
CREATE INDEX IF NOT EXISTS idx_ptc_status ON orm.pre_trade_check (check_status, check_type);

CREATE TABLE IF NOT EXISTS orm.restricted_list (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid,
    portfolio_id uuid,
    security_id uuid,
    issuer_id uuid,
    restriction_type varchar(30) NOT NULL,
    restriction_action varchar(30) NOT NULL,
    reason varchar(500),
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    is_active bool DEFAULT true NOT NULL,
    added_by uuid,
    approved_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT restricted_list_pkey PRIMARY KEY (id),
    CONSTRAINT chk_rl_type CHECK (restriction_type IN (
        'TRADING_BLACKOUT','RESTRICTED_LIST','WATCH_LIST','GREY_LIST',
        'MNPI','SECTION_16','INSIDER','REGULATORY','CLIENT_DIRECTED')),
    CONSTRAINT chk_rl_action CHECK (restriction_action IN (
        'PROHIBIT','REQUIRE_APPROVAL','LIMIT','MONITOR','PROHIBIT_BUY',
        'PROHIBIT_SELL','PROHIBIT_SHORT'))
);
CREATE INDEX IF NOT EXISTS idx_rl_security ON orm.restricted_list (security_id, is_active) WHERE security_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rl_account ON orm.restricted_list (account_id, is_active) WHERE account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rl_tenant ON orm.restricted_list (tenant_id);
