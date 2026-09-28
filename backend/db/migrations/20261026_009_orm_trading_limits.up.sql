-- 20261026_009_orm_trading_limits.up.sql

CREATE TABLE IF NOT EXISTS orm.trading_limit (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    limit_type varchar(30) NOT NULL,
    scope_type varchar(30) NOT NULL,
    scope_id uuid,
    scope_reference_cd varchar(50),
    limit_value numeric(24,6) NOT NULL,
    limit_unit varchar(20),
    currency varchar(3),
    warning_threshold_pct numeric(7,4),
    hard_breach bool DEFAULT true NOT NULL,
    requires_override bool DEFAULT false NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    approved_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT trading_limit_pkey PRIMARY KEY (id),
    CONSTRAINT chk_tl_type CHECK (limit_type IN (
        'TRADER_ORDER_SIZE','TRADER_NOTIONAL','STRATEGY_POSITION',
        'ACCOUNT_CONCENTRATION','ASSET_CLASS_EXPOSURE','CURRENCY_EXPOSURE',
        'VENUE_CONCENTRATION','BROKER_CONCENTRATION','DAILY_TRADE_COUNT',
        'DAILY_NOTIONAL','FAT_FINGER_PRICE','FAT_FINGER_QUANTITY')),
    CONSTRAINT chk_tl_scope CHECK (scope_type IN (
        'GLOBAL','TENANT','MANAGER','TRADER','STRATEGY','ACCOUNT','SECURITY',
        'ASSET_CLASS','CURRENCY','VENUE','BROKER')),
    CONSTRAINT chk_tl_unit CHECK (limit_unit IS NULL OR limit_unit IN (
        'ABSOLUTE','NOTIONAL','PERCENTAGE','BPS','COUNT','SHARES'))
);
CREATE INDEX IF NOT EXISTS idx_tl_scope ON orm.trading_limit (scope_type, scope_id, limit_type);
CREATE INDEX IF NOT EXISTS idx_tl_tenant ON orm.trading_limit (tenant_id);
