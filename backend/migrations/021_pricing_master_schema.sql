-- backend/migrations/021_pricing_master_schema.sql
-- Pricing Master Gold Copy Schema
-- Entities: price_master, fx_rate_master, curve_master, vol_surface_master
-- All tables follow the same bi-temporal + tenant_id/core_id convention
-- used by Security Master and Portfolio Master.

-- ============================================================
-- 1. PRICE MASTER
--    Gold-copy point-in-time price for a security from a given source.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.price_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Business identifiers / cluster key
    security_id              UUID         NOT NULL,   -- FK to edm.security_master
    price_type               VARCHAR(50)  NOT NULL    -- e.g. 'Close', 'Bid', 'Ask', 'Mid', 'NAV'
                                 CHECK (price_type IN ('Close','Open','Bid','Ask','Mid','Last','NAV','Settlement','Other')),
    price_date               DATE         NOT NULL,
    -- Core price fields
    price_value              NUMERIC(28,10) NOT NULL
                                 CHECK (price_value > 0),
    price_time               TIMESTAMPTZ,
    price_currency           VARCHAR(3)   NOT NULL,   -- ISO-4217
    fx_rate_to_base          NUMERIC(18,8),
    -- Provenance
    price_source             VARCHAR(100) NOT NULL,   -- 'Bloomberg', 'Refinitiv', etc.
    -- Quality flags
    price_confidence         INT          NOT NULL DEFAULT 80
                                 CHECK (price_confidence BETWEEN 0 AND 100),
    is_composite_price       BOOLEAN      NOT NULL DEFAULT false,
    composite_method         VARCHAR(50),
    is_stale_price           BOOLEAN      NOT NULL DEFAULT false,
    stale_reason             TEXT,
    -- Source provenance JSONB
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit / bi-temporal
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_price_master_cluster UNIQUE (security_id, price_type, price_date, price_source, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_price_master_tenant      ON edm.price_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_price_master_security    ON edm.price_master(security_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_price_master_date        ON edm.price_master(price_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_price_master_type        ON edm.price_master(price_type, tenant_id);
CREATE INDEX IF NOT EXISTS idx_price_master_valid       ON edm.price_master(valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_price_master_stale       ON edm.price_master(is_stale_price) WHERE is_stale_price = true;

ALTER TABLE edm.price_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS price_master_tenant_isolation ON edm.price_master;
CREATE POLICY price_master_tenant_isolation ON edm.price_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 2. FX RATE MASTER
--    FX spot/forward rate between two currencies.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.fx_rate_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Cluster key
    base_currency            VARCHAR(3)   NOT NULL,   -- ISO-4217
    quote_currency           VARCHAR(3)   NOT NULL,   -- ISO-4217
    fx_rate_date             DATE         NOT NULL,
    fx_tenor                 VARCHAR(20)  NOT NULL DEFAULT 'Spot'
                                 CHECK (fx_tenor IN ('Spot','TN','SN','1W','2W','1M','2M','3M','6M','9M','12M','2Y','5Y','10Y')),
    -- Core fields
    fx_rate                  NUMERIC(18,8) NOT NULL
                                 CHECK (fx_rate > 0),
    fx_source                VARCHAR(100) NOT NULL,
    fx_forward_points        NUMERIC(12,6),
    -- Quality
    fx_confidence            INT          NOT NULL DEFAULT 80
                                 CHECK (fx_confidence BETWEEN 0 AND 100),
    -- Source provenance
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit / bi-temporal
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_fx_rate_master_cluster UNIQUE (base_currency, quote_currency, fx_rate_date, fx_tenor, fx_source, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_fx_rate_master_tenant    ON edm.fx_rate_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_fx_rate_master_pair      ON edm.fx_rate_master(base_currency, quote_currency, tenant_id);
CREATE INDEX IF NOT EXISTS idx_fx_rate_master_date      ON edm.fx_rate_master(fx_rate_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_fx_rate_master_valid     ON edm.fx_rate_master(valid_from, valid_to);

ALTER TABLE edm.fx_rate_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS fx_rate_master_tenant_isolation ON edm.fx_rate_master;
CREATE POLICY fx_rate_master_tenant_isolation ON edm.fx_rate_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 3. CURVE MASTER
--    Yield/discount/credit curve for pricing and risk.
--    tenor_points: [{"tenor":"3M","rate":0.0525}, ...]
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.curve_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Cluster key
    curve_type               VARCHAR(50)  NOT NULL   -- 'OIS','LIBOR','Treasury','CreditSpread','Discount'
                                 CHECK (curve_type IN ('OIS','LIBOR','SOFR','Treasury','CreditSpread','Discount','ForwardRate','HazardRate','Other')),
    curve_currency           VARCHAR(3)   NOT NULL,
    curve_as_of_date         DATE         NOT NULL,
    -- Core fields
    curve_source             VARCHAR(100) NOT NULL DEFAULT 'MarketDataVendor',
    curve_tenor_points       JSONB        NOT NULL DEFAULT '[]',  -- [{tenor, rate, discount_factor}]
    curve_interpolation      VARCHAR(50)  DEFAULT 'Linear',
    curve_extrapolation      VARCHAR(50)  DEFAULT 'Flat',
    -- Quality
    curve_confidence         INT          NOT NULL DEFAULT 80
                                 CHECK (curve_confidence BETWEEN 0 AND 100),
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit / bi-temporal
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_curve_master_cluster UNIQUE (curve_type, curve_currency, curve_as_of_date, curve_source, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_curve_master_tenant      ON edm.curve_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_curve_master_type        ON edm.curve_master(curve_type, curve_currency, tenant_id);
CREATE INDEX IF NOT EXISTS idx_curve_master_date        ON edm.curve_master(curve_as_of_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_curve_master_valid       ON edm.curve_master(valid_from, valid_to);

ALTER TABLE edm.curve_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS curve_master_tenant_isolation ON edm.curve_master;
CREATE POLICY curve_master_tenant_isolation ON edm.curve_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 4. VOL SURFACE MASTER
--    Volatility surface for options and derivatives.
--    vol_grid: {"strikes":[0.9,1.0,1.1], "tenors":["1M","3M"], "vols":[[0.18,0.20],[0.17,0.19]]}
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.vol_surface_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    core_id                  UUID,
    -- Cluster key
    underlier_security_id    UUID         NOT NULL,  -- FK to edm.security_master
    vol_surface_type         VARCHAR(50)  NOT NULL
                                 CHECK (vol_surface_type IN ('Equity','Rates','FX','Credit','Commodity','Other')),
    vol_as_of_date           DATE         NOT NULL,
    vol_source               VARCHAR(100) NOT NULL DEFAULT 'MarketDataVendor',
    -- Core fields
    vol_grid                 JSONB        NOT NULL DEFAULT '{}',  -- {strikes, tenors, vols}
    vol_interpolation        VARCHAR(50)  DEFAULT 'BiLinear',
    vol_extrapolation        VARCHAR(50)  DEFAULT 'Flat',
    -- Quality
    vol_confidence           INT          NOT NULL DEFAULT 80
                                 CHECK (vol_confidence BETWEEN 0 AND 100),
    source_systems           JSONB        NOT NULL DEFAULT '{}',
    -- Audit / bi-temporal
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by               UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_vol_surface_master_cluster UNIQUE (underlier_security_id, vol_surface_type, vol_as_of_date, vol_source, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_vol_surface_master_tenant     ON edm.vol_surface_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_vol_surface_master_underlier  ON edm.vol_surface_master(underlier_security_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_vol_surface_master_date       ON edm.vol_surface_master(vol_as_of_date, tenant_id);
CREATE INDEX IF NOT EXISTS idx_vol_surface_master_valid      ON edm.vol_surface_master(valid_from, valid_to);

ALTER TABLE edm.vol_surface_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS vol_surface_master_tenant_isolation ON edm.vol_surface_master;
CREATE POLICY vol_surface_master_tenant_isolation ON edm.vol_surface_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 5. PRICE GOLD TRACE
--    Per-field lineage for every Pricing survivorship run.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.price_gold_trace (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    entity_type              VARCHAR(50)  NOT NULL   -- 'price','fx_rate','curve','vol_surface'
                                 CHECK (entity_type IN ('price','fx_rate','curve','vol_surface')),
    entity_id                UUID         NOT NULL,
    run_id                   UUID         NOT NULL,
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

CREATE INDEX IF NOT EXISTS idx_price_gold_trace_entity  ON edm.price_gold_trace(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_price_gold_trace_run     ON edm.price_gold_trace(run_id);
CREATE INDEX IF NOT EXISTS idx_price_gold_trace_tenant  ON edm.price_gold_trace(tenant_id);

ALTER TABLE edm.price_gold_trace ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS price_gold_trace_isolation ON edm.price_gold_trace;
CREATE POLICY price_gold_trace_isolation ON edm.price_gold_trace
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));
