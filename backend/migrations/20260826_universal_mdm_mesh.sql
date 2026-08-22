-- 20260826_universal_mdm_mesh.sql
-- Universal Multi-Domain MDM Core, Graph Cross-Referencing Mesh & Master Entity Registries

CREATE SCHEMA IF NOT EXISTS mdm;

-- 1. Universal Domain Master Registry
CREATE TABLE IF NOT EXISTS mdm.master_domain_registry (
    domain_key VARCHAR(50) PRIMARY KEY, -- SECURITY, PRICING, ISSUER, FUND, ACCOUNT, CALENDAR, PRODUCT, CORP_ACTION
    domain_name TEXT NOT NULL,
    default_bo_key VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO mdm.master_domain_registry (domain_key, domain_name, default_bo_key) VALUES
    ('SECURITY', 'Security Master', 'security_master'),
    ('PRICING', 'Pricing & Curves Master', 'pricing_master'),
    ('ISSUER', 'Legal Entity & Issuer Master', 'issuer_master'),
    ('FUND', 'Fund & Vehicle Master', 'fund_master'),
    ('ACCOUNT', 'Custodial & Allocation Account Master', 'account_master'),
    ('CALENDAR', 'Exchange & Holiday Calendar Master', 'exchange_calendar'),
    ('PRODUCT', 'Product & Benchmark Master', 'benchmark_master'),
    ('CORP_ACTION', 'Corporate Actions Event Master', 'corporate_action')
ON CONFLICT (domain_key) DO NOTHING;

-- 2. Graph-Native Cross-Referencing (XREF) Identifier Mesh
CREATE TABLE IF NOT EXISTS mdm.universal_identifier_xref (
    xref_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL REFERENCES mdm.master_domain_registry(domain_key),
    master_entity_sid VARCHAR(100) NOT NULL, -- Canonical Golden Record SID
    id_type VARCHAR(50) NOT NULL,            -- ISIN, CUSIP, SEDOL, FIGI, TICKER, LEI, ACCOUNT_NUM, FUND_CODE
    id_value VARCHAR(100) NOT NULL,
    vendor_source VARCHAR(50) NOT NULL,      -- BLOOMBERG, REFINITIV, IDC, CRIMS, MANUAL
    is_primary BOOLEAN DEFAULT FALSE,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_xref_entry UNIQUE (tenant_id, domain_key, id_type, id_value, vendor_source, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_xref_lookup 
ON mdm.universal_identifier_xref (tenant_id, domain_key, id_type, id_value);

-- 3. Universal Field-Level Survivorship Rules (Config-Before-Code / Rule 1)
CREATE TABLE IF NOT EXISTS mdm.universal_survivorship_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL REFERENCES mdm.master_domain_registry(domain_key),
    field_name VARCHAR(100) NOT NULL,
    strategy VARCHAR(50) NOT NULL,           -- SOURCE_PRIORITY, MOST_RECENT, CONFIDENCE_SCORE, CONSERVATIVE_MIN, MAKER_CHECKER
    priority_vendors JSONB DEFAULT '[]'::jsonb, -- ["BLOOMBERG", "REFINITIV", "IDC", "INTERNAL"]
    anomaly_tolerance_pct NUMERIC(6, 4) DEFAULT 10.00, -- Flag if deviation > 10.00%
    staleness_max_age_sec INT DEFAULT 86400,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_survivorship_field UNIQUE (tenant_id, domain_key, field_name)
);

-- 4. MDM Golden Record Master Store (Bi-temporal Dual-Time Coordinates / Rule 4 & 6)
CREATE TABLE IF NOT EXISTS mdm.golden_record_store (
    golden_record_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL REFERENCES mdm.master_domain_registry(domain_key),
    master_entity_sid VARCHAR(100) NOT NULL,
    golden_attributes JSONB NOT NULL,         -- Authoritative master properties
    winning_sources JSONB NOT NULL,            -- Lineage mapping per field: {"px_last": "BLOOMBERG", "rating": "REFINITIV"}
    effective_date DATE NOT NULL,              -- Te (Market Effective Time)
    knowledge_timestamp TIMESTAMPTZ NOT NULL,  -- Tk (System Knowledge Time)
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_golden_snapshot UNIQUE (tenant_id, domain_key, master_entity_sid, effective_date, knowledge_timestamp)
);

-- 5. MDM Exception & Anomaly Queue (Human-in-the-Loop Temporal Signals)
CREATE TABLE IF NOT EXISTS mdm.universal_exception_queue (
    exception_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    domain_key VARCHAR(50) NOT NULL REFERENCES mdm.master_domain_registry(domain_key),
    master_entity_sid VARCHAR(100) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    competing_values JSONB NOT NULL,          -- [{"vendor": "BLOOMBERG", "value": 120.00}, {"vendor": "IDC", "value": 12.00}]
    anomaly_type VARCHAR(50) NOT NULL,        -- PRICE_TOLERANCE_BREACH, CHECKSUM_FAILURE, UNRESOLVED_XREF, STALE_FEED
    status VARCHAR(30) DEFAULT 'OPEN',        -- OPEN, IN_REVIEW, RESOLVED, OVERRIDDEN, REJECTED
    assigned_steward_id VARCHAR(100),
    steward_override_value JSONB,
    steward_override_reason TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
