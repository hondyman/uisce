-- 0014_mastering_merge_requests.down.sql
\set ON_ERROR_STOP on
BEGIN;
DROP TABLE IF EXISTS mdm.golden_merge_vote;
DROP TABLE IF EXISTS mdm.golden_merge_request;
COMMIT;
