-- backend/migrations/016_security_master_schema.sql
-- Security Master Gold Copy Schema
-- Entities: issuer_master, security_master, fixed_income_attributes,
--           equity_attributes, fund_attributes, derivative_attributes
-- All tables follow the same bi-temporal + tenant_id/core_id convention
-- used by portfolio_master.

-- ============================================================
-- 1. ISSUER MASTER
--    Legal entities that issue financial instruments.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.issuer_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,       -- NULL = this IS the core record
    issuer_id                VARCHAR(100) NOT NULL,  -- stable business key
    issuer_name              VARCHAR(255) NOT NULL,
    short_name               VARCHAR(100),
    lei                      VARCHAR(20),            -- Legal Entity Identifier (ISO 17442)
    country_of_incorporation VARCHAR(3),             -- ISO-3166 alpha-3
    sector                   VARCHAR(50),
    industry                 VARCHAR(50),
    parent_company_id        UUID,
    rating_composite         VARCHAR(10),
    esg_score                NUMERIC(5,2),
    status                   VARCHAR(20)  NOT NULL DEFAULT 'Active'
                                 CHECK (status IN ('Active','Inactive','Dissolved')),
    confidence_score         INT          NOT NULL DEFAULT 80
                                 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_issuer_master_id_tenant UNIQUE (issuer_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_issuer_master_tenant     ON edm.issuer_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_issuer_master_lei        ON edm.issuer_master(lei) WHERE lei IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_issuer_master_valid      ON edm.issuer_master(valid_from, valid_to);

ALTER TABLE edm.issuer_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS issuer_master_tenant_isolation ON edm.issuer_master;
CREATE POLICY issuer_master_tenant_isolation ON edm.issuer_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 2. SECURITY MASTER (root / core entity)
--    Covers all instrument types. Subtype-specific attributes
--    live in the *_attributes tables below (1-1 with this).
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.security_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Business identifiers
    security_id              VARCHAR(100) NOT NULL,  -- stable internal key
    primary_identifier       VARCHAR(50)  NOT NULL,  -- ISIN or FIGI
    isin                     VARCHAR(12),
    cusip                    VARCHAR(9),
    sedol                    VARCHAR(7),
    figi                     VARCHAR(12),
    ticker                   VARCHAR(20),
    local_ticker             VARCHAR(20),
    ric                      VARCHAR(50),
    bbg_id                   VARCHAR(50),
    vendor_ids               JSONB        NOT NULL DEFAULT '{}',
    -- Descriptive
    security_name            VARCHAR(255) NOT NULL,
    short_name               VARCHAR(100),
    description              TEXT,
    -- Classification
    asset_class              VARCHAR(50)  NOT NULL
                                 CHECK (asset_class IN ('Equity','FixedIncome','Fund','Derivative','FX','Commodity')),
    sub_asset_class          VARCHAR(50),
    instrument_type          VARCHAR(50),
    security_type_schema     VARCHAR(50),
    sector                   VARCHAR(50),
    industry                 VARCHAR(50),
    country_of_issue         VARCHAR(3),             -- ISO-3166 alpha-3
    country_of_risk          VARCHAR(3),
    region                   VARCHAR(20),
    -- Financial
    currency                 VARCHAR(3)   NOT NULL,  -- ISO-4217
    settlement_currency      VARCHAR(3),
    quotation_type           VARCHAR(50),
    -- Dates
    issue_date               DATE,
    first_trade_date         DATE,
    maturity_date            DATE,
    callable_from_date       DATE,
    puttable_from_date       DATE,
    final_maturity_date      DATE,
    -- Links
    issuer_id                UUID REFERENCES edm.issuer_master(id),
    guarantor_id             UUID,
    parent_company_id        UUID,
    -- Market / exchange
    listing_exchange         VARCHAR(20),
    primary_listing_venue    VARCHAR(50),
    exchange_code            VARCHAR(10),
    -- Status & risk
    status                   VARCHAR(20)  NOT NULL DEFAULT 'Active'
                                 CHECK (status IN ('Active','Inactive','Delisted','Matured')),
    liquidity_profile        VARCHAR(20),
    regulatory_classification VARCHAR(50),
    -- Gold copy metadata
    confidence_score         INT          NOT NULL DEFAULT 80
                                 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit / bi-temporal
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_security_master_id_tenant UNIQUE (security_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_security_master_tenant    ON edm.security_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_master_isin      ON edm.security_master(isin) WHERE isin IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_master_cusip     ON edm.security_master(cusip) WHERE cusip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_master_figi      ON edm.security_master(figi) WHERE figi IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_master_ticker    ON edm.security_master(ticker, tenant_id) WHERE ticker IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_master_asset     ON edm.security_master(asset_class, tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_master_valid     ON edm.security_master(valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_security_master_issuer    ON edm.security_master(issuer_id) WHERE issuer_id IS NOT NULL;

ALTER TABLE edm.security_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS security_master_tenant_isolation ON edm.security_master;
CREATE POLICY security_master_tenant_isolation ON edm.security_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 3. FIXED INCOME ATTRIBUTES  (1-1 extension of security_master)
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.fixed_income_attributes (
    security_id              UUID PRIMARY KEY REFERENCES edm.security_master(id) ON DELETE CASCADE,
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    coupon_type              VARCHAR(20)  NOT NULL DEFAULT 'Fixed'
                                 CHECK (coupon_type IN ('Fixed','Floating','Zero','StepUp','PIK')),
    coupon_rate              NUMERIC(12,6),
    coupon_frequency         VARCHAR(20)
                                 CHECK (coupon_frequency IN ('Annual','SemiAnnual','Quarterly','Monthly','AtMaturity')),
    day_count_convention     VARCHAR(20),            -- 30/360, ACT/360, ACT/ACT, etc.
    issue_price              NUMERIC(18,8),
    issue_size               NUMERIC(28,8),
    par_value                NUMERIC(18,8),
    yield_to_maturity        NUMERIC(12,6),          -- derived / sourced
    yield_to_call            NUMERIC(12,6),
    yield_to_worst           NUMERIC(12,6),
    call_schedule            JSONB        NOT NULL DEFAULT '[]',
    put_schedule             JSONB        NOT NULL DEFAULT '[]',
    seniority                VARCHAR(30),            -- Senior, SubordinatedUnsecured, etc.
    secured                  BOOLEAN      NOT NULL DEFAULT false,
    collateral_type          VARCHAR(50),
    rating_agency_ratings    JSONB        NOT NULL DEFAULT '{}', -- {"Moody": "Aa2", "SP": "AA"}
    rating_composite         VARCHAR(10),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fi_attributes_tenant  ON edm.fixed_income_attributes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_fi_attributes_coupon  ON edm.fixed_income_attributes(coupon_type);

ALTER TABLE edm.fixed_income_attributes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS fi_attributes_tenant_isolation ON edm.fixed_income_attributes;
CREATE POLICY fi_attributes_tenant_isolation ON edm.fixed_income_attributes
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 4. EQUITY ATTRIBUTES  (1-1 extension of security_master)
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.equity_attributes (
    security_id              UUID PRIMARY KEY REFERENCES edm.security_master(id) ON DELETE CASCADE,
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    share_class              VARCHAR(10),            -- A, B, C etc.
    shares_outstanding       NUMERIC(28,8),
    free_float               NUMERIC(10,4),          -- as a percentage
    dividend_yield           NUMERIC(10,4),
    dividend_frequency       VARCHAR(20),            -- Quarterly, Annual, etc.
    corporate_action_policy  UUID,                   -- link to CA rules
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_eq_attributes_tenant  ON edm.equity_attributes(tenant_id);

ALTER TABLE edm.equity_attributes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS eq_attributes_tenant_isolation ON edm.equity_attributes;
CREATE POLICY eq_attributes_tenant_isolation ON edm.equity_attributes
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 5. FUND ATTRIBUTES  (1-1 extension of security_master)
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.fund_attributes (
    security_id              UUID PRIMARY KEY REFERENCES edm.security_master(id) ON DELETE CASCADE,
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    fund_type                VARCHAR(50)  NOT NULL
                                 CHECK (fund_type IN ('MutualFund','ETF','HedgeFund','PrivateEquity','REIT','MoneyMarket','Other')),
    domicile                 VARCHAR(3),             -- ISO-3166 alpha-3
    management_company       VARCHAR(255),
    administrator            VARCHAR(255),
    custodian                VARCHAR(255),
    total_expense_ratio      NUMERIC(8,4),
    management_fee           NUMERIC(8,4),
    performance_fee          NUMERIC(8,4),
    distribution_policy      VARCHAR(20)
                                 CHECK (distribution_policy IN ('Accumulating','Distributing','Mixed')),
    prospectus_link          TEXT,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fund_attributes_tenant    ON edm.fund_attributes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_fund_attributes_type      ON edm.fund_attributes(fund_type);

ALTER TABLE edm.fund_attributes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS fund_attributes_tenant_isolation ON edm.fund_attributes;
CREATE POLICY fund_attributes_tenant_isolation ON edm.fund_attributes
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 6. DERIVATIVE ATTRIBUTES  (1-1 extension of security_master)
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.derivative_attributes (
    security_id              UUID PRIMARY KEY REFERENCES edm.security_master(id) ON DELETE CASCADE,
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    underlier_security_id    UUID REFERENCES edm.security_master(id),
    underlier_type           VARCHAR(30)
                                 CHECK (underlier_type IN ('Index','SingleName','Basket','Rate','Currency','Commodity')),
    contract_size            NUMERIC(28,8),
    contract_month           VARCHAR(10),
    strike_price             NUMERIC(18,8),
    option_type              VARCHAR(10)
                                 CHECK (option_type IN ('Call','Put')),
    exercise_style           VARCHAR(20)
                                 CHECK (exercise_style IN ('American','European','Bermudan')),
    settlement_type          VARCHAR(20)
                                 CHECK (settlement_type IN ('Physical','Cash')),
    expiry_date              DATE,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deriv_attributes_tenant      ON edm.derivative_attributes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_deriv_attributes_underlier   ON edm.derivative_attributes(underlier_security_id);

ALTER TABLE edm.derivative_attributes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS deriv_attributes_tenant_isolation ON edm.derivative_attributes;
CREATE POLICY deriv_attributes_tenant_isolation ON edm.derivative_attributes
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 7. SECURITY GOLD TRACE
--    Per-field lineage for every Security gold copy run.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.security_gold_trace (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    security_id              UUID         NOT NULL REFERENCES edm.security_master(id) ON DELETE CASCADE,
    run_id                   UUID         NOT NULL,
    entity_type              VARCHAR(50)  NOT NULL DEFAULT 'security',
    field_name               VARCHAR(100) NOT NULL,
    chosen_value             TEXT,
    chosen_source            VARCHAR(100),
    survivorship_rule        VARCHAR(100),
    rejected_sources         JSONB        NOT NULL DEFAULT '[]',
    dq_rules_passed          TEXT[]       NOT NULL DEFAULT '{}',
    dq_rules_failed          TEXT[]       NOT NULL DEFAULT '{}',
    confidence_contribution  INT          NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sec_gold_trace_security ON edm.security_gold_trace(security_id);
CREATE INDEX IF NOT EXISTS idx_sec_gold_trace_run      ON edm.security_gold_trace(run_id);
CREATE INDEX IF NOT EXISTS idx_sec_gold_trace_tenant   ON edm.security_gold_trace(tenant_id);

ALTER TABLE edm.security_gold_trace ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS security_gold_trace_isolation ON edm.security_gold_trace;
CREATE POLICY security_gold_trace_isolation ON edm.security_gold_trace
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));
