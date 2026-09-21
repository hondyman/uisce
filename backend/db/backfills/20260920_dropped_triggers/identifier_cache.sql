-- Backfill: crims orm.security — populate isin/cusip/ticker from
-- orm.security_identifier primary rows.
--
-- The deleted `sync_identifier_cache` trigger copied id_value into
-- the matching column on orm.security when a primary identifier was set.
-- This script applies the same rule to the current state.
--
-- Run against the crims database after 002h creates the publication, but
-- before the consumer takes over (otherwise CDC events for any changes
-- would race this update).
--   psql "$CRIMS_DB_URL" -f this_file.sql

BEGIN;

UPDATE orm.security s
SET    isin  = si.id_value
FROM   orm.security_identifier si
WHERE  si.security_id  = s.id
  AND  si.is_primary   = true
  AND  si.effective_to IS NULL
  AND  si.id_type      = 'ISIN'
  AND  s.isin IS DISTINCT FROM si.id_value;

UPDATE orm.security s
SET    cusip = si.id_value
FROM   orm.security_identifier si
WHERE  si.security_id  = s.id
  AND  si.is_primary   = true
  AND  si.effective_to IS NULL
  AND  si.id_type      = 'CUSIP'
  AND  s.cusip IS DISTINCT FROM si.id_value;

UPDATE orm.security s
SET    ticker = si.id_value
FROM   orm.security_identifier si
WHERE  si.security_id  = s.id
  AND  si.is_primary   = true
  AND  si.effective_to IS NULL
  AND  si.id_type      = 'TICKER'
  AND  s.ticker IS DISTINCT FROM si.id_value;

COMMIT;
