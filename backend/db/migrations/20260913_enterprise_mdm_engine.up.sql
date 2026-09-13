-- 20260913_enterprise_mdm_engine.up.sql
-- Enterprise MDM Engine, Symbology Cross-Reference & Golden Copy Survivorship Ledger

CREATE SCHEMA IF NOT EXISTS catalog_mdm;

-- 1. Master Data Domains
CREATE TABLE IF NOT EXISTS catalog_mdm.domains (
    domain_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_key VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Master Entity Golden Records
CREATE TABLE IF NOT EXISTS catalog_mdm.golden_entities (
    golden_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_id UUID NOT NULL,
    canonical_node_id UUID NOT NULL,
    primary_business_key VARCHAR(150) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    golden_attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    merkle_version_hash VARCHAR(64) NOT NULL,
    last_mastered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_domain_bk UNIQUE (tenant_id, domain_id, primary_business_key)
);

-- 3. Cross-Vendor Identifier Resolution Index
CREATE TABLE IF NOT EXISTS catalog_mdm.identifier_cross_reference (
    id_ref_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    golden_id UUID NOT NULL REFERENCES catalog_mdm.golden_entities(golden_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    identifier_type VARCHAR(30) NOT NULL,
    identifier_value VARCHAR(100) NOT NULL,
    vendor_source VARCHAR(50) NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
    valid_to DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_id_value UNIQUE (tenant_id, identifier_type, identifier_value, vendor_source)
);

-- 4. Dynamic Attribute-Level Survivorship Rules
CREATE TABLE IF NOT EXISTS catalog_mdm.survivorship_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_id UUID NOT NULL,
    attribute_name VARCHAR(100) NOT NULL,
    survivorship_strategy VARCHAR(50) NOT NULL,
    priority_vendor_order JSONB NOT NULL DEFAULT '["BLOOMBERG", "REFINITIV", "FACTSET", "INTERNAL"]'::jsonb,
    tolerance_pct NUMERIC(6, 4) DEFAULT 0.0200,
    stale_tolerance_minutes INT DEFAULT 1440,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_domain_attr UNIQUE (tenant_id, domain_id, attribute_name)
);

-- 5. Multi-Vendor Staged Raw Payloads
CREATE TABLE IF NOT EXISTS catalog_mdm.staged_vendor_records (
    staged_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    domain_id UUID NOT NULL,
    vendor_source VARCHAR(50) NOT NULL,
    vendor_record_id VARCHAR(100) NOT NULL,
    raw_payload JSONB NOT NULL,
    extracted_identifiers JSONB NOT NULL,
    effective_date TIMESTAMPTZ NOT NULL,
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. MDM Exception & Break Triage Ledger
CREATE TABLE IF NOT EXISTS catalog_mdm.exceptions (
    exception_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_id UUID REFERENCES catalog_mdm.golden_entities(golden_id) ON DELETE SET NULL,
    staged_id UUID NOT NULL REFERENCES catalog_mdm.staged_vendor_records(staged_id) ON DELETE CASCADE,
    attribute_name VARCHAR(100) NOT NULL,
    exception_type VARCHAR(50) NOT NULL,
    primary_vendor_value TEXT,
    secondary_vendor_value TEXT,
    variance_delta_pct NUMERIC(8, 4),
    status VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    override_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mdm_id_lookup 
ON catalog_mdm.identifier_cross_reference (tenant_id, identifier_type, identifier_value);

CREATE INDEX IF NOT EXISTS idx_mdm_exceptions_status 
ON catalog_mdm.exceptions (tenant_id, status, exception_type);
