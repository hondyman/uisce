DROP POLICY IF EXISTS staging_account_warnings_tenant ON staging.account_warnings;
DROP TABLE IF EXISTS staging.account_warnings;
DROP POLICY IF EXISTS staging_account_data_tenant ON staging.account_data;
DROP TABLE IF EXISTS staging.account_data;
DROP POLICY IF EXISTS account_master_tenant ON mdm.account_master;
DROP TABLE IF EXISTS mdm.account_master;
