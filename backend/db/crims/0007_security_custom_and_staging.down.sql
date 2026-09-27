DROP POLICY IF EXISTS staging_security_warnings_tenant ON staging.security_warnings;
DROP TABLE IF EXISTS staging.security_warnings;
DROP POLICY IF EXISTS staging_security_data_tenant ON staging.security_data;
DROP TABLE IF EXISTS staging.security_data;
DROP INDEX IF EXISTS mdm.idx_security_master_custom_gin;
ALTER TABLE mdm.security_master DROP COLUMN IF EXISTS custom_attributes;
