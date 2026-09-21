-- crims publication covering orm.security_identifier — the single source
-- table the consumer needs from the crims database for the
-- sync_identifier_cache handler.
--
-- Mirrors the pattern of orm_cdc_publication in alpha (a one-table
-- intent-specific publication). The orm_oms_slot slot already exists on
-- crims but is inactive and was never connected to a publication; we
-- create a new narrow publication + slot for this work.
--
-- Run against the crims database:
--   psql ... -d crims -f this_file.up.sql

BEGIN;

DROP PUBLICATION IF EXISTS crims_trigger_source_publication;

CREATE PUBLICATION crims_trigger_source_publication FOR TABLE
    orm.security_identifier;

COMMIT;
