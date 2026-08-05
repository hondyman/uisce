-- ============================================================================
-- 0002_ref_tables.sql
-- Reference / lookup tables for the trade order system.
-- Run after 0001_init_schemas.sql
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- ref.asset_class
-- --------------------------------------------------------------------------
CREATE TABLE ref.asset_class (
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

-- --------------------------------------------------------------------------
-- ref.currency
-- --------------------------------------------------------------------------
CREATE TABLE ref.currency (
    id              SERIAL PRIMARY KEY,
    iso3_code       CHAR(3)      NOT NULL UNIQUE,
    numeric_code    CHAR(3),
    name            VARCHAR(100) NOT NULL,
    minor_unit      INTEGER      DEFAULT 2,
    is_active       BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
INSERT INTO ref.currency (iso3_code, numeric_code, name) VALUES
    ('USD','840','US Dollar'),('EUR','978','Euro'),('GBP','826','British Pound'),
    ('JPY','392','Japanese Yen'),('CHF','756','Swiss Franc'),('AUD','036','Australian Dollar'),
    ('CAD','124','Canadian Dollar'),('HKD','344','Hong Kong Dollar'),('SGD','702','Singapore Dollar'),
    ('SEK','752','Swedish Krona'),('NOK','578','Norwegian Krone'),('DKK','208','Danish Krone'),
    ('NZD','554','New Zealand Dollar'),('CNY','156','Chinese Yuan'),('INR','356','Indian Rupee'),
    ('BRL','076','Brazilian Real'),('MXN','484','Mexican Peso'),('ZAR','710','South African Rand')
ON CONFLICT (iso3_code) DO NOTHING;

-- --------------------------------------------------------------------------
-- ref.exchange
-- --------------------------------------------------------------------------
CREATE TABLE ref.exchange (
    id          SERIAL PRIMARY KEY,
    mic         CHAR(4)       NOT NULL UNIQUE,
    name        VARCHAR(200)  NOT NULL,
    country     VARCHAR(100),
    timezone    VARCHAR(50),
    currency_id INTEGER REFERENCES ref.currency(id),
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
INSERT INTO ref.exchange (mic, name, country, timezone) VALUES
    ('XNAS','Nasdaq','US','America/New_York'),
    ('XNYS','New York Stock Exchange','US','America/New_York'),
    ('ARCX','NYSE Arca','US','America/New_York'),
    ('BATS','Cboe BATS','US','America/Chicago'),
    ('EDGX','Cboe EDGX','US','America/Chicago'),
    ('CHIU','Chicago Stock Exchange','US','America/Chicago'),
    ('XLON','London Stock Exchange','UK','Europe/London'),
    ('XSWX','SIX Swiss Exchange','CH','Europe/Zurich'),
    ('XTKS','Tokyo Stock Exchange','JP','Asia/Tokyo'),
    ('XHKG','Hong Kong Stock Exchange','HK','Asia/Hong_Kong'),
    ('XSES','Singapore Exchange','SG','Asia/Singapore'),
    ('XASX','Australian Securities Exchange','AU','Australia/Sydney'),
    ('XTSE','Toronto Stock Exchange','CA','America/Toronto'),
    ('XFRA','Deutsche Boerse','DE','Europe/Berlin'),
    ('MILX','Borsa Italiana','IT','Europe/Rome')
ON CONFLICT (mic) DO NOTHING;

-- --------------------------------------------------------------------------
-- ref.order_type
-- --------------------------------------------------------------------------
CREATE TABLE ref.order_type (
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

-- --------------------------------------------------------------------------
-- ref.time_in_force
-- --------------------------------------------------------------------------
CREATE TABLE ref.time_in_force (
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

-- --------------------------------------------------------------------------
-- ref.side
-- --------------------------------------------------------------------------
CREATE TABLE ref.side (
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

-- --------------------------------------------------------------------------
-- ref.order_status
-- --------------------------------------------------------------------------
CREATE TABLE ref.order_status (
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
    ('FAILED',           'Technical / operational failure'),
    ('SUSPENDED',        'Suspended (regulatory / risk freeze)')
ON CONFLICT (code) DO NOTHING;

-- --------------------------------------------------------------------------
-- ref.order_event_type
-- --------------------------------------------------------------------------
CREATE TABLE ref.order_event_type (
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
    ('AMENDED',             'Order amended (price / qty change)'),
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

-- --------------------------------------------------------------------------
-- ref.event_source
-- --------------------------------------------------------------------------
CREATE TABLE ref.event_source (
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

-- --------------------------------------------------------------------------
-- ref.order_link_type
-- --------------------------------------------------------------------------
CREATE TABLE ref.order_link_type (
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

-- --------------------------------------------------------------------------
-- ref.venue_type
-- --------------------------------------------------------------------------
CREATE TABLE ref.venue_type (
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

-- --------------------------------------------------------------------------
-- ref.allocation_status
-- --------------------------------------------------------------------------
CREATE TABLE ref.allocation_status (
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

-- --------------------------------------------------------------------------
-- ref.settlement_status
-- --------------------------------------------------------------------------
CREATE TABLE ref.settlement_status (
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

-- --------------------------------------------------------------------------
-- ref.liquidity_flag
-- --------------------------------------------------------------------------
CREATE TABLE ref.liquidity_flag (
    id          SERIAL PRIMARY KEY,
    code        CHAR(1)      NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- SEC Rule 605 / MiFID II style
INSERT INTO ref.liquidity_flag (code, description) VALUES
    ('A', 'Add — initiator / maker liquidity (order added to book)'),
    ('R', 'Remove — taker liquidity (order removed from book)'),
    ('N', 'Neither — non-displayable / hidden order'),
    ('X', 'Auction — print from auction process')
ON CONFLICT (code) DO NOTHING;

COMMIT;
