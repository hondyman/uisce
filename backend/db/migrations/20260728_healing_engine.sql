-- Uisce Self-Healing Mapping Audit Schema
-- Rule 6 (Semantic/OLTP Boundary) & Rule 7 (Security Mandate)

BEGIN;

CREATE TABLE IF NOT EXISTS platform.self_healing_audit (
    healing_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    table_name VARCHAR(150) NOT NULL,
    old_column VARCHAR(100) NOT NULL,
    suggested_column VARCHAR(100) NOT NULL,
    confidence_score NUMERIC(5, 4) NOT NULL, -- e.g., 0.9650
    status VARCHAR(50) DEFAULT 'AUTO_REPAIRED', -- AUTO_REPAIRED, PENDING_REVIEW, REJECTED
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_healing_tenant ON platform.self_healing_audit(tenant_id, status);

COMMIT;
