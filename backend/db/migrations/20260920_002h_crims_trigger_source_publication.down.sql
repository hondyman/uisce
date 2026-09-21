-- Rollback for 20260920_002h_crims_trigger_source_publication.up.sql

BEGIN;

DROP PUBLICATION IF EXISTS crims_trigger_source_publication;

COMMIT;
