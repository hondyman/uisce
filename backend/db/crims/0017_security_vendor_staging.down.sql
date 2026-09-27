-- 0017_security_vendor_staging.down.sql
\set ON_ERROR_STOP on
BEGIN;
DROP TABLE IF EXISTS staging.ice_security;
DROP TABLE IF EXISTS staging.rdp_security;
DROP TABLE IF EXISTS staging.bbg_security;
COMMIT;
