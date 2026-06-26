-- Create business_terms table
CREATE TABLE IF NOT EXISTS business_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term VARCHAR(255) NOT NULL,
    definition TEXT NOT NULL,
    synonyms JSONB DEFAULT '[]',
    scope VARCHAR(255),
    canonical_key VARCHAR(255),
    tenant_id UUID NOT NULL,
    embedding vector(1536),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, term)
);

-- Create data_profiles table
CREATE TABLE IF NOT EXISTS data_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL, -- References catalog_node(id)
    row_count BIGINT,
    freshness TIMESTAMP WITH TIME ZONE,
    null_rates JSONB DEFAULT '{}',
    distincts JSONB DEFAULT '{}',
    distributions JSONB DEFAULT '{}',
    tenant_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_business_terms_lookup ON business_terms(tenant_id, term);
CREATE INDEX IF NOT EXISTS idx_data_profiles_entity ON data_profiles(entity_id);
