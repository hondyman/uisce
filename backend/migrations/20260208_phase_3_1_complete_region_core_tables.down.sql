-- Rollback Phase 3.1 Complete

DROP VIEW IF EXISTS ops_incidents_by_region;

ALTER TABLE IF EXISTS ops_incidents DROP COLUMN IF EXISTS region;
DROP INDEX IF EXISTS idx_ops_incidents_region;
DROP INDEX IF EXISTS idx_ops_incidents_region_started_at;

DROP INDEX IF EXISTS idx_ops_events_region;
DROP INDEX IF EXISTS idx_ops_events_region_occurred_at;

ALTER TABLE IF EXISTS ops_action_history DROP COLUMN IF EXISTS region;
DROP INDEX IF EXISTS idx_ops_action_history_region;
DROP INDEX IF EXISTS idx_ops_action_history_region_created_at;
