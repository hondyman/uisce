-- 0001_orm_schema.up.sql
--
-- The ORM (crims) order/trading schema, as it exists in a tenant's own
-- database (ADR-030, ADR-042). Applied by internal/migrations.TenantRunner to
-- every database provisioned with App = "orm".
--
-- Derived from the migration set, which is the schema authority (ADR-024):
--   migrations/20260909_create_local_orm_schema.sql  (order, placement,
--     order_allocation, execution, execution_allocation, account)
--   migrations/20260910_create_orm_broker.sql         (broker)
--   db/migrations/20261026_002..014                    (the 20261026 batch)
-- with the tenant_id of 20261017_001 applied, the three `oms` foreign keys
-- dropped, and no row-level security. All three are ADR-042 decisions; read
-- it before changing anything here.
--
-- No COMMIT/ROLLBACK: the runner wraps this file and its log row in one
-- transaction. Never edit this file once applied (append-only, sha256 drift
-- is fatal) -- add 0002_*.up.sql instead.

CREATE SCHEMA IF NOT EXISTS orm;

-- from migrations/20260909_create_local_orm_schema.sql
CREATE TABLE IF NOT EXISTS orm.account (

    account_id      VARCHAR(20) NOT NULL PRIMARY KEY,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    is_discretionary BOOLEAN    NOT NULL DEFAULT true,
    tenant_id          UUID          NOT NULL
)
;

-- from db/migrations/20261026_011_orm_baskets.up.sql
CREATE TABLE IF NOT EXISTS orm.basket (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    basket_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    basket_type varchar(30) NOT NULL,
    source_type varchar(30),
    source_id uuid,
    total_notional numeric(24,4),
    currency varchar(3),
    status varchar(20) DEFAULT 'DRAFT' NOT NULL,
    created_by uuid,
    approved_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT basket_pkey PRIMARY KEY (id),
    CONSTRAINT uq_basket_cd UNIQUE (tenant_id, basket_cd),
    CONSTRAINT chk_b_type CHECK (basket_type IN (
        'CUSTOM','INDEX_REPLICATION','REBALANCE','PAIR','BASKET',
        'PORTFOLIO_TRANSITION','ETF_CREATE','ETF_REDEEM')),
    CONSTRAINT chk_b_source CHECK (source_type IS NULL OR source_type IN (
        'MODEL','INDEX','CUSTOM_UPLOAD','PORTFOLIO')),
    CONSTRAINT chk_b_status CHECK (status IN (
        'DRAFT','READY','ROUTED','PARTIAL','COMPLETED','CANCELLED'))
)
;

-- from migrations/20260910_create_orm_broker.sql
CREATE TABLE IF NOT EXISTS orm.broker (

    broker_id VARCHAR(20) NOT NULL PRIMARY KEY,
    status    VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    tenant_id          UUID          NOT NULL
)
;

-- from db/migrations/20261026_006_orm_position_keeping.up.sql
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
)
;

-- from db/migrations/20261026_014_orm_fx_exposure.up.sql
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
)
;

-- from db/migrations/20261026_014_orm_fx_exposure.up.sql
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
)
;

-- from db/migrations/20261026_007_orm_market_data.up.sql
CREATE TABLE IF NOT EXISTS orm.market_data_snapshot (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    as_of_date date NOT NULL,
    open_price numeric(18,9),
    high_price numeric(18,9),
    low_price numeric(18,9),
    close_price numeric(18,9),
    vwap numeric(18,9),
    volume numeric(24,4),
    trade_count int4,
    adjusted_close numeric(18,9),
    currency varchar(3),
    source varchar(50),
    is_final bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT market_data_snapshot_pkey PRIMARY KEY (id),
    CONSTRAINT uq_mds UNIQUE (tenant_id, security_id, as_of_date)
)
;

-- from db/migrations/20261026_010_orm_model_portfolios.up.sql
CREATE TABLE IF NOT EXISTS orm.model_portfolio (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    model_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    model_type varchar(30) NOT NULL,
    base_currency varchar(3) NOT NULL,
    manager_id uuid,
    benchmark_id uuid,
    risk_profile varchar(20),
    rebalance_frequency varchar(20),
    rebalance_threshold_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT model_portfolio_pkey PRIMARY KEY (id),
    CONSTRAINT uq_mp_cd UNIQUE (tenant_id, model_cd, effective_from),
    CONSTRAINT chk_mp_type CHECK (model_type IN (
        'BALANCED','GROWTH','INCOME','ESG','THEMATIC','TARGET_RISK',
        'TARGET_DATE','CUSTOM'))
)
;

-- from migrations/20260909_create_local_orm_schema.sql
CREATE TABLE IF NOT EXISTS orm."order" (

    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    sec_id          NUMERIC(18,0) NOT NULL,
    side            VARCHAR(10)   NOT NULL,
    order_type      VARCHAR(20)   NOT NULL,
    status          VARCHAR(20)   NOT NULL DEFAULT 'NEW',
    target_qty      NUMERIC(18,4) NOT NULL,
    executed_qty    NUMERIC(18,4) NOT NULL DEFAULT 0,
    leaves_qty      NUMERIC(18,4) NOT NULL,
    limit_price     NUMERIC(18,9),
    avg_price       NUMERIC(18,9) DEFAULT 0,
    time_in_force   VARCHAR(10)   DEFAULT 'DAY',
    trade_date      DATE          NOT NULL,
    manager_id      UUID,
    trader_id       UUID,
    custom_attributes JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id          UUID          NOT NULL
)
;

-- from db/migrations/20261026_013_orm_intraday_pnl.up.sql
CREATE TABLE IF NOT EXISTS orm.pnl_intraday (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    portfolio_id uuid,
    as_of_timestamp timestamptz NOT NULL,
    currency varchar(3) NOT NULL,
    realized_pnl numeric(24,4) DEFAULT 0,
    unrealized_pnl numeric(24,4) DEFAULT 0,
    total_pnl numeric(24,4) DEFAULT 0,
    income numeric(24,4) DEFAULT 0,
    commissions numeric(24,4) DEFAULT 0,
    fees numeric(24,4) DEFAULT 0,
    net_pnl numeric(24,4),
    market_value numeric(24,4),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT pnl_intraday_pkey PRIMARY KEY (id)
)
;

-- from db/migrations/20261026_006_orm_position_keeping.up.sql
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
)
;

-- from db/migrations/20261026_006_orm_position_keeping.up.sql
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
        'FIFO','LIFO','HIFO','AVG_COST','SPEC_ID'))
)
;

-- from db/migrations/20261026_007_orm_market_data.up.sql
CREATE TABLE IF NOT EXISTS orm.quote (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    venue_mic varchar(4),
    quote_time timestamptz NOT NULL,
    bid_price numeric(18,9),
    bid_size numeric(18,4),
    ask_price numeric(18,9),
    ask_size numeric(18,4),
    mid_price numeric(18,9),
    last_price numeric(18,9),
    last_size numeric(18,4),
    spread_bps numeric(12,6),
    quote_currency varchar(3),
    is_indicative bool DEFAULT false NOT NULL,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT quote_pkey PRIMARY KEY (id, quote_time)
)
 PARTITION BY RANGE (quote_time)
;

-- from db/migrations/20261026_004_orm_pre_trade_compliance.up.sql
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
)
;

-- from db/migrations/20261026_012_orm_routing_rules.up.sql
CREATE TABLE IF NOT EXISTS orm.routing_rule (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    priority int4 DEFAULT 100 NOT NULL,
    asset_class_cd varchar(20),
    sec_sub_typ_cd varchar(30),
    order_type varchar(20),
    side varchar(10),
    quantity_min numeric(18,4),
    quantity_max numeric(18,4),
    notional_min numeric(18,4),
    notional_max numeric(18,4),
    venue_mic varchar(4),
    broker_id uuid,
    routing_strategy varchar(30),
    algo_cd varchar(50),
    is_active bool DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT routing_rule_pkey PRIMARY KEY (id),
    CONSTRAINT uq_rr_cd UNIQUE (tenant_id, rule_cd),
    CONSTRAINT chk_rr_side CHECK (side IS NULL OR side IN ('BUY','SELL','SHORT','COVER')),
    CONSTRAINT chk_rr_strategy CHECK (routing_strategy IS NULL OR routing_strategy IN (
        'DIRECT','SMART_ORDER_ROUTER','DARK_FIRST','LIT_FIRST',
        'VWAP','TWAP','IS','POV','MANUAL'))
)
;

-- from db/migrations/20261026_005_orm_short_sell_locate.up.sql
CREATE TABLE IF NOT EXISTS orm.short_sell_locate (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid NOT NULL,
    account_id uuid NOT NULL,
    locate_type varchar(20) NOT NULL,
    locate_status varchar(20) NOT NULL,
    requested_at timestamptz NOT NULL,
    confirmed_at timestamptz,
    expiry_at timestamptz,
    quantity_requested numeric(18,4),
    quantity_confirmed numeric(18,4),
    quantity_used numeric(18,4) DEFAULT 0,
    locate_reference varchar(100),
    broker_id uuid,
    locate_fee_bps numeric(10,4),
    is_easy_to_borrow bool,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT short_sell_locate_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ssl_type CHECK (locate_type IN (
        'EASY_TO_BORROW','HARD_TO_BORROW','PRE_BORROW','RECALL',
        'BROKER_LOCATE','INTERNAL_LOCATE')),
    CONSTRAINT chk_ssl_status CHECK (locate_status IN (
        'REQUESTED','CONFIRMED','REJECTED','EXPIRED','USED','CANCELLED'))
)
;

-- from db/migrations/20261026_008_orm_trading_calendar.up.sql
CREATE TABLE IF NOT EXISTS orm.trading_halt (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    security_id uuid,
    mic varchar(4),
    halt_start timestamptz NOT NULL,
    halt_end timestamptz,
    halt_type varchar(30) NOT NULL,
    halt_reason varchar(500),
    is_resumed bool DEFAULT false NOT NULL,
    resumed_at timestamptz,
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT trading_halt_pkey PRIMARY KEY (id),
    CONSTRAINT chk_th_type CHECK (halt_type IN (
        'NEWS_PENDING','VOLATILITY','REGULATORY','TECHNICAL',
        'CIRCUIT_BREAKER','LULD','IPO','MERGER'))
)
;

-- from db/migrations/20261026_009_orm_trading_limits.up.sql
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
)
;

-- from db/migrations/20261026_008_orm_trading_calendar.up.sql
CREATE TABLE IF NOT EXISTS orm.trading_session (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    market_id uuid,
    mic varchar(4),
    session_date date NOT NULL,
    session_type varchar(30) NOT NULL,
    open_time time,
    close_time time,
    time_zone varchar(100),
    is_early_close bool DEFAULT false NOT NULL,
    early_close_reason varchar(255),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT trading_session_pkey PRIMARY KEY (id),
    CONSTRAINT uq_ts UNIQUE (tenant_id, session_date, session_type, mic),
    CONSTRAINT chk_ts_type CHECK (session_type IN (
        'PRE_MARKET','OPENING_AUCTION','CONTINUOUS','CLOSING_AUCTION',
        'POST_MARKET','CLOSED','HALT','EARLY_CLOSE','LATE_OPEN'))
)
;

-- from db/migrations/20261026_010_orm_model_portfolios.up.sql
CREATE TABLE IF NOT EXISTS orm.account_model_assignment (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    model_portfolio_id uuid NOT NULL,
    allocation_pct numeric(7,4) DEFAULT 100.0 NOT NULL,
    drift_tolerance_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT account_model_assignment_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ama_model FOREIGN KEY (model_portfolio_id) REFERENCES orm.model_portfolio(id)
)
;

-- from db/migrations/20261026_011_orm_baskets.up.sql
CREATE TABLE IF NOT EXISTS orm.basket_item (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    basket_id uuid NOT NULL,
    security_id uuid NOT NULL,
    side varchar(10) NOT NULL,
    target_qty numeric(18,4),
    target_notional numeric(18,4),
    target_weight_pct numeric(7,4),
    order_id uuid,
    sequence_number int4,
    status varchar(20) DEFAULT 'PENDING' NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT basket_item_pkey PRIMARY KEY (id),
    CONSTRAINT chk_bi_side CHECK (side IN ('BUY','SELL','SELL_SHORT','BUY_TO_COVER')),
    CONSTRAINT chk_bi_status CHECK (status IN ('PENDING','ROUTED','PARTIAL','FILLED','CANCELLED')),
    CONSTRAINT fk_bi_basket FOREIGN KEY (basket_id) REFERENCES orm.basket(id) ON DELETE CASCADE
)
;

-- from db/migrations/20261026_010_orm_model_portfolios.up.sql
CREATE TABLE IF NOT EXISTS orm.model_portfolio_target (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    model_portfolio_id uuid NOT NULL,
    target_type varchar(30) NOT NULL,
    target_reference_cd varchar(50),
    security_id uuid,
    target_weight_pct numeric(7,4) NOT NULL,
    min_weight_pct numeric(7,4),
    max_weight_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT model_portfolio_target_pkey PRIMARY KEY (id),
    CONSTRAINT chk_mpt_type CHECK (target_type IN (
        'SECURITY','ASSET_CLASS','SECTOR','REGION','SLEEVE','ETF','MUTUAL_FUND')),
    CONSTRAINT fk_mpt_model FOREIGN KEY (model_portfolio_id) REFERENCES orm.model_portfolio(id) ON DELETE CASCADE
)
;

-- from migrations/20260909_create_local_orm_schema.sql
CREATE TABLE IF NOT EXISTS orm.order_allocation (

    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    order_id        UUID          NOT NULL,
    account_id      VARCHAR(20)   NOT NULL,
    target_qty      NUMERIC(18,4) NOT NULL,
    allocated_qty   NUMERIC(18,4) NOT NULL DEFAULT 0,
    status          VARCHAR(20)   NOT NULL DEFAULT 'NEW',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT order_allocation_order_id_fkey FOREIGN KEY (order_id) REFERENCES orm."order"(id),
    tenant_id          UUID          NOT NULL
)
;

-- from db/migrations/20261026_002_orm_order_lifecycle.up.sql
CREATE TABLE IF NOT EXISTS orm.order_amendment (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    amendment_num int4 NOT NULL,
    amendment_time timestamptz NOT NULL,
    field_changed varchar(50) NOT NULL,
    prior_value text,
    new_value text,
    reason varchar(500),
    amended_by uuid,
    is_client_directed bool DEFAULT false NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_amendment_pkey PRIMARY KEY (id),
    CONSTRAINT uq_oa UNIQUE (tenant_id, order_id, amendment_num),
    CONSTRAINT fk_oa_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
)
;

-- from db/migrations/20261026_003_orm_execution_quality.up.sql
CREATE TABLE IF NOT EXISTS orm.order_benchmark (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    benchmark_type varchar(30) NOT NULL,
    benchmark_time timestamptz,
    benchmark_price numeric(18,9),
    benchmark_quantity numeric(18,4),
    benchmark_currency varchar(3),
    source varchar(50),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_benchmark_pkey PRIMARY KEY (id),
    CONSTRAINT chk_ob_type CHECK (benchmark_type IN (
        'ARRIVAL','DECISION','CLOSE','OPEN','VWAP','TWAP','INTERVAL',
        'IMPLEMENTATION_SHORTFALL','PRE_TRADE','POST_TRADE')),
    CONSTRAINT fk_ob_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
)
;

-- from db/migrations/20261026_002_orm_order_lifecycle.up.sql
CREATE TABLE IF NOT EXISTS orm.order_event (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    event_type varchar(30) NOT NULL,
    from_status varchar(20),
    to_status varchar(20) NOT NULL,
    event_time timestamptz NOT NULL,
    event_source varchar(30),
    message_id uuid,
    reason_cd varchar(50),
    reason_text varchar(500),
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_event_pkey PRIMARY KEY (id),
    CONSTRAINT chk_oe_event_type CHECK (event_type IN (
        'NEW','ACKNOWLEDGED','PARTIAL_FILL','FILLED','CANCELLED',
        'REJECTED','EXPIRED','REPLACED','PENDING_NEW','PENDING_CANCEL',
        'SUSPENDED','HELD','RELEASED')),
    CONSTRAINT chk_oe_source CHECK (event_source IS NULL OR event_source IN (
        'OMS','BROKER','EXCHANGE','MANUAL','SYSTEM')),
    CONSTRAINT fk_oe_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
)
;

-- from db/migrations/20261026_002_orm_order_lifecycle.up.sql
CREATE TABLE IF NOT EXISTS orm.order_history (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    version_num int4 NOT NULL,
    valid_from timestamptz NOT NULL,
    valid_to timestamptz,
    is_current bool DEFAULT true NOT NULL,
    record_snapshot jsonb NOT NULL,
    changed_columns text[],
    change_source varchar(30),
    changed_by uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_history_pkey PRIMARY KEY (id),
    CONSTRAINT uq_oh_version UNIQUE (tenant_id, order_id, version_num),
    CONSTRAINT fk_oh_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
)
;

-- from db/migrations/20261026_002_orm_order_lifecycle.up.sql
CREATE TABLE IF NOT EXISTS orm.order_reject (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id uuid NOT NULL,
    placement_id uuid,
    reject_time timestamptz NOT NULL,
    reject_source varchar(30) NOT NULL,
    reject_code varchar(50),
    reject_reason varchar(500),
    reject_text text,
    is_retriable bool DEFAULT false NOT NULL,
    retried_as_order_id uuid,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT order_reject_pkey PRIMARY KEY (id),
    CONSTRAINT chk_orj_source CHECK (reject_source IN (
        'OMS','BROKER','EXCHANGE','COMPLIANCE','RISK','MANUAL')),
    CONSTRAINT fk_orj_order FOREIGN KEY (order_id) REFERENCES orm."order"(id) ON DELETE CASCADE
)
;

-- from migrations/20260909_create_local_orm_schema.sql
CREATE TABLE IF NOT EXISTS orm.placement (

    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    order_id        UUID          NOT NULL,
    broker_id       VARCHAR(20)   NOT NULL,
    venue_id        VARCHAR(20),
    routed_qty      NUMERIC(18,4) NOT NULL,
    executed_qty    NUMERIC(18,4) NOT NULL DEFAULT 0,
    leaves_qty      NUMERIC(18,4) NOT NULL,
    status          VARCHAR(20)   NOT NULL DEFAULT 'NEW',
    fix_clordid     VARCHAR(120),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT placement_order_id_fkey FOREIGN KEY (order_id) REFERENCES orm."order"(id),
    tenant_id          UUID          NOT NULL
)
;

-- from db/migrations/20261026_004_orm_pre_trade_compliance.up.sql
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
)
;

-- from migrations/20260909_create_local_orm_schema.sql
CREATE TABLE IF NOT EXISTS orm.execution (

    id              UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    placement_id    UUID          NOT NULL,
    order_id        UUID          NOT NULL,
    exec_qty        NUMERIC(18,4) NOT NULL,
    exec_price      NUMERIC(18,9) NOT NULL,
    broker_id       VARCHAR(50),
    exec_time       TIMESTAMPTZ   NOT NULL,
    transact_time   TIMESTAMPTZ   NOT NULL,
    status          VARCHAR(20)   NOT NULL,
    broker_exec_id  VARCHAR(120),
    last_capacity   VARCHAR(1),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ   DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT execution_placement_id_fkey FOREIGN KEY (placement_id) REFERENCES orm.placement(id),
    CONSTRAINT execution_order_id_fkey FOREIGN KEY (order_id) REFERENCES orm."order"(id),
    tenant_id          UUID          NOT NULL
)
;

-- from migrations/20260909_create_local_orm_schema.sql
CREATE TABLE IF NOT EXISTS orm.execution_allocation (

    id                  UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    execution_id        UUID          NOT NULL,
    order_allocation_id UUID          NOT NULL,
    alloc_exec_qty      NUMERIC(18,4) NOT NULL,
    alloc_exec_price    NUMERIC(18,9) NOT NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT execution_allocation_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES orm.execution(id),
    CONSTRAINT execution_allocation_order_allocation_id_fkey FOREIGN KEY (order_allocation_id) REFERENCES orm.order_allocation(id),
    tenant_id          UUID          NOT NULL
)
;

-- from db/migrations/20261026_003_orm_execution_quality.up.sql
CREATE TABLE IF NOT EXISTS orm.execution_quality (

    id uuid DEFAULT gen_random_uuid() NOT NULL,
    execution_id uuid NOT NULL,
    arrival_price numeric(18,9),
    arrival_price_time timestamptz,
    interval_vwap numeric(18,9),
    market_vwap numeric(18,9),
    slippage_bps numeric(12,6),
    slippage_amount numeric(18,4),
    implementation_shortfall_bps numeric(12,6),
    implementation_shortfall_amount numeric(18,4),
    opportunity_cost_bps numeric(12,6),
    opportunity_cost_amount numeric(18,4),
    participation_rate_pct numeric(7,4),
    benchmark_ref varchar(50),
    measurement_notes text,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT execution_quality_pkey PRIMARY KEY (id),
    CONSTRAINT uq_eq UNIQUE (tenant_id, execution_id),
    CONSTRAINT fk_eq_exec FOREIGN KEY (execution_id) REFERENCES orm.execution(id) ON DELETE CASCADE
)
;

-- `quote` is RANGE-partitioned by quote_time; a tenant database gets the
-- default partition only, as the source does. Partition management (pg_partman
-- or monthly drops) is a retention concern and belongs to ADR-035, not here.
CREATE TABLE IF NOT EXISTS orm.quote_default PARTITION OF orm.quote DEFAULT;

-- Indexes, from the same sources. The three that lived on orm.quote_default
-- are reattached to orm.quote, the partitioned parent, where they propagate.
CREATE INDEX IF NOT EXISTS idx_orm_placement_order ON orm.placement(order_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_allocation_order ON orm.order_allocation(order_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_placement ON orm.execution(placement_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_order ON orm.execution(order_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_allocation_execution ON orm.execution_allocation(execution_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_allocation_order_allocation ON orm.execution_allocation(order_allocation_id);
CREATE INDEX IF NOT EXISTS idx_oe_order ON orm.order_event (order_id, event_time);
CREATE INDEX IF NOT EXISTS idx_oe_tenant ON orm.order_event (tenant_id);
CREATE INDEX IF NOT EXISTS idx_oa_order ON orm.order_amendment (order_id, amendment_time);
CREATE INDEX IF NOT EXISTS idx_orj_order ON orm.order_reject (order_id, reject_time);
CREATE INDEX IF NOT EXISTS idx_oh_order ON orm.order_history (order_id, is_current);
CREATE INDEX IF NOT EXISTS idx_eq_exec ON orm.execution_quality (execution_id);
CREATE INDEX IF NOT EXISTS idx_ob_order ON orm.order_benchmark (order_id, benchmark_type);
CREATE INDEX IF NOT EXISTS idx_ptc_order ON orm.pre_trade_check (order_id, check_time);
CREATE INDEX IF NOT EXISTS idx_ptc_status ON orm.pre_trade_check (check_status, check_type);
CREATE INDEX IF NOT EXISTS idx_rl_security ON orm.restricted_list (security_id, is_active) WHERE security_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rl_account ON orm.restricted_list (account_id, is_active) WHERE account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rl_tenant ON orm.restricted_list (tenant_id);
CREATE INDEX IF NOT EXISTS idx_ssl_sec ON orm.short_sell_locate (security_id, locate_status);
CREATE INDEX IF NOT EXISTS idx_ssl_acct ON orm.short_sell_locate (account_id, locate_status);
CREATE INDEX IF NOT EXISTS idx_ssl_expiry ON orm.short_sell_locate (expiry_at) WHERE locate_status = 'CONFIRMED';
CREATE INDEX IF NOT EXISTS idx_pl_position ON orm.position_lot (position_id, is_open);
CREATE INDEX IF NOT EXISTS idx_pl_acct_sec ON orm.position_lot (account_id, security_id, is_open);
CREATE INDEX IF NOT EXISTS idx_ph_acct_date ON orm.position_history (account_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_ph_sec ON orm.position_history (security_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_cb_acct ON orm.cash_balance (account_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_quote_sec ON orm.quote (security_id, quote_time);
CREATE INDEX IF NOT EXISTS idx_quote_mic ON orm.quote (venue_mic, quote_time);
CREATE INDEX IF NOT EXISTS idx_quote_tenant ON orm.quote (tenant_id);
CREATE INDEX IF NOT EXISTS idx_mds_sec ON orm.market_data_snapshot (security_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_ts_date ON orm.trading_session (session_date, mic);
CREATE INDEX IF NOT EXISTS idx_th_sec ON orm.trading_halt (security_id, halt_start) WHERE security_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_th_mic ON orm.trading_halt (mic, halt_start) WHERE mic IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tl_scope ON orm.trading_limit (scope_type, scope_id, limit_type);
CREATE INDEX IF NOT EXISTS idx_tl_tenant ON orm.trading_limit (tenant_id);
CREATE INDEX IF NOT EXISTS idx_mp_cd ON orm.model_portfolio (model_cd, is_current);
CREATE INDEX IF NOT EXISTS idx_mpt_model ON orm.model_portfolio_target (model_portfolio_id, is_current);
CREATE INDEX IF NOT EXISTS idx_ama_account ON orm.account_model_assignment (account_id, is_current);
CREATE INDEX IF NOT EXISTS idx_bi_basket ON orm.basket_item (basket_id);
CREATE INDEX IF NOT EXISTS idx_rr_priority ON orm.routing_rule (priority, is_active);
CREATE INDEX IF NOT EXISTS idx_pnl_acct ON orm.pnl_intraday (account_id, as_of_timestamp);
CREATE INDEX IF NOT EXISTS idx_pnl_portfolio ON orm.pnl_intraday (portfolio_id, as_of_timestamp) WHERE portfolio_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_fxe_acct ON orm.fx_exposure (account_id, as_of_date);
CREATE INDEX IF NOT EXISTS idx_fxh_acct ON orm.fx_hedge (account_id, is_active);

-- Every table is tenant-scoped (ADR-042) and every query filters on
-- tenant_id, so each gets one. The tables the sources already index by
-- tenant (idx_oe_tenant, idx_rl_tenant, idx_tl_tenant, idx_quote_tenant) are
-- left alone; the rest would otherwise seq-scan as a tenant's data grows.
CREATE INDEX IF NOT EXISTS idx_orm_account_tenant ON orm.account (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_basket_tenant ON orm.basket (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_broker_tenant ON orm.broker (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_cash_balance_tenant ON orm.cash_balance (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_fx_exposure_tenant ON orm.fx_exposure (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_fx_hedge_tenant ON orm.fx_hedge (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_market_data_snapshot_tenant ON orm.market_data_snapshot (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_model_portfolio_tenant ON orm.model_portfolio (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_tenant ON orm."order" (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_pnl_intraday_tenant ON orm.pnl_intraday (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_position_history_tenant ON orm.position_history (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_position_lot_tenant ON orm.position_lot (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_quote_tenant ON orm.quote (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_restricted_list_tenant ON orm.restricted_list (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_routing_rule_tenant ON orm.routing_rule (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_short_sell_locate_tenant ON orm.short_sell_locate (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_trading_halt_tenant ON orm.trading_halt (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_trading_limit_tenant ON orm.trading_limit (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_trading_session_tenant ON orm.trading_session (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_account_model_assignment_tenant ON orm.account_model_assignment (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_basket_item_tenant ON orm.basket_item (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_model_portfolio_target_tenant ON orm.model_portfolio_target (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_allocation_tenant ON orm.order_allocation (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_amendment_tenant ON orm.order_amendment (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_benchmark_tenant ON orm.order_benchmark (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_event_tenant ON orm.order_event (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_history_tenant ON orm.order_history (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_order_reject_tenant ON orm.order_reject (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_placement_tenant ON orm.placement (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_pre_trade_check_tenant ON orm.pre_trade_check (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_tenant ON orm.execution (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_allocation_tenant ON orm.execution_allocation (tenant_id);
CREATE INDEX IF NOT EXISTS idx_orm_execution_quality_tenant ON orm.execution_quality (tenant_id);

