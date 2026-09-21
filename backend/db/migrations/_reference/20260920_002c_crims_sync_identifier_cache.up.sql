-- Drop the single non-updated_at trigger from crims: orm.sync_identifier_cache
-- Per directive: only want updated_at triggers; all others go to CDC evaluation.
--
-- This trigger syncs ISIN/CUSIP/TICKER values from orm.security_identifier
-- to orm.security.isin/cusip/ticker when a primary identifier is set.
-- Low volume, but should be evaluated for CDC replacement.
--
-- Run: psql ... -f this_file

BEGIN;

DROP TRIGGER IF EXISTS trg_sec_ident_cache ON orm.security_identifier;
DROP FUNCTION  IF EXISTS orm.sync_identifier_cache();

COMMIT;
