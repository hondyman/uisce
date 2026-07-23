-- backend/migrations/011_portfolio_master.sql
-- Portfolio Master Gold Copy Schema
-- Adds: source_registry, portfolio_source, portfolio_golden tables
-- Extends: edm.source_preferences with account_type dimension

-- ============================================================
-- 1. SOURCE REGISTRY
--    Canonical catalog of external data source systems.
--    Each row represents a vendor (Bloomberg, Refinitiv, etc.)
--    and captures its capabilities, coverage, and confidence base.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.source_registry (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_name      VARCHAR(100) NOT NULL,
    source_code      VARCHAR(20)  NOT NULL,               -- SHORT code (e.g. BBG, RFN, SP, FS)
    source_type      VARCHAR(50)  NOT NULL DEFAULT 'API', -- API | FILE | DATABASE
    endpoint_url     TEXT,
    is_active        BOOLEAN      NOT NULL DEFAULT true,
    priority_score   INT          NOT NULL DEFAULT 3
                         CHECK (priority_score BETWEEN 1 AND 5),
    confidence_base  INT          NOT NULL DEFAULT 80
                         CHECK (confidence_base BETWEEN 0 AND 100),
    account_types    TEXT[]       NOT NULL DEFAULT '{}',  -- retail, institutional, etc.
    asset_classes    TEXT[]       NOT NULL DEFAULT '{}',  -- EQUITY, FIXED_INCOME, etc.
    regions          TEXT[]       NOT NULL DEFAULT '{}',  -- GLOBAL, NAM, EMEA, APAC, etc.
    metadata         JSONB        NOT NULL DEFAULT '{}',
    tenant_id        UUID         NOT NULL,
    core_id          UUID REFERENCES edm.source_registry(id),  -- NULL = core record
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by       UUID         NOT NULL,
    updated_by       UUID,
    CONSTRAINT uq_source_registry_name_tenant UNIQUE (source_name, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_source_registry_tenant  ON edm.source_registry(tenant_id);
CREATE INDEX IF NOT EXISTS idx_source_registry_active  ON edm.source_registry(is_active);
CREATE INDEX IF NOT EXISTS idx_source_registry_code    ON edm.source_registry(source_code);

ALTER TABLE edm.source_registry ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS source_registry_tenant_isolation ON edm.source_registry;
CREATE POLICY source_registry_tenant_isolation ON edm.source_registry
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 2. EXTEND source_preferences WITH account_type
--    Adds the account_type dimension to existing source preference
--    records. Defaults to 'GLOBAL' so existing rows remain valid.
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'edm'
          AND table_name   = 'source_preferences'
          AND column_name  = 'account_type'
    ) THEN
        ALTER TABLE edm.source_preferences
            ADD COLUMN account_type VARCHAR(50) NOT NULL DEFAULT 'GLOBAL';
    END IF;
END
$$;

-- Update the business-object/term/region index to include account_type
DROP INDEX IF EXISTS idx_source_prefs_bo;
CREATE INDEX IF NOT EXISTS idx_source_prefs_bo
    ON edm.source_preferences(business_object, semantic_term, region, account_type);

-- ============================================================
-- 3. PORTFOLIO SOURCE (Staging / Raw Ingestion)
--    One row per (portfolio_id, security_id, source) per ingest.
--    This is the staging layer before golden record generation.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.portfolio_source (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID         NOT NULL,
    source_registry_id  UUID         NOT NULL REFERENCES edm.source_registry(id),
    portfolio_id        VARCHAR(100) NOT NULL,
    account_type        VARCHAR(50)  NOT NULL
                            CHECK (account_type IN ('retail','institutional','private_wealth','private_markets')),
    security_id         VARCHAR(100) NOT NULL,
    security_name       VARCHAR(255),
    quantity            NUMERIC(28,8),
    price               NUMERIC(28,8),
    market_value        NUMERIC(28,8),
    currency            VARCHAR(10)  NOT NULL DEFAULT 'USD',
    asset_class         VARCHAR(50),
    country             VARCHAR(50),
    region              VARCHAR(50),
    confidence_score    INT          NOT NULL DEFAULT 80
                            CHECK (confidence_score BETWEEN 0 AND 100),
    ingestion_job_id    UUID,
    source_timestamp    TIMESTAMPTZ,
    raw_payload         JSONB        NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_from          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to            TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_portfolio_source_tenant    ON edm.portfolio_source(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_source_portfolio ON edm.portfolio_source(portfolio_id, account_type);
CREATE INDEX IF NOT EXISTS idx_portfolio_source_security  ON edm.portfolio_source(security_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_source_source    ON edm.portfolio_source(source_registry_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_source_valid     ON edm.portfolio_source(valid_from, valid_to);

ALTER TABLE edm.portfolio_source ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS portfolio_source_tenant_isolation ON edm.portfolio_source;
CREATE POLICY portfolio_source_tenant_isolation ON edm.portfolio_source
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 4. PORTFOLIO GOLDEN (Golden Record)
--    One authoritative row per (tenant, portfolio_id, security_id).
--    Synthesised from portfolio_source using source preferences.
--    Bi-temporal: valid_from / valid_to tracks record history.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.portfolio_golden (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    portfolio_id     VARCHAR(100) NOT NULL,
    account_type     VARCHAR(50)  NOT NULL
                         CHECK (account_type IN ('retail','institutional','private_wealth','private_markets')),
    security_id      VARCHAR(100) NOT NULL,
    security_name    VARCHAR(255),
    quantity         NUMERIC(28,8) NOT NULL,
    price            NUMERIC(28,8) NOT NULL,
    market_value     NUMERIC(28,8) NOT NULL,   -- Derived: quantity * price
    currency         VARCHAR(10)  NOT NULL DEFAULT 'USD',
    asset_class      VARCHAR(50),
    country          VARCHAR(50),
    region           VARCHAR(50),
    confidence_score INT NOT NULL DEFAULT 80
                         CHECK (confidence_score BETWEEN 0 AND 100),
    -- Tracks which source supplied each field: {"price": "Bloomberg", "quantity": "FactSet"}
    source_systems   JSONB NOT NULL DEFAULT '{}',
    -- Semantic lineage: which source_registry rows contributed
    contributing_sources UUID[] NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       UUID NOT NULL,
    updated_by       UUID,
    valid_from       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to         TIMESTAMPTZ,
    CONSTRAINT uq_portfolio_golden_position
        UNIQUE (tenant_id, portfolio_id, security_id, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_golden_tenant    ON edm.portfolio_golden(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_golden_portfolio ON edm.portfolio_golden(portfolio_id, account_type);
CREATE INDEX IF NOT EXISTS idx_portfolio_golden_security  ON edm.portfolio_golden(security_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_golden_valid     ON edm.portfolio_golden(valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_portfolio_golden_current
    ON edm.portfolio_golden(tenant_id, account_type) WHERE valid_to IS NULL;

ALTER TABLE edm.portfolio_golden ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS portfolio_golden_tenant_isolation ON edm.portfolio_golden;
CREATE POLICY portfolio_golden_tenant_isolation ON edm.portfolio_golden
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));
