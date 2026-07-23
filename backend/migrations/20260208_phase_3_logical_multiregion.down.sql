-- Rollback for Phase 3.1: Logical Multi-Region Architecture

-- Drop region routing metadata and indexes
DROP TABLE IF EXISTS region_routing CASCADE;

-- Drop region config metadata and indexes
DROP TABLE IF EXISTS region_config CASCADE;

-- Remove region columns from core tables
ALTER TABLE IF EXISTS ops_action_history DROP COLUMN IF EXISTS region;
DROP INDEX IF EXISTS idx_ops_action_history_region;
DROP INDEX IF EXISTS idx_ops_action_history_region_created_at;

-- Remove region indexes (columns were already present in audit_log and events)
DROP INDEX IF EXISTS idx_ops_audit_log_region;
DROP INDEX IF EXISTS idx_ops_events_region;
DROP INDEX IF EXISTS idx_ops_events_region_occurred_at;
DROP INDEX IF EXISTS idx_ops_incidents_region;
DROP INDEX IF EXISTS idx_ops_incidents_region_created_at;

-- Note: region column stays in ops_incidents and ops_events since they're core for Phase 3
-- Only metadata support is rolled back
