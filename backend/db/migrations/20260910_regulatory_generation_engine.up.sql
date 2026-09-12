-- 20260910_regulatory_generation_engine.up.sql
-- Automated Regulatory Generation & Attested Filing Engine (SEC 13F / N-PORT / Form PF)

CREATE SCHEMA IF NOT EXISTS catalog_regulatory;

-- 1. Supported Regulatory Regimes & Validation Specifications
CREATE TABLE IF NOT EXISTS catalog_regulatory.filing_definitions (
    definition_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    regime_code VARCHAR(50) NOT NULL UNIQUE, -- SEC_13F_HR, SEC_NPORT, SEC_FORM_PF, SOLVENCY_II_QRT
    title VARCHAR(150) NOT NULL,
    governing_body VARCHAR(50) NOT NULL, -- SEC, CFTC, EIOPA, FINRA
    xsd_schema_version VARCHAR(20) NOT NULL,
    xml_namespace_uri VARCHAR(255) NOT NULL,
    filing_frequency VARCHAR(20) NOT NULL, -- MONTHLY, QUARTERLY, ANNUAL
    filing_deadline_days INT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Period Filing Execution Runs
CREATE TABLE IF NOT EXISTS catalog_regulatory.filing_period_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    definition_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    filing_period_end_date DATE NOT NULL,
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_qualifying_holdings INT NOT NULL DEFAULT 0,
    gross_reportable_value_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    validation_status VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    generated_xml_payload TEXT,
    xml_sha256_checksum VARCHAR(64),
    merkle_attestation_root VARCHAR(64),
    attested_by VARCHAR(100),
    attested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_filing_period UNIQUE (tenant_id, definition_id, portfolio_node_id, filing_period_end_date)
);

-- 3. Normalized Filing Holdings & Look-Through Constituent Ledger
CREATE TABLE IF NOT EXISTS catalog_regulatory.filing_holding_constituents (
    constituent_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES catalog_regulatory.filing_period_runs(run_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    cusip VARCHAR(9) NOT NULL,
    isin VARCHAR(12),
    issuer_name VARCHAR(200) NOT NULL,
    title_of_class VARCHAR(100) NOT NULL,
    shares_or_principal_amount NUMERIC(28, 6) NOT NULL,
    market_value_usd NUMERIC(28, 4) NOT NULL,
    investment_discretion VARCHAR(20) NOT NULL DEFAULT 'SOLE',
    voting_authority_sole NUMERIC(28, 6) NOT NULL DEFAULT 0,
    voting_authority_shared NUMERIC(28, 6) NOT NULL DEFAULT 0,
    voting_authority_none NUMERIC(28, 6) NOT NULL DEFAULT 0,
    is_de_minimis_excluded BOOLEAN DEFAULT FALSE,
    look_through_feeder_node_id UUID,
    merkle_leaf_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Audit & Attestation Non-Repudiation Passport Ledger (SEC Rule 17a-4)
CREATE TABLE IF NOT EXISTS catalog_regulatory.filing_attestation_passports (
    passport_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES catalog_regulatory.filing_period_runs(run_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    officer_user_id VARCHAR(100) NOT NULL,
    officer_role VARCHAR(50) NOT NULL,
    signature_algorithm VARCHAR(30) NOT NULL DEFAULT 'HMAC_SHA256_RSA4096',
    digital_signature_hash TEXT NOT NULL,
    verification_assertions JSONB NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    signed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_filing_period_lookup 
ON catalog_regulatory.filing_period_runs (tenant_id, filing_period_end_date, validation_status);
