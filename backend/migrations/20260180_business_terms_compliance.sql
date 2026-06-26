CREATE TABLE IF NOT EXISTS business_terms (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    pii_flag BOOLEAN DEFAULT FALSE,
    residency TEXT DEFAULT 'GLOBAL',
    sensitivity_level TEXT DEFAULT 'LOW',
    semantic_term_ids TEXT[], -- Array of UUIDs referring to catalog_node
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    updated_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_business_terms_tenant ON business_terms(tenant_id);
-- Index for quick lookup of business terms by semantic term ID (using GIN for array containment)
CREATE INDEX IF NOT EXISTS idx_business_terms_semantic_ids ON business_terms USING GIN (semantic_term_ids);
