-- Create catalog_aliases table to map friendly names to canonical keys
CREATE TABLE IF NOT EXISTS catalog_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alias VARCHAR(255) NOT NULL,
    canonical_key VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, alias)
);

-- Index for fast lookup by alias
CREATE INDEX IF NOT EXISTS idx_catalog_aliases_lookup ON catalog_aliases(tenant_id, alias);
