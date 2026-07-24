-- Semantic View Versioning Schema
-- Enables version-controlled semantic views with migration paths

-- Table to store different versions of semantic views
CREATE TABLE IF NOT EXISTS semantic_view_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    view_id UUID NOT NULL,
    version INTEGER NOT NULL,
    
    -- Version metadata
    schema JSONB NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_deprecated BOOLEAN NOT NULL DEFAULT false,
    
    -- Migration information
    migration_script TEXT, -- SQL or transformation logic to migrate from previous version
    breaking_changes BOOLEAN NOT NULL DEFAULT false,
    compatibility_notes TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deprecated_at TIMESTAMPTZ,
    created_by TEXT,
    
    -- Ensure uniqueness
    CONSTRAINT ux_view_version UNIQUE (view_id, version)
);

CREATE INDEX IF NOT EXISTS idx_view_versions_view_id ON semantic_view_versions(view_id);
CREATE INDEX IF NOT EXISTS idx_view_versions_active ON semantic_view_versions(view_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_view_versions_deprecated ON semantic_view_versions(is_deprecated);

-- Table to track schema migrations between versions
CREATE TABLE IF NOT EXISTS view_schema_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    view_id UUID NOT NULL,
    from_version INTEGER NOT NULL,
    to_version INTEGER NOT NULL,
    
    -- Migration metadata
    migration_type TEXT NOT NULL CHECK (migration_type IN ('additive', 'breaking', 'compatible', 'deprecation')),
    migration_status TEXT NOT NULL DEFAULT 'pending' CHECK (migration_status IN ('pending', 'in_progress', 'completed', 'failed', 'rolled_back')),
    
    -- Execution details
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    rows_affected INTEGER,
    
    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    executed_by TEXT,
    
    CONSTRAINT ux_migration UNIQUE (view_id, from_version, to_version)
);

CREATE INDEX IF NOT EXISTS idx_migrations_view_id ON view_schema_migrations(view_id);
CREATE INDEX IF NOT EXISTS idx_migrations_status ON view_schema_migrations(migration_status);
CREATE INDEX IF NOT EXISTS idx_migrations_created ON view_schema_migrations(created_at DESC);

-- Table to track clients using specific versions
CREATE TABLE IF NOT EXISTS view_version_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    view_id UUID NOT NULL,
    version INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Usage metadata
    first_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    usage_count INTEGER NOT NULL DEFAULT 1,
    
    CONSTRAINT ux_version_usage UNIQUE (view_id, version, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_version_usage_view ON view_version_usage(view_id, version);
CREATE INDEX IF NOT EXISTS idx_version_usage_tenant ON view_version_usage(tenant_id);

-- Function to get the latest active version of a view
CREATE OR REPLACE FUNCTION get_latest_view_version(p_view_id UUID)
RETURNS INTEGER AS $$
DECLARE
    latest_version INTEGER;
BEGIN
    SELECT MAX(version) INTO latest_version
    FROM semantic_view_versions
    WHERE view_id = p_view_id
      AND is_active = true;
    
    RETURN COALESCE(latest_version, 1);
END;
$$ LANGUAGE plpgsql;

-- Function to check if a migration is safe
CREATE OR REPLACE FUNCTION is_migration_safe(p_view_id UUID, p_from_version INTEGER, p_to_version INTEGER)
RETURNS BOOLEAN AS $$
DECLARE
    has_breaking_changes BOOLEAN;
    active_users INTEGER;
BEGIN
    -- Check if target version has breaking changes
    SELECT breaking_changes INTO has_breaking_changes
    FROM semantic_view_versions
    WHERE view_id = p_view_id
      AND version = p_to_version;
    
    IF has_breaking_changes THEN
        -- Check if there are active users on the old version
        SELECT COUNT(*) INTO active_users
        FROM view_version_usage
        WHERE view_id = p_view_id
          AND version = p_from_version
          AND last_used_at > NOW() - INTERVAL '7 days';
        
        -- Migration is unsafe if there are active users and breaking changes
        RETURN active_users = 0;
    END IF;
    
    -- Non-breaking changes are always safe
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update last_used_at in view_version_usage
CREATE OR REPLACE FUNCTION update_view_version_usage()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO view_version_usage (view_id, version, tenant_id, usage_count)
    VALUES (NEW.view_id, NEW.version, NEW.tenant_id, 1)
    ON CONFLICT (view_id, version, tenant_id)
    DO UPDATE SET
        last_used_at = NOW(),
        usage_count = view_version_usage.usage_count + 1;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE semantic_view_versions IS 'Stores version history of semantic views for safe schema evolution';
COMMENT ON TABLE view_schema_migrations IS 'Tracks migrations between semantic view versions';
COMMENT ON TABLE view_version_usage IS 'Tracks which tenants are using which versions for migration planning';
COMMENT ON FUNCTION get_latest_view_version IS 'Returns the highest active version number for a semantic view';
COMMENT ON FUNCTION is_migration_safe IS 'Checks if migrating from one version to another is safe based on breaking changes and active usage';
