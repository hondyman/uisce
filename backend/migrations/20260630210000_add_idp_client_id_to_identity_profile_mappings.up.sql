-- Add idp_client_id to security.identity_profile_mappings for federated users.
-- Existing deployments may have created the table without this column.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND column_name = 'idp_client_id'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        ADD COLUMN idp_client_id VARCHAR(100) NOT NULL DEFAULT '__legacy__';
    END IF;
END $$;

-- Drop the old tenant/group unique constraint if it exists and create the
-- client/group constraint that prevents cross-tenant IdP spoofing.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND constraint_name = 'uq_tenant_group'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        DROP CONSTRAINT uq_tenant_group;
    END IF;
END $$;

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
