-- Allow bi-temporal versions on mdm.security_master (Account pattern).
-- Replace hard UNIQUE(security_id, tenant_id) with current-row uniqueness.

ALTER TABLE mdm.security_master
    DROP CONSTRAINT IF EXISTS uq_security_master_id_tenant;

CREATE UNIQUE INDEX IF NOT EXISTS uq_security_master_current
    ON mdm.security_master (tenant_id, security_id)
    WHERE valid_to IS NULL;
