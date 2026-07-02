-- +goose Down
-- Revert rename idp_group_id to idp_group_claim and recreate constraint/index.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND column_name = 'idp_group_id'
    ) THEN
        ALTER TABLE security.identity_profile_mappings 
        RENAME COLUMN idp_group_id TO idp_group_claim;
    END IF;
END $$;

-- Recreate index with old column name
DROP INDEX IF EXISTS security.idx_idp_mappings;
CREATE INDEX IF NOT EXISTS idx_idp_mappings ON security.identity_profile_mappings(idp_group_claim, tenant_id);

-- Drop new constraint if exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND constraint_name = 'uq_client_group_id'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        DROP CONSTRAINT uq_client_group_id;
    END IF;
END $$;

-- Create old constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND constraint_name = 'uq_client_group'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        ADD CONSTRAINT uq_client_group UNIQUE (idp_client_id, idp_group_claim);
    END IF;
END $$;
