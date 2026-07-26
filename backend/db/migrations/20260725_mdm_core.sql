-- Uisce MDM Core Master Schema
-- Rule 6 (Semantic/OLTP Boundary) & Rule 7 (Security Mandate)

BEGIN;

CREATE SCHEMA IF NOT EXISTS mdm;

-- 1. Vendor Preference Survivorship Matrix Table (Rule 1 Config-Before-Code)
CREATE TABLE IF NOT EXISTS mdm.survivorship_rule (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bo_type VARCHAR(100) NOT NULL,            -- e.g., 'SECURITY_MASTER', 'PRICING_MASTER'
    field_name VARCHAR(100) NOT NULL,         -- e.g., 'coupon_rate', 'closing_price'
    vendor_priority_json JSONB NOT NULL,       -- e.g., {"BLOOMBERG": 100, "REFINITIV": 80, "MANUAL_OVERRIDE": 1000}
    tolerance_pct NUMERIC(5, 2) DEFAULT 10.00, -- e.g., 10.00 (% jump alert threshold)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, bo_type, field_name)
);

-- 2. MDM Exception Queue (Rule 6 Semantic/OLTP Boundary)
CREATE TABLE IF NOT EXISTS mdm.exception_queue (
    exception_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_id VARCHAR(255) NOT NULL,        -- Temporal Workflow ID
    entity_key VARCHAR(100) NOT NULL,         -- e.g., ISIN/CUSIP
    bo_type VARCHAR(100) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    competing_values_json JSONB NOT NULL,     -- e.g., {"BLOOMBERG": 120.0, "IDC": 12.0}
    status VARCHAR(50) DEFAULT 'OPEN',        -- OPEN, RESOLVED, OVERRIDDEN, REJECTED
    assigned_steward VARCHAR(100),
    ai_recommendation VARCHAR(500),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Indexing for sub-millisecond tenant isolation (Rule 7)
CREATE INDEX IF NOT EXISTS idx_mdm_survivorship_tenant ON mdm.survivorship_rule(tenant_id, bo_type);
CREATE INDEX IF NOT EXISTS idx_mdm_exception_tenant ON mdm.exception_queue(tenant_id, status);

COMMIT;
