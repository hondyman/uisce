-- backend/db/migrations/000063_fix_identity_profile_mappings.up.sql
-- Additive migration: ensures security.identity_profile_mappings table exists and has proper columns & tenant constraints

CREATE SCHEMA IF NOT EXISTS security;

CREATE TABLE IF NOT EXISTS security.identity_profile_mappings (
    mapping_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL, -- Core isolation fence
    idp_group_claim   VARCHAR(255) NOT NULL, -- e.g., 'GG-Uisce-Compliance'
    functional_role   VARCHAR(100) NOT NULL, -- e.g., 'compliance_officer'
    clearance_level   VARCHAR(50) NOT NULL,  -- e.g., 'L3'
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Ensure all expected columns exist if the table was created by a previous schema variant
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'security' AND table_name = 'identity_profile_mappings' AND column_name = 'mapping_id'
    ) THEN
        ALTER TABLE security.identity_profile_mappings ADD COLUMN mapping_id UUID DEFAULT gen_random_uuid();
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'security' AND table_name = 'identity_profile_mappings' AND column_name = 'idp_group_claim'
    ) THEN
        -- If an older column name like identity_sub exists, map or add idp_group_claim
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_schema = 'security' AND table_name = 'identity_profile_mappings' AND column_name = 'identity_sub'
        ) THEN
            ALTER TABLE security.identity_profile_mappings RENAME COLUMN identity_sub TO idp_group_claim;
        ELSE
            ALTER TABLE security.identity_profile_mappings ADD COLUMN idp_group_claim VARCHAR(255);
        END IF;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'security' AND table_name = 'identity_profile_mappings' AND column_name = 'functional_role'
    ) THEN
        ALTER TABLE security.identity_profile_mappings ADD COLUMN functional_role VARCHAR(100);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'security' AND table_name = 'identity_profile_mappings' AND column_name = 'clearance_level'
    ) THEN
        ALTER TABLE security.identity_profile_mappings ADD COLUMN clearance_level VARCHAR(50);
    END IF;
END $$;

-- Create index if not exists
CREATE INDEX IF NOT EXISTS idx_idp_mappings ON security.identity_profile_mappings(idp_group_claim, tenant_id);
