-- +goose Up
-- Rename idp_group_claim to idp_group_id and recreate constraint/index.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND column_name = 'idp_group_claim'
    ) THEN
        ALTER TABLE security.identity_profile_mappings 
        RENAME COLUMN idp_group_claim TO idp_group_id;
    END IF;
END $$;

-- Recreate index with new column name
DROP INDEX IF EXISTS security.idx_idp_mappings;
CREATE INDEX IF NOT EXISTS idx_idp_mappings ON security.identity_profile_mappings(idp_group_id, tenant_id);

-- Drop old constraint if exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND constraint_name = 'uq_client_group'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        DROP CONSTRAINT uq_client_group;
    END IF;
END $$;

-- Create new constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND constraint_name = 'uq_client_group_id'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        ADD CONSTRAINT uq_client_group_id UNIQUE (idp_client_id, idp_group_id);
    END IF;
END $$;
