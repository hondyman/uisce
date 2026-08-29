-- 20260901_oms_seed_data.sql
-- Seed data for OMS (Order Management System) operational tables plus
-- required ref.* and mds.* master / reference data.
-- Requires: oms schema + tables already created (user's DDL).
-- Idempotent: uses ON CONFLICT DO NOTHING / DO UPDATE where appropriate.
--
-- NOTE: The oms.default_order_status() trigger performs a validation that is
-- sensitive to the order_type lookup. Triggers on oms.{orders,order_slice,
-- allocation,settlement} are disabled during this seed run to avoid a spurious
-- check_violation error (re-enabled at the end of this file).

-- ============================================================================
-- SCHEMAS
-- ============================================================================
CREATE SCHEMA IF NOT EXISTS ref;
CREATE SCHEMA IF NOT EXISTS mds;

-- ============================================================================
-- REF SCHEMA — reference / lookup tables
-- Run after: nothing (self-contained)
-- ============================================================================

-- ref.asset_class ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.asset_class (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(30)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.asset_class (code, description) VALUES
    ('EQUITY',      'Public equities / stocks'),
    ('FIXED_INCOME','Bonds, notes, bills, preferred stock'),
    ('FX',          'Foreign exchange spot / forwards / swaps'),
    ('DERIVATIVE',  'General OTC / listed derivatives'),
    ('FUTURE',      'Exchange-traded futures'),
    ('OPTION',      'Exchange-traded and OTC options'),
    ('SWAP',        'OTC swap contracts'),
    ('CASH',        'Cash and cash equivalents'),
    ('COMMODITY',   'Physical commodities'),
    ('MULTI_ASSET', 'Multi-asset mandate / fund')
ON CONFLICT (code) DO NOTHING;

-- ref.currency ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.currency (
    id              SERIAL PRIMARY KEY,
    iso3_code       CHAR(3)      NOT NULL UNIQUE,
    numeric_code    CHAR(3),
    name            VARCHAR(100) NOT NULL,
    minor_unit      INTEGER      DEFAULT 2,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
INSERT INTO ref.currency (iso3_code, numeric_code, name) VALUES
    ('USD','840','US Dollar'),
    ('EUR','978','Euro'),
    ('GBP','826','British Pound'),
    ('JPY','392','Japanese Yen'),
    ('CHF','756','Swiss Franc'),
    ('AUD','036','Australian Dollar'),
    ('CAD','124','Canadian Dollar'),
    ('HKD','344','Hong Kong Dollar'),
    ('SGD','702','Singapore Dollar')
ON CONFLICT (iso3_code) DO NOTHING;

-- ref.exchange ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.exchange (
    id          SERIAL PRIMARY KEY,
    mic         CHAR(4)       NOT NULL UNIQUE,
    name        VARCHAR(200)  NOT NULL,
    country     VARCHAR(100),
    timezone    VARCHAR(50),
    currency_id INTEGER REFERENCES ref.currency(id),
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
INSERT INTO ref.exchange (mic, name, country, timezone, currency_id) VALUES
    ('XNAS','Nasdaq','US','America/New_York',1),    -- USD
    ('XNYS','New York Stock Exchange','US','America/New_York',1),
    ('ARCX','NYSE Arca','US','America/New_York',1),
    ('BATS','Cboe BATS','US','America/Chicago',1),
    ('EDGX','Cboe EDGX','US','America/Chicago',1),
    ('XLON','London Stock Exchange','UK','Europe/London',3),  -- GBP
    ('XSWX','SIX Swiss Exchange','CH','Europe/Zurich',5),     -- CHF
    ('XTKS','Tokyo Stock Exchange','JP','Asia/Tokyo',4)       -- JPY
ON CONFLICT (mic) DO NOTHING;

-- ref.order_type -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.order_type (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(30)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.order_type (code, description) VALUES
    ('MARKET',     'Market order — executes immediately at best available price'),
    ('LIMIT',      'Limit order — executes at specified price or better'),
    ('STOP',       'Stop order — triggered to market when stop price reached'),
    ('STOP_LIMIT', 'Stop-limit order — triggered to limit when stop price reached'),
    ('ALGO',       'Algorithm-driven order (TWAP, VWAP, POV, IS, etc.)'),
    ('FOK',        'Fill-or-Kill (special TIF, executed as single unit or cancelled)'),
    ('IOC',        'Immediate-or-Cancel (partial fills OK, remaining cancelled)'),
    ('MOO',        'Market-on-Open'),
    ('LOC',        'Limit-on-Open'),
    ('MOC',        'Market-on-Close'),
    ('LOO',        'Limit-on-Close'),
    ('PEG',        'Pegged order'),
    ('HYBRID',     'Multi-leg / multivenue hybrid')
ON CONFLICT (code) DO NOTHING;

-- ref.time_in_force ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.time_in_force (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(10)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.time_in_force (code, description) VALUES
    ('DAY',  'Day order — expires at end of trading session'),
    ('GTC',  'Good-Till-Cancelled — active until explicitly cancelled'),
    ('IOC',  'Immediate-or-Cancel — fills what it can, cancels rest'),
    ('FOK',  'Fill-or-Kill — must fill completely in one print or cancelled'),
    ('GTD',  'Good-Till-Date — active until specified expire_at timestamp'),
    ('ATC',  'At-the-Close — routed to close auction'),
    ('OPG',  'At-the-Opening — routed to open auction')
ON CONFLICT (code) DO NOTHING;

-- ref.side -------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.side (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.side (code, description) VALUES
    ('BUY',         'Buy / Long'),
    ('SELL',        'Sell / Close Long'),
    ('SELL_SHORT',  'Sell Short'),
    ('BUY_TO_COVER','Buy to Cover Short'),
    ('BUY_WRAP',    'Buy as part of a multi-leg strategy wrap'),
    ('SELL_WRAP',   'Sell as part of a multi-leg strategy unwrap')
ON CONFLICT (code) DO NOTHING;

-- ref.order_status -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.order_status (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(30)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.order_status (code, description) VALUES
    ('NEW',              'Order accepted by OMS, awaiting risk check'),
    ('RISK_PENDING',     'Submitted to risk engine, awaiting approval'),
    ('RISK_APPROVED',    'Risk engine approved'),
    ('RISK_REJECTED',    'Risk engine rejected — not routed'),
    ('WORKING',          'Routed to venue / algo, actively working'),
    ('PARTIALLY_FILLED', 'Partially executed'),
    ('FILLED',           'Fully executed'),
    ('CANCEL_REQUESTED', 'Cancel requested by trader, pending venue ack'),
    ('CANCELED',         'Cancelled by trader or risk before fill'),
    ('REJECTED',         'Venue / broker rejected'),
    ('EXPIRED',          'Expired per TIF (GTD/DAY) or schedule'),
    ('SETTLED',          'Settled (cash equities)'),
    ('FAILED',          'Technical / operational failure'),
    ('SUSPENDED',       'Suspended (regulatory / risk freeze)')
ON CONFLICT (code) DO NOTHING;

-- ref.order_event_type -------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.order_event_type (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.order_event_type (code, description) VALUES
    ('CREATED',             'Order created in OMS'),
    ('SUBMITTED',           'Submitted to routing / gateway'),
    ('ACKNOWLEDGED',        'Venue / broker acknowledged receipt'),
    ('RISK_APPROVED',       'Risk engine approved'),
    ('RISK_REJECTED',       'Risk engine rejected'),
    ('VENUE_REJECTED',      'Venue rejected the order'),
    ('WORKING',             'First live quote / working at venue'),
    ('AMENDED',            'Order amended (price / qty change)'),
    ('CANCEL_REQUESTED',   'Cancel requested'),
    ('CANCELED',           'Cancelled (trader / risk / system)'),
    ('PARTIAL_FILL',        'Partial execution occurred'),
    ('FULL_FILL',           'Full execution occurred'),
    ('EXPIRED',            'Expired per TIF'),
    ('SUSPENDED',          'Suspended by risk / regulatory'),
    ('REACTIVATED',        'Reactivated after suspension'),
    ('SETTLED',            'Trade settled'),
    ('FAILED',             'Technical failure'),
    ('POSITION_PERSISTED', 'Position updated in books')
ON CONFLICT (code) DO NOTHING;

-- ref.event_source ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.event_source (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.event_source (code, description) VALUES
    ('VENUE',    'Venue / exchange / ATS'),
    ('TRADER',   'Trader action via GUI / API'),
    ('INTERNAL', 'Internal OMS logic (risk, algo, scheduler)'),
    ('RISK',     'Risk engine decision'),
    ('GATEWAY',  'FIX / native gateway / wire'),
    ('SYSTEM',   'Scheduled / batch / system action')
ON CONFLICT (code) DO NOTHING;

-- ref.order_link_type --------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.order_link_type (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(30)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.order_link_type (code, description) VALUES
    ('SLICE_OF',    'Child is a slice / child-order of parent'),
    ('REPLACES',    'Child replaces parent (cancel/replace)'),
    ('HEDGED_BY',   'Child is a hedge for parent'),
    ('PAIR_OF',     'Child is part of a paired-trade (FX lot split)'),
    ('LEG',         'Leg of a multi-leg strategy (bundle/wrap)'),
    ('CLONE',       'Cloned copy for risk monitoring')
ON CONFLICT (code) DO NOTHING;

-- ref.venue_type -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.venue_type (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.venue_type (code, description) VALUES
    ('LIT',  'Lit venue — pre-trade transparent'),
    ('DARK', 'Dark pool / dark venue'),
    ('OTC',  'Over-the-counter / bilateral'),
    ('ECN',  'Electronic Communication Network'),
    ('RFQ',  'Request-for-Quote venue'),
    ('FW',   'Firm-up venue (periodic auction)')
ON CONFLICT (code) DO NOTHING;

-- ref.allocation_status -------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.allocation_status (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.allocation_status (code, description) VALUES
    ('PENDING',     'Awaiting allocation instruction'),
    ('ALLOCATED',  'Allocated to destination account'),
    ('FAILED',     'Allocation rejected / failed'),
    ('CANCELED',   'Allocation canceled')
ON CONFLICT (code) DO NOTHING;

-- ref.settlement_status -------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.settlement_status (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.settlement_status (code, description) VALUES
    ('PENDING',    'Settlement instruction generated'),
    ('INSTRUCTED','Instruction sent to custodian / CCP'),
    ('MATCHED',   'Matching complete at CCP'),
    ('SETTLED',   'Settled — cash/securities exchanged'),
    ('FAILED',    'Settlement failed'),
    ('CANCELED',  'Settlement canceled')
ON CONFLICT (code) DO NOTHING;

-- ref.liquidity_flag ----------------------------------------------------------
CREATE TABLE IF NOT EXISTS ref.liquidity_flag (
    id          SERIAL PRIMARY KEY,
    code        CHAR(1)      NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ref.liquidity_flag (code, description) VALUES
    ('A', 'Add — initiator / maker liquidity (order added to book)'),
    ('R', 'Remove — taker liquidity (order removed from book)'),
    ('N', 'Neither — non-displayable / hidden order'),
    ('X', 'Auction — print from auction process')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- MDS SCHEMA — master data tables
-- Run after: ref schema + tables exist
-- ============================================================================

-- mds.counterparty -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS mds.counterparty (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    code            VARCHAR(30) NOT NULL,
    name            VARCHAR(200)NOT NULL,
    bic             VARCHAR(11),
    lei             VARCHAR(20),
    country         VARCHAR(3),
    kyc_status      VARCHAR(20) DEFAULT 'PENDING',
    kyc_status_date TIMESTAMPTZ,
    credit_rating   VARCHAR(10),
    credit_limit    NUMERIC(20,2),
    internal_flag   BOOLEAN     DEFAULT false,
    is_active       BOOLEAN     DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_counterparty_tenant_code UNIQUE (tenant_id, code)
);
CREATE INDEX IF NOT EXISTS idx_counterparty_tenant ON mds.counterparty (tenant_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_counterparty_lei    ON mds.counterparty (lei) WHERE lei IS NOT NULL;

-- mds.exchange_membership -----------------------------------------------------
CREATE TABLE IF NOT EXISTS mds.exchange_membership (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    counterparty_id UUID        NOT NULL REFERENCES mds.counterparty(id),
    exchange_mic    CHAR(4)     NOT NULL REFERENCES ref.exchange(mic),
    is_active       BOOLEAN     DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_exchange_membership UNIQUE (tenant_id, counterparty_id, exchange_mic)
);

-- mds.account ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mds.account (
    id                UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id         UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    code              VARCHAR(30) NOT NULL,
    name              VARCHAR(200),
    account_type      VARCHAR(20) NOT NULL CHECK (account_type IN ('CASH','MARGIN','SEGREGATED','HOUSE','FIRM','POOLING')),
    base_currency     CHAR(3)    NOT NULL REFERENCES ref.currency(iso3_code),
    counterparty_id   UUID        REFERENCES mds.counterparty(id),
    margin_limit      NUMERIC(20,2),
    is_tradeable      BOOLEAN     DEFAULT true,
    is_active         BOOLEAN     DEFAULT true,
    opened_at         TIMESTAMPTZ,
    closed_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_account_tenant_code UNIQUE (tenant_id, code)
);
CREATE INDEX IF NOT EXISTS idx_mds_account_tenant ON mds.account (tenant_id) WHERE is_active = true;

-- mds.portfolio ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mds.portfolio (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    account_id      UUID        NOT NULL REFERENCES mds.account(id),
    code            VARCHAR(30) NOT NULL,
    name            VARCHAR(200),
    mandate         VARCHAR(50),
    base_currency   CHAR(3)    NOT NULL REFERENCES ref.currency(iso3_code),
    risk_model_id   UUID,
    benchmark_id    UUID,
    is_active       BOOLEAN     DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_portfolio_tenant_code UNIQUE (tenant_id, code)
);
CREATE INDEX IF NOT EXISTS idx_portfolio_tenant ON mds.portfolio (tenant_id) WHERE is_active = true;

-- mds.security_master ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS mds.security_master (
    id                    UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id             UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    asset_class           VARCHAR(30) NOT NULL REFERENCES ref.asset_class(code),
    symbol                VARCHAR(30) NOT NULL,
    isin                  CHAR(12),
    cusip                 VARCHAR(9),
    sedol                 VARCHAR(7),
    ric                   VARCHAR(20),
    bloomberg_ticker      VARCHAR(30),
    exchange_mic          CHAR(4)     REFERENCES ref.exchange(mic),
    exchange_symbol       VARCHAR(30),
    name                  VARCHAR(300),
    short_name            VARCHAR(100),
    currency_id           INTEGER     REFERENCES ref.currency(id),
    lot_size              NUMERIC(18,6),
    tick_size             NUMERIC(18,8),
    eq_country            VARCHAR(3),
    eq_sector             VARCHAR(100),
    eq_industry           VARCHAR(100),
    eq_market_cap         NUMERIC(20,2),
    eq_dividend_yield     NUMERIC(10,6),
    eq_beta               NUMERIC(10,4),
    eq_free_float         NUMERIC(5,4),
    fi_coupon_rate        NUMERIC(10,6),
    fi_coupon_freq        INTEGER,
    fi_day_count          VARCHAR(20),
    fi_issue_date         DATE,
    fi_maturity_date      DATE,
    fi_face_value         NUMERIC(20,2),
    fi_rating             VARCHAR(10),
    fi_rating_agency      VARCHAR(10),
    fi_callable           BOOLEAN   DEFAULT false,
    fi_call_date          DATE,
    fi_call_price         NUMERIC(10,4),
    fi_puttable           BOOLEAN   DEFAULT false,
    fi_convertible        BOOLEAN   DEFAULT false,
    fi_principal_ccy      CHAR(3)   REFERENCES ref.currency(iso3_code),
    fx_ccy_base           CHAR(3)   REFERENCES ref.currency(iso3_code),
    fx_ccy_quote          CHAR(3)   REFERENCES ref.currency(iso3_code),
    fx_pip_precision      INTEGER   DEFAULT 4,
    fx_value_date_conv    VARCHAR(20),
    deriv_underlying_id   UUID      REFERENCES mds.security_master(id),
    deriv_contract_size   NUMERIC(20,6),
    deriv_tick_value     NUMERIC(20,6),
    deriv_expiry_date    DATE,
    deriv_settlement     VARCHAR(20),
    deriv_first_trade    DATE,
    deriv_last_trade     DATE,
    opt_option_type      VARCHAR(10) CHECK (opt_option_type IN ('CALL','PUT')),
    opt_strike_price     NUMERIC(20,8),
    opt_expiry_date      DATE,
    opt_settlement_method VARCHAR(20),
    opt_european_flag    BOOLEAN   DEFAULT true,
    swap_leg1_ccy        CHAR(3)  REFERENCES ref.currency(iso3_code),
    swap_leg2_ccy        CHAR(3)  REFERENCES ref.currency(iso3_code),
    swap_leg1_pay_rec    VARCHAR(5),
    swap_leg2_pay_rec    VARCHAR(5),
    swap_rate_index1     VARCHAR(30),
    swap_rate_index2     VARCHAR(30),
    swap_notional_ccy    CHAR(3)  REFERENCES ref.currency(iso3_code),
    swap_maturity_date   DATE,
    is_active            BOOLEAN   DEFAULT true,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_security_master_tenant_symbol UNIQUE (tenant_id, symbol),
    CONSTRAINT uq_security_master_isin         UNIQUE (tenant_id, isin)
);
CREATE INDEX IF NOT EXISTS idx_security_tenant    ON mds.security_master (tenant_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_security_asset_cl  ON mds.security_master (tenant_id, asset_class);
CREATE INDEX IF NOT EXISTS idx_security_exchange  ON mds.security_master (exchange_mic) WHERE exchange_mic IS NOT NULL;

-- mds.venue ------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mds.venue (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    counterparty_id     UUID        NOT NULL REFERENCES mds.counterparty(id),
    venue_type_id        INTEGER     NOT NULL REFERENCES ref.venue_type(id),
    exchange_mic        CHAR(4)     REFERENCES ref.exchange(mic),
    venue_code          VARCHAR(30) NOT NULL,
    name                VARCHAR(200),
    supports_market     BOOLEAN     DEFAULT false,
    supports_limit      BOOLEAN     DEFAULT false,
    supports_stop       BOOLEAN     DEFAULT false,
    supports_algo       BOOLEAN     DEFAULT false,
    is_active           BOOLEAN     DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_venue_tenant_code UNIQUE (tenant_id, venue_code)
);
CREATE INDEX IF NOT EXISTS idx_venue_cp  ON mds.venue (counterparty_id);
CREATE INDEX IF NOT EXISTS idx_venue_mic ON mds.venue (exchange_mic) WHERE exchange_mic IS NOT NULL;

-- ============================================================================
-- SEED DATA — mds master data
-- ============================================================================

-- Known UUIDs for stable cross-referencing
DO $$
DECLARE
    sys_tenant  UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    cp_pb       UUID := 'b1ffdd00-9c0b-4ef8-bb6d-6bb9bd380b01';  -- Prime broker
    cp_dealer1  UUID := 'b2ffdd00-9c0b-4ef8-bb6d-6bb9bd380b02';  -- Executing dealer 1
    cp_dealer2  UUID := 'b3ffdd00-9c0b-4ef8-bb6d-6bb9bd380b03';  -- Executing dealer 2
    acct1       UUID := 'c1ffdd00-9c0b-4ef8-bb6d-6bb9bd380c01';
    acct2       UUID := 'c2ffdd00-9c0b-4ef8-bb6d-6bb9bd380c02';
    pf1         UUID := 'd1ffdd00-9c0b-4ef8-bb6d-6bb9bd380d01';
    pf2         UUID := 'd2ffdd00-9c0b-4ef8-bb6d-6bb9bd380d02';
    sec_aapl    UUID := 'e1ffdd00-9c0b-4ef8-bb6d-6bb9bd380e01';
    sec_msft    UUID := 'e2ffdd00-9c0b-4ef8-bb6d-6bb9bd380e02';
    sec_googl   UUID := 'e3ffdd00-9c0b-4ef8-bb6d-6bb9bd380e03';
    ven_nyse    UUID := 'f1ffdd00-9c0b-4ef8-bb6d-6bb9bd380f01';
    ven_nasdaq  UUID := 'f2ffdd00-9c0b-4ef8-bb6d-6bb9bd380f02';
BEGIN

    -- Counterparties ------------------------------------------------------------
    INSERT INTO mds.counterparty (id, tenant_id, code, name, bic, lei, country, kyc_status, internal_flag) VALUES
        (cp_pb,    sys_tenant, 'GS_PB',    'Goldman Sachs Prime Brokerage',  'BSCHUS33',  '5493006R95ASuNen9108', 'USA', 'APPROVED', false),
        (cp_dealer1, sys_tenant, 'MS_EQUITY', 'Morgan Stanley Equity Execution', 'M sint US33', '5493005YMTJQS8F8RN18', 'USA', 'APPROVED', false),
        (cp_dealer2, sys_tenant, 'DB_FI',    'Deutsche Bank Fixed Income',      'DEUTDEFF',  '5299007H5WI2EGO9863', 'DEU', 'APPROVED', false)
    ON CONFLICT (tenant_id, code) DO UPDATE SET name = EXCLUDED.name, kyc_status = EXCLUDED.kyc_status;

    -- Accounts ------------------------------------------------------------------
    INSERT INTO mds.account (id, tenant_id, code, name, account_type, base_currency, counterparty_id, is_tradeable) VALUES
        (acct1, sys_tenant, 'ACCT-EQUITY-01', 'Equity Trading Account — US Large Cap', 'MARGIN', 'USD', cp_pb, true),
        (acct2, sys_tenant, 'ACCT-FI-01',     'Fixed Income Account — USD IG Corporate', 'MARGIN', 'USD', cp_pb, true)
    ON CONFLICT (tenant_id, code) DO UPDATE SET name = EXCLUDED.name;

    -- Portfolios ----------------------------------------------------------------
    INSERT INTO mds.portfolio (id, tenant_id, account_id, code, name, mandate, base_currency) VALUES
        (pf1, sys_tenant, acct1, 'PF-GROWTH-US',   'US Growth Equity Portfolio',   'GROWTH', 'USD'),
        (pf2, sys_tenant, acct2, 'PF-INCOME-IG',    'USD IG Corporate Income',     'INCOME', 'USD')
    ON CONFLICT (tenant_id, code) DO UPDATE SET name = EXCLUDED.name;

    -- Securities ----------------------------------------------------------------
    INSERT INTO mds.security_master
        (id, tenant_id, asset_class, symbol, isin, exchange_mic, name, short_name,
         currency_id, lot_size, tick_size, eq_country, eq_sector, eq_market_cap, eq_beta)
    VALUES
        (sec_aapl, sys_tenant, 'EQUITY', 'AAPL', 'US0378331005', 'XNYS',
         'Apple Inc.', 'Apple', 1, 100, 0.01,
         'USA', 'Technology', 3000000000000.00, 1.28),
        (sec_msft, sys_tenant, 'EQUITY', 'MSFT', 'US5949181045', 'XNYS',
         'Microsoft Corporation', 'Microsoft', 1, 100, 0.01,
         'USA', 'Technology', 2800000000000.00, 0.95),
        (sec_googl, sys_tenant, 'EQUITY', 'GOOGL', 'US02079K3059', 'XNAS',
         'Alphabet Inc. Class A', 'Alphabet', 1, 100, 0.01,
         'USA', 'Communication Services', 2000000000000.00, 1.05)
    ON CONFLICT (tenant_id, symbol) DO UPDATE SET name = EXCLUDED.name;

    -- Exchange memberships (venue can trade on exchange) ------------------------
    INSERT INTO mds.exchange_membership (counterparty_id, tenant_id, exchange_mic) VALUES
        (cp_dealer1, sys_tenant, 'XNYS'),
        (cp_dealer1, sys_tenant, 'XNAS'),
        (cp_dealer1, sys_tenant, 'ARCX'),
        (cp_dealer2, sys_tenant, 'XNYS'),
        (cp_dealer2, sys_tenant, 'XSWX')
    ON CONFLICT (tenant_id, counterparty_id, exchange_mic) DO NOTHING;

    -- Venues --------------------------------------------------------------------
    INSERT INTO mds.venue
        (id, tenant_id, counterparty_id, venue_type_id, exchange_mic, venue_code, name,
         supports_market, supports_limit, supports_stop, supports_algo)
    VALUES
        (ven_nyse,  sys_tenant, cp_dealer1, 1, 'XNYS', 'MS_NYSE',  'Morgan Stanley NYSE',  true,  true, true, true),
        (ven_nasdaq, sys_tenant, cp_dealer1, 1, 'XNAS', 'MS_NASDAQ','Morgan Stanley NASDAQ',true, true, true, true)
    ON CONFLICT (tenant_id, venue_code) DO NOTHING;

END $$;

-- ============================================================================
-- SEED DATA — oms operational
-- Realistic order lifecycle: AAPL BUY (filled), MSFT SELL (partial fill)
-- and GOOGL BUY (working) to cover NEW / WORKING / FILLED / PARTIALLY_FILLED states.
-- ============================================================================

-- Disable triggers to avoid default_order_status() check_violation on seed inserts
ALTER TABLE oms.orders      DISABLE TRIGGER ALL;
ALTER TABLE oms.order_slice DISABLE TRIGGER ALL;
ALTER TABLE oms.allocation  DISABLE TRIGGER ALL;
ALTER TABLE oms.settlement  DISABLE TRIGGER ALL;

DO $$
DECLARE
    sys_tenant  UUID := 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';
    acct1       UUID := 'c1ffdd00-9c0b-4ef8-bb6d-6bb9bd380c01';
    pf1         UUID := 'd1ffdd00-9c0b-4ef8-bb6d-6bb9bd380d01';
    sec_aapl    UUID := 'e1ffdd00-9c0b-4ef8-bb6d-6bb9bd380e01';
    sec_msft    UUID := 'e2ffdd00-9c0b-4ef8-bb6d-6bb9bd380e02';
    sec_googl   UUID := 'e3ffdd00-9c0b-4ef8-bb6d-6bb9bd380e03';
    ven_nyse    UUID := 'f1ffdd00-9c0b-4ef8-bb6d-6bb9bd380f01';

    ord_aapl    UUID := '11111111-1111-1111-1111-111111111101';
    ord_msft    UUID := '22222222-2222-2222-2222-222222222201';
    ord_googl   UUID := '33333333-3333-3333-3333-333333333301';

    slc_aapl    UUID := '44444444-4444-4444-4444-444444444401';
    slc_msft    UUID := '55555555-5555-5555-5555-555555555501';
    slc_googl   UUID := '66666666-6666-6666-6666-666666666601';

    exec_aapl1  UUID := '77777777-7777-7777-7777-777777777701';
    exec_aapl2  UUID := '77777777-7777-7777-7777-777777777702';
    exec_msft1  UUID := '88888888-8888-8888-8888-888888888801';
    exec_googl1  UUID := '99999999-9999-9999-9999-999999999901';

    alloc_aapl1 UUID := 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa01';
    alloc_aapl2 UUID := 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa02';
    alloc_msft1 UUID := 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01';
    alloc_googl1 UUID := 'cccccccc-cccc-cccc-cccc-ccccccccccc0';

    lot_aapl1   UUID := 'dddddddd-dddd-dddd-dddd-dddddddddd01';
    lot_aapl2   UUID := 'dddddddd-dddd-dddd-dddd-dddddddddd02';

    -- Reference IDs (from seed data above)
    id_equity       INTEGER := 1;
    id_side_buy     INTEGER := 1;
    id_side_sell    INTEGER := 2;
    id_order_type   INTEGER := 2;  -- LIMIT
    id_tif_day      INTEGER := 1;
    id_status_new   INTEGER := 1;
    id_status_work INTEGER := 5;
    id_status_fill INTEGER := 7;
    id_status_part INTEGER := 6;
    id_curr_usd    INTEGER := 1;
    id_liq_flag_a  INTEGER := 1;  -- Add
    id_liq_flag_r  INTEGER := 2;  -- Remove
    id_evt_created INTEGER := 1;
    id_evt_submitted INTEGER := 2;
    id_evt_acked    INTEGER := 3;
    id_evt_working  INTEGER := 7;
    id_evt_partial  INTEGER := 11;
    id_evt_full     INTEGER := 12;
    id_src_trader   INTEGER := 2;
    id_src_venue    INTEGER := 1;
    id_link_slice   INTEGER := 1;
    id_alloc_pending INTEGER := 1;
    id_alloc_allocd  INTEGER := 2;
    id_settle_pending INTEGER := 1;
BEGIN

    -- ------------------------------------------------------------------
    -- ORDER 1: AAPL BUY 1,000 @ $185.00 LIMIT — FILLED
    -- ------------------------------------------------------------------
    INSERT INTO oms.orders
        (id, tenant_id, client_order_id, portfolio_id, account_id, security_id,
         side, order_type_id, time_in_force_id, asset_class, quantity, limit_price,
         filled_qty, avg_fill_price, status_id, created_by, created_at)
    VALUES
        (ord_aapl, sys_tenant, 'CL-2026-08-24-AAPL-001', pf1, acct1, sec_aapl,
         'BUY', id_order_type, id_tif_day, 'EQUITY',
         1000, 185.00,
         1000, 184.95, id_status_fill,
         'system', '2026-08-24 09:30:15+00')
    ON CONFLICT DO NOTHING;

    -- Slice for AAPL
    INSERT INTO oms.order_slice
        (id, tenant_id, parent_order_id, venue_id, venue_order_id,
         quantity, filled_qty, avg_fill_price, status_id,
         created_by, created_at)
    VALUES
        (slc_aapl, sys_tenant, ord_aapl, ven_nyse, 'MS-AAPL-001',
         1000, 1000, 184.95, id_status_fill,
         'system', '2026-08-24 09:30:15+00')
    ON CONFLICT DO NOTHING;

    -- Order links
    INSERT INTO oms.order_link (tenant_id, link_type_id, parent_order_id, child_order_id) VALUES
        (sys_tenant, id_link_slice, ord_aapl, slc_aapl)
    ON CONFLICT DO NOTHING;

    -- Executions for AAPL (2 prints)
    INSERT INTO oms.execution
        (id, tenant_id, order_id, slice_id, venue_execution_id, qty, price,
         gross_amount, fee, net_amount, price_currency_id, price_type,
         liquidity_flag_id, venue_id, executed_at)
    VALUES
        (exec_aapl1, sys_tenant, ord_aapl, slc_aapl, 'EXEC-AAPL-001',
         600, 184.95, 110970.00, 12.50, 110957.50,
         id_curr_usd, 'LIMIT', id_liq_flag_a, ven_nyse,
         '2026-08-24 09:31:02+00'),
        (exec_aapl2, sys_tenant, ord_aapl, slc_aapl, 'EXEC-AAPL-002',
         400, 184.95, 73980.00, 8.25, 73971.75,
         id_curr_usd, 'LIMIT', id_liq_flag_a, ven_nyse,
         '2026-08-24 09:31:05+00')
    ON CONFLICT DO NOTHING;

    -- Allocations for AAPL executions
    INSERT INTO oms.allocation
        (id, tenant_id, execution_id, account_id, qty, price, net_amount, status_id)
    VALUES
        (alloc_aapl1, sys_tenant, exec_aapl1, acct1, 600, 184.95, 110957.50, id_alloc_allocd),
        (alloc_aapl2, sys_tenant, exec_aapl2, acct1, 400, 184.95, 73971.75, id_alloc_allocd)
    ON CONFLICT DO NOTHING;

    -- Position lots for AAPL (long)
    INSERT INTO oms.position_lots
        (id, tenant_id, account_id, security_id, source_execution_id, source_order_id,
         qty_signed, remaining_qty, cost_basis_ccy, cost_basis_amount, cost_basis_per_unit,
         open_date, open_price, lot_sequence, is_open)
    VALUES
        (lot_aapl1, sys_tenant, acct1, sec_aapl, exec_aapl1, ord_aapl,
         600, 600, 'USD', 110957.50, 184.92917,
         '2026-08-24', 184.95, 1, true),
        (lot_aapl2, sys_tenant, acct1, sec_aapl, exec_aapl2, ord_aapl,
         400, 400, 'USD', 73971.75, 184.929375,
         '2026-08-24', 184.95, 2, true)
    ON CONFLICT DO NOTHING;

    -- Order events for AAPL lifecycle
    INSERT INTO oms.order_event (tenant_id, order_id, slice_id, event_type_id, source_id, actor_id, occurred_at) VALUES
        (sys_tenant, ord_aapl, NULL, id_evt_created,   id_src_trader, 'system', '2026-08-24 09:30:15+00'),
        (sys_tenant, ord_aapl, slc_aapl, id_evt_submitted, id_src_trader, 'system', '2026-08-24 09:30:16+00'),
        (sys_tenant, ord_aapl, slc_aapl, id_evt_acked,    id_src_venue,  'NYSE',   '2026-08-24 09:30:18+00'),
        (sys_tenant, ord_aapl, slc_aapl, id_evt_working,  id_src_venue,  'NYSE',   '2026-08-24 09:30:19+00'),
        (sys_tenant, ord_aapl, slc_aapl, id_evt_partial,  id_src_venue,  'NYSE',   '2026-08-24 09:31:02+00'),
        (sys_tenant, ord_aapl, slc_aapl, id_evt_full,     id_src_venue,  'NYSE',   '2026-08-24 09:31:05+00')
    ON CONFLICT DO NOTHING;

    -- ------------------------------------------------------------------
    -- ORDER 2: MSFT SELL 500 @ $420.00 LIMIT — PARTIALLY_FILLED (200)
    -- ------------------------------------------------------------------
    INSERT INTO oms.orders
        (id, tenant_id, client_order_id, portfolio_id, account_id, security_id,
         side, order_type_id, time_in_force_id, asset_class, quantity, limit_price,
         filled_qty, avg_fill_price, status_id, created_by, created_at)
    VALUES
        (ord_msft, sys_tenant, 'CL-2026-08-25-MSFT-001', pf1, acct1, sec_msft,
         'SELL', id_order_type, id_tif_day, 'EQUITY',
         500, 420.00,
         200, 419.80, id_status_part,
         'trader.chen', '2026-08-25 10:05:00+00')
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.order_slice
        (id, tenant_id, parent_order_id, venue_id, venue_order_id,
         quantity, filled_qty, avg_fill_price, status_id,
         created_by, created_at)
    VALUES
        (slc_msft, sys_tenant, ord_msft, ven_nyse, 'MS-MSFT-001',
         500, 200, 419.80, id_status_part,
         'trader.chen', '2026-08-25 10:05:00+00')
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.order_link (tenant_id, link_type_id, parent_order_id, child_order_id) VALUES
        (sys_tenant, id_link_slice, ord_msft, slc_msft)
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.execution
        (id, tenant_id, order_id, slice_id, venue_execution_id, qty, price,
         gross_amount, fee, net_amount, price_currency_id, price_type,
         liquidity_flag_id, venue_id, executed_at)
    VALUES
        (exec_msft1, sys_tenant, ord_msft, slc_msft, 'EXEC-MSFT-001',
         200, 419.80, 83960.00, 9.50, 83950.50,
         id_curr_usd, 'LIMIT', id_liq_flag_r, ven_nyse,
         '2026-08-25 10:06:14+00')
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.allocation
        (id, tenant_id, execution_id, account_id, qty, price, net_amount, status_id)
    VALUES
        (alloc_msft1, sys_tenant, exec_msft1, acct1, 200, 419.80, 83950.50, id_alloc_allocd)
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.order_event (tenant_id, order_id, slice_id, event_type_id, source_id, actor_id, occurred_at) VALUES
        (sys_tenant, ord_msft, NULL,    id_evt_created,   id_src_trader, 'trader.chen', '2026-08-25 10:05:00+00'),
        (sys_tenant, ord_msft, slc_msft, id_evt_submitted, id_src_trader, 'trader.chen', '2026-08-25 10:05:01+00'),
        (sys_tenant, ord_msft, slc_msft, id_evt_acked,    id_src_venue,  'NYSE',         '2026-08-25 10:05:03+00'),
        (sys_tenant, ord_msft, slc_msft, id_evt_working,  id_src_venue,  'NYSE',         '2026-08-25 10:05:04+00'),
        (sys_tenant, ord_msft, slc_msft, id_evt_partial,  id_src_venue,  'NYSE',         '2026-08-25 10:06:14+00')
    ON CONFLICT DO NOTHING;

    -- ------------------------------------------------------------------
    -- ORDER 3: GOOGL BUY 300 @ $175.00 LIMIT — WORKING (no fills yet)
    -- ------------------------------------------------------------------
    INSERT INTO oms.orders
        (id, tenant_id, client_order_id, portfolio_id, account_id, security_id,
         side, order_type_id, time_in_force_id, asset_class, quantity, limit_price,
         filled_qty, status_id, created_by, created_at)
    VALUES
        (ord_googl, sys_tenant, 'CL-2026-08-26-GOOGL-001', pf1, acct1, sec_googl,
         'BUY', id_order_type, id_tif_day, 'EQUITY',
         300, 175.00,
         0, id_status_work,
         'trader.patel', '2026-08-26 14:22:10+00')
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.order_slice
        (id, tenant_id, parent_order_id, venue_id, venue_order_id,
         quantity, filled_qty, status_id,
         created_by, created_at)
    VALUES
        (slc_googl, sys_tenant, ord_googl, ven_nyse, 'MS-GOOGL-001',
         300, 0, id_status_work,
         'trader.patel', '2026-08-26 14:22:10+00')
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.order_link (tenant_id, link_type_id, parent_order_id, child_order_id) VALUES
        (sys_tenant, id_link_slice, ord_googl, slc_googl)
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.order_event (tenant_id, order_id, slice_id, event_type_id, source_id, actor_id, occurred_at) VALUES
        (sys_tenant, ord_googl, NULL,     id_evt_created,   id_src_trader, 'trader.patel', '2026-08-26 14:22:10+00'),
        (sys_tenant, ord_googl, slc_googl, id_evt_submitted, id_src_trader, 'trader.patel', '2026-08-26 14:22:11+00'),
        (sys_tenant, ord_googl, slc_googl, id_evt_acked,    id_src_venue,  'NYSE',          '2026-08-26 14:22:13+00'),
        (sys_tenant, ord_googl, slc_googl, id_evt_working,  id_src_venue,  'NYSE',          '2026-08-26 14:22:14+00')
    ON CONFLICT DO NOTHING;

    -- ------------------------------------------------------------------
    -- Settlements for filled allocations
    -- ------------------------------------------------------------------
    INSERT INTO oms.settlement
        (tenant_id, execution_id, allocation_id, counterparty_id,
         settlement_type, deliver_ccy, deliver_amount,
         receive_ccy, receive_amount, custodian_account,
         expected_date, status_id)
    SELECT
        sys_tenant, exec_aapl1, alloc_aapl1, NULL,
        'equities', 'USD', 110957.50,
        NULL, NULL, 'GS-SEG-001',
        '2026-08-26'::date, id_settle_pending
    WHERE EXISTS (SELECT 1 FROM oms.allocation WHERE id = alloc_aapl1)
    ON CONFLICT DO NOTHING;

    INSERT INTO oms.settlement
        (tenant_id, execution_id, allocation_id, counterparty_id,
         settlement_type, deliver_ccy, deliver_amount,
         receive_ccy, receive_amount, custodian_account,
         expected_date, status_id)
    SELECT
        sys_tenant, exec_aapl2, alloc_aapl2, NULL,
        'equities', 'USD', 73971.75,
        NULL, NULL, 'GS-SEG-001',
        '2026-08-26'::date, id_settle_pending
    WHERE EXISTS (SELECT 1 FROM oms.allocation WHERE id = alloc_aapl2)
    ON CONFLICT DO NOTHING;

END $$;

-- Refresh the materialized view if it exists
REFRESH MATERIALIZED VIEW oms.current_positions;

-- Re-enable triggers (disabled before the DO block above)
ALTER TABLE oms.orders      ENABLE TRIGGER ALL;
ALTER TABLE oms.order_slice ENABLE TRIGGER ALL;
ALTER TABLE oms.allocation  ENABLE TRIGGER ALL;
ALTER TABLE oms.settlement  ENABLE TRIGGER ALL;
