-- 20261118_001_mdm_source_scoring_schema.up.sql
-- MDM License Source Scoring, Evaluation Mart, and Displacement Analytics

CREATE SCHEMA IF NOT EXISTS mdm_eval;

-- 1. Tolerance Registry (Reference rules stored in data, not code)
CREATE TABLE IF NOT EXISTS mdm_eval.attribute_tolerance (
    attribute_code       VARCHAR(64) PRIMARY KEY,
    tier                 SMALLINT NOT NULL CHECK (tier IN (1, 2, 3)),
    match_type           VARCHAR(24) NOT NULL CHECK (match_type IN ('EXACT', 'FUZZY_JARO', 'NUMERIC_BP', 'NUMERIC_PCT', 'DATE_LAG')),
    tolerance_val        NUMERIC(12, 6) DEFAULT 0,
    tier_weight          NUMERIC(4, 3) NOT NULL,
    description          TEXT,
    created_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 2. Certified Independent Reference Benchmark (Circularity Guard)
CREATE TABLE IF NOT EXISTS mdm_eval.certified_reference_set (
    entity_id            BIGINT NOT NULL,
    attribute_code       VARCHAR(64) NOT NULL,
    certified_value      TEXT NOT NULL,
    certified_as_of      DATE NOT NULL,
    audited_by           VARCHAR(64) NOT NULL,
    created_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (entity_id, attribute_code, certified_as_of)
);

-- 3. Golden Record Master Values (Survivorship Output consumed by trading/OMS)
CREATE TABLE IF NOT EXISTS mdm_eval.golden_value (
    tenant_id            UUID NOT NULL,
    as_of_ts             TIMESTAMP WITH TIME ZONE NOT NULL,
    entity_id            BIGINT NOT NULL,
    attribute_code       VARCHAR(64) NOT NULL,
    golden_value         TEXT,
    winning_vendor_id    VARCHAR(64) NOT NULL,
    rule_applied         VARCHAR(64) NOT NULL,
    created_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, as_of_ts, entity_id, attribute_code)
);
CREATE INDEX IF NOT EXISTS ix_golden_eval ON mdm_eval.golden_value (attribute_code, winning_vendor_id, as_of_ts);

-- 4. Value Override Register (Steward Human-in-the-Loop votes)
CREATE TABLE IF NOT EXISTS mdm_eval.value_override (
    override_id          BIGSERIAL PRIMARY KEY,
    tenant_id            UUID NOT NULL,
    entity_id            BIGINT NOT NULL,
    attribute_code       VARCHAR(64) NOT NULL,
    prior_golden_value   TEXT,
    prior_vendor_id      VARCHAR(64),
    overridden_value     TEXT NOT NULL,
    endorsement_vendor_id VARCHAR(64),
    steward_id           VARCHAR(64) NOT NULL,
    defect_reason        VARCHAR(128) NOT NULL,
    override_ts          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_override_eval ON mdm_eval.value_override (endorsement_vendor_id, attribute_code);

-- Seed initial attribute tolerances for Tier 1, Tier 2, and Tier 3 fields
INSERT INTO mdm_eval.attribute_tolerance (attribute_code, tier, match_type, tolerance_val, tier_weight, description)
VALUES
    -- Tier 1: Trade / valuation / compliance blocking (weight 0.50)
    ('LEI', 1, 'EXACT', 0, 0.50, 'Legal Entity Identifier exact check'),
    ('ISIN', 1, 'EXACT', 0, 0.50, 'International Securities Identification Number'),
    ('CLOSING_PRICE', 1, 'NUMERIC_BP', 0.0001, 0.50, 'Closing price within 1 basis point'),
    ('COMPOSITE_RATING', 1, 'EXACT', 0, 0.50, 'Issuer credit rating normalized'),
    ('COUNTRY_OF_RISK', 1, 'EXACT', 0, 0.50, 'Country of ultimate risk'),
    ('SANCTIONS_FLAG', 1, 'EXACT', 0, 0.50, 'OFAC / EU / UN sanctions status'),

    -- Tier 2: Reporting & client-facing (weight 0.30)
    ('LEGAL_NAME', 2, 'FUZZY_JARO', 0.92, 0.30, 'Issuer legal entity name fuzzy match'),
    ('DOMICILE', 2, 'EXACT', 0, 0.30, 'Legal jurisdiction of incorporation'),
    ('GICS_SECTOR', 2, 'EXACT', 0, 0.30, 'Global Industry Classification Standard'),
    ('MARKET_CAP', 2, 'NUMERIC_PCT', 0.01, 0.30, 'Market capitalization within 0.01%'),
    ('SHARES_OUTSTANDING', 2, 'NUMERIC_PCT', 0.01, 0.30, 'Total issued shares within 0.01%'),
    ('PARENT_SUBSIDIARY', 2, 'EXACT', 0, 0.30, 'Immediate parent entity LEI/ID'),

    -- Tier 3: Enrichment & Metadata (weight 0.20)
    ('NAICS_INDUSTRY', 3, 'EXACT', 0, 0.20, 'North American Industry Classification'),
    ('EMPLOYEE_COUNT', 3, 'NUMERIC_PCT', 2.00, 0.20, 'Reported head count within 2%'),
    ('WEBSITE', 3, 'EXACT', 0, 0.20, 'Canonical primary URL'),
    ('YEAR_FOUNDED', 3, 'EXACT', 0, 0.20, 'Year of incorporation')
ON CONFLICT (attribute_code) DO NOTHING;
