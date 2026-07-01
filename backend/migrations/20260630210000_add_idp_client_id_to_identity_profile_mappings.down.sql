-- Revert idp_client_id addition on security.identity_profile_mappings.

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

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND constraint_name = 'uq_tenant_group'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        ADD CONSTRAINT uq_tenant_group UNIQUE (tenant_id, idp_group_claim);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'security'
          AND table_name = 'identity_profile_mappings'
          AND column_name = 'idp_client_id'
    ) THEN
        ALTER TABLE security.identity_profile_mappings
        DROP COLUMN idp_client_id;
    END IF;
END $$;
