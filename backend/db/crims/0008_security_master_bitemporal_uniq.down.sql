DROP INDEX IF EXISTS mdm.uq_security_master_current;
-- Restoring the old constraint will fail if multiple versions exist.
ALTER TABLE mdm.security_master
    ADD CONSTRAINT uq_security_master_id_tenant UNIQUE (security_id, tenant_id);
