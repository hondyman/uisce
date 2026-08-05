-- ============================================================================
-- 0003_mds.sql
-- Master Data tables: counterparty, account, portfolio, security_master.
-- Run after 0002_ref_tables.sql
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- mds.counterparty
-- --------------------------------------------------------------------------
CREATE TABLE mds.counterparty (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    code            VARCHAR(30) NOT NULL UNIQUE,
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

CREATE INDEX idx_counterparty_tenant ON mds.counterparty (tenant_id) WHERE is_active = true;
CREATE INDEX idx_counterparty_lei    ON mds.counterparty (lei) WHERE lei IS NOT NULL;
CREATE INDEX idx_counterparty_kyc   ON mds.counterparty (tenant_id, kyc_status);

-- --------------------------------------------------------------------------
-- mds.exchange_membership  (which venues a counterparty can trade on)
-- --------------------------------------------------------------------------
CREATE TABLE mds.exchange_membership (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    counterparty_id UUID        NOT NULL REFERENCES mds.counterparty(id),
    exchange_mic    CHAR(4)     NOT NULL REFERENCES ref.exchange(mic),
    is_active       BOOLEAN     DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_exchange_membership UNIQUE (tenant_id, counterparty_id, exchange_mic)
);

CREATE INDEX idx_exch_mem_cp   ON mds.exchange_membership (counterparty_id);
CREATE INDEX idx_exch_mem_mic ON mds.exchange_membership (exchange_mic);

-- --------------------------------------------------------------------------
-- mds.account
-- --------------------------------------------------------------------------
CREATE TABLE mds.account (
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

CREATE INDEX idx_account_tenant   ON mds.account (tenant_id) WHERE is_active = true;
CREATE INDEX idx_account_cp       ON mds.account (counterparty_id) WHERE counterparty_id IS NOT NULL;
CREATE INDEX idx_account_currency ON mds.account (tenant_id, base_currency);

-- --------------------------------------------------------------------------
-- mds.portfolio
-- --------------------------------------------------------------------------
CREATE TABLE mds.portfolio (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    account_id      UUID        NOT NULL REFERENCES mds.account(id),
    code            VARCHAR(30) NOT NULL,
    name            VARCHAR(200),
    mandate         VARCHAR(50),  -- e.g. 'GROWTH','INCOME','BALANCED','ALPHA','LIQUIDITY'
    base_currency   CHAR(3)    NOT NULL REFERENCES ref.currency(iso3_code),
    risk_model_id   UUID,
    benchmark_id    UUID,         -- FK to security_master for benchmark
    is_active       BOOLEAN     DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_portfolio_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX idx_portfolio_tenant  ON mds.portfolio (tenant_id) WHERE is_active = true;
CREATE INDEX idx_portfolio_account ON mds.portfolio (account_id);
CREATE INDEX idx_portfolio_mandate ON mds.portfolio (tenant_id, mandate) WHERE mandate IS NOT NULL;

-- --------------------------------------------------------------------------
-- mds.security_master
-- Multi-asset: asset_class discriminator + typed columns per asset class.
-- Unused columns for a given asset_class are NULL.
-- --------------------------------------------------------------------------
CREATE TABLE mds.security_master (
    id                    UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id             UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    asset_class           VARCHAR(30) NOT NULL REFERENCES ref.asset_class(code),

    -- Identifiers
    symbol                VARCHAR(30) NOT NULL,
    isin                  CHAR(12),
    cusip                 VARCHAR(9),
    sedol                 VARCHAR(7),
    ric                   VARCHAR(20),
    bloomberg_ticker      VARCHAR(30),
    exchange_mic          CHAR(4)     REFERENCES ref.exchange(mic),
    exchange_symbol       VARCHAR(30),

    -- Common descriptive
    name                  VARCHAR(300),
    short_name            VARCHAR(100),
    currency_id           INTEGER     REFERENCES ref.currency(id),
    lot_size              NUMERIC(18,6),
    tick_size             NUMERIC(18,8),

    -- EQUITY fields
    eq_country            VARCHAR(3),
    eq_sector             VARCHAR(100),
    eq_industry           VARCHAR(100),
    eq_market_cap         NUMERIC(20,2),
    eq_dividend_yield     NUMERIC(10,6),
    eq_beta               NUMERIC(10,4),
    eq_free_float         NUMERIC(5,4),

    -- FIXED_INCOME fields
    fi_coupon_rate        NUMERIC(10,6),
    fi_coupon_freq        INTEGER,           -- 1=annual, 2=semi, 4=quarterly
    fi_day_count          VARCHAR(20),       -- '30/360','ACT/ACT','ACT/365','ACT/360'
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

    -- FX fields
    fx_ccy_base           CHAR(3)   REFERENCES ref.currency(iso3_code),
    fx_ccy_quote           CHAR(3)   REFERENCES ref.currency(iso3_code),
    fx_pip_precision      INTEGER   DEFAULT 4,
    fx_value_date_conv    VARCHAR(20),       -- 'T+0','T+1','T+2'

    -- DERIVATIVE / FUTURE fields
    deriv_underlying_id    UUID      REFERENCES mds.security_master(id),
    deriv_contract_size    NUMERIC(20,6),
    deriv_tick_value      NUMERIC(20,6),
    deriv_expiry_date     DATE,
    deriv_settlement      VARCHAR(20),        -- 'CASH','PHYSICAL'
    deriv_first_trade     DATE,
    deriv_last_trade      DATE,

    -- OPTION fields (extends derivative)
    opt_option_type       VARCHAR(10) CHECK (opt_option_type IN ('CALL','PUT')),
    opt_strike_price      NUMERIC(20,8),
    opt_expiry_date       DATE,
    opt_settlement_method VARCHAR(20),
    opt_european_flag     BOOLEAN   DEFAULT true,

    -- SWAP fields
    swap_leg1_ccy         CHAR(3)  REFERENCES ref.currency(iso3_code),
    swap_leg2_ccy         CHAR(3)  REFERENCES ref.currency(iso3_code),
    swap_leg1_pay_rec     VARCHAR(5),        -- 'PAY','REC'
    swap_leg2_pay_rec     VARCHAR(5),
    swap_rate_index1      VARCHAR(30),
    swap_rate_index2      VARCHAR(30),
    swap_notional_ccy     CHAR(3)  REFERENCES ref.currency(iso3_code),
    swap_maturity_date    DATE,

    -- Operational
    is_active             BOOLEAN   DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_security_master_tenant_symbol UNIQUE (tenant_id, symbol),
    CONSTRAINT uq_security_master_isin         UNIQUE (tenant_id, isin)
);

CREATE INDEX idx_security_tenant     ON mds.security_master (tenant_id) WHERE is_active = true;
CREATE INDEX idx_security_asset_cl  ON mds.security_master (tenant_id, asset_class);
CREATE INDEX idx_security_exchange  ON mds.security_master (exchange_mic) WHERE exchange_mic IS NOT NULL;
CREATE INDEX idx_security_isin      ON mds.security_master (isin)         WHERE isin IS NOT NULL;
CREATE INDEX idx_security_ric       ON mds.security_master (ric)          WHERE ric IS NOT NULL;
CREATE UNIQUE INDEX idx_security_isin_uniq ON mds.security_master (tenant_id, isin) WHERE isin IS NOT NULL;
CREATE INDEX idx_security_underlying ON mds.security_master (deriv_underlying_id) WHERE deriv_underlying_id IS NOT NULL;
CREATE INDEX idx_security_maturity  ON mds.security_master (fi_maturity_date)   WHERE fi_maturity_date IS NOT NULL;
CREATE INDEX idx_security_opt_exp   ON mds.security_master (opt_expiry_date)     WHERE opt_expiry_date IS NOT NULL;
CREATE INDEX idx_security_deriv_exp ON mds.security_master (deriv_expiry_date)   WHERE deriv_expiry_date IS NOT NULL;

-- --------------------------------------------------------------------------
-- mds.venue  (venue / broker / prime broker / executing broker)
-- --------------------------------------------------------------------------
CREATE TABLE mds.venue (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    counterparty_id     UUID        NOT NULL REFERENCES mds.counterparty(id),
    venue_type_id       INTEGER     NOT NULL REFERENCES ref.venue_type(id),
    exchange_mic        CHAR(4)     REFERENCES ref.exchange(mic),
    venue_code          VARCHAR(30) NOT NULL,  -- e.g. 'GS_TW','MS_INT','COINBASE'
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

CREATE INDEX idx_venue_cp     ON mds.venue (counterparty_id);
CREATE INDEX idx_venue_mic    ON mds.venue (exchange_mic) WHERE exchange_mic IS NOT NULL;

COMMIT;
