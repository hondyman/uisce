-- 0013_mastering_overrides.down.sql
-- Reverses 0013. Drops overrides and their history: only for a crims rebuild.

\set ON_ERROR_STOP on
BEGIN;
DROP TABLE IF EXISTS mdm.golden_override_vote;
DROP TABLE IF EXISTS mdm.golden_override;
DROP TABLE IF EXISTS mdm.mastering_policy_audit;
DROP TABLE IF EXISTS mdm.mastering_policy;
COMMIT;
