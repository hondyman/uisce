-- 0021_security_instrument_type.down.sql
-- Removes the column added by 0021. The backfill is not reversible: the values
-- came from staging.bbg_security, which is unchanged, so re-running 0021's
-- backfill restores them.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0021_security_instrument_type.down.sql

\set ON_ERROR_STOP on
BEGIN;

ALTER TABLE staging.security_data
    DROP COLUMN IF EXISTS instrument_type;

COMMIT;
