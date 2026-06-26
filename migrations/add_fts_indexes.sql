-- Add full-text search indexes
CREATE INDEX IF NOT EXISTS idx_catalog_node_fts ON catalog_node USING GIN (to_tsvector('english', node_name || ' ' || COALESCE(description, '') || ' ' || COALESCE(properties::text, '')));
CREATE INDEX IF NOT EXISTS idx_business_terms_fts ON business_terms USING GIN (to_tsvector('english', term || ' ' || definition || ' ' || COALESCE(synonyms::text, '')));
