-- Template for creating a tenant-specific schema
-- Replace {{.SchemaName}} with the actual schema name (e.g., tenant_t1)

CREATE SCHEMA IF NOT EXISTS {{.SchemaName}};

CREATE TABLE IF NOT EXISTS {{.SchemaName}}.chunks (
  chunk_id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  chunk_index INT NOT NULL,
  text TEXT NOT NULL,
  token_count INT NOT NULL,
  metadata JSONB DEFAULT '{}'::jsonb,
  source_snapshot_id TEXT,
  embedding vector(1536),
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Index for vector similarity search (IVFFlat)
-- Note: lists parameter should be tuned based on dataset size (rows / 1000)
CREATE INDEX IF NOT EXISTS chunks_embedding_idx 
ON {{.SchemaName}}.chunks 
USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);

-- Metadata indexes for hybrid filtering
CREATE INDEX IF NOT EXISTS chunks_document_idx ON {{.SchemaName}}.chunks (document_id);
CREATE INDEX IF NOT EXISTS chunks_meta_section_idx ON {{.SchemaName}}.chunks ((metadata->>'section'));
