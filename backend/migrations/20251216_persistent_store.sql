-- Create the generic persistent store for schema-less JSONB storage
-- This enables the "Ignite -> JSONB" pattern where schema evolution is handled in the app/metadata layer.

CREATE TABLE IF NOT EXISTS persistent_store (
    id TEXT PRIMARY KEY,
    object_type TEXT NOT NULL,
    data JSONB NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- GIN Index for high-performance JSON querying
CREATE INDEX IF NOT EXISTS idx_persistent_store_gin ON persistent_store USING GIN (data);

-- Index on object_type for filtering by Business Object type
CREATE INDEX IF NOT EXISTS idx_persistent_store_type ON persistent_store (object_type);

-- Index on tenant_id for multi-tenancy isolation
CREATE INDEX IF NOT EXISTS idx_persistent_store_tenant ON persistent_store (tenant_id);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_persistent_store_modtime
    BEFORE UPDATE ON persistent_store
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();
