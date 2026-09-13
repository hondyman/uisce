CREATE TABLE IF NOT EXISTS ai_semantic_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    prompt_embedding vector(1536),
    query_payload JSONB NOT NULL,
    hits INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '12 hours'
);

-- Optimize for fast cosine distance matching
CREATE INDEX IF NOT EXISTS idx_semantic_cache_embedding 
ON ai_semantic_cache USING hnsw (prompt_embedding vector_cosine_ops);

-- Index for quick cleanup of expired cache entries
CREATE INDEX IF NOT EXISTS idx_semantic_cache_expires 
ON ai_semantic_cache(expires_at);
