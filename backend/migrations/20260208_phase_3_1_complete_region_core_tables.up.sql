-- Phase 3.1 Complete: Add region to core ops tables
-- Finalize region as a first-class dimension across incidents and events
-- This migration ensures ALL critical incident tracking tables have region awareness

-- 1. Add region to ops_incidents if not present
ALTER TABLE IF EXISTS ops_incidents ADD COLUMN IF NOT EXISTS region VARCHAR(50);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_region ON ops_incidents(region);
CREATE INDEX IF NOT EXISTS idx_ops_incidents_region_started_at ON ops_incidents(region, started_at DESC);

-- 2. Ensure region is indexed on ops_events for efficient filtering
CREATE INDEX IF NOT EXISTS idx_ops_events_region ON ops_events(region);
CREATE INDEX IF NOT EXISTS idx_ops_events_region_occurred_at ON ops_events(region, occurred_at DESC);

-- 3. Add region to action_history for region-scoped audit trails
ALTER TABLE IF EXISTS ops_action_history ADD COLUMN IF NOT EXISTS region VARCHAR(50);
CREATE INDEX IF NOT EXISTS idx_ops_action_history_region ON ops_action_history(region);
CREATE INDEX IF NOT EXISTS idx_ops_action_history_region_created_at ON ops_action_history(region, created_at DESC);

-- 4. Add comments for clarity
COMMENT ON COLUMN ops_incidents.region IS 'Geographic region (e.g., us-east-1, eu-west-1, ap-southeast-1) - incident scope';
COMMENT ON COLUMN ops_events.region IS 'Geographic region for this event';
COMMENT ON COLUMN ops_action_history.region IS 'Region where action was executed';

-- 5. Add constraints if needed (NOT NULL with defaults for new columns is handled at app layer)
-- Backfill existing records with default region (us-east-1)
UPDATE ops_incidents SET region = 'us-east-1' WHERE region IS NULL;
UPDATE ops_events SET region = 'us-east-1' WHERE region IS NULL;
UPDATE ops_action_history SET region = 'us-east-1' WHERE region IS NULL;

-- Create a view for recent incidents by region for operational dashboards
CREATE OR REPLACE VIEW ops_incidents_by_region AS
SELECT 
    region,
    COUNT(*) as total_incidents,
    COUNT(CASE WHEN status = 'open' THEN 1 END) as open_incidents,
    COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed_incidents,
    MAX(severity) as max_severity,
    MAX(started_at) as latest_incident
FROM ops_incidents
WHERE started_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
GROUP BY region;

COMMENT ON VIEW ops_incidents_by_region IS 'Rolling 24-hour view of incident distribution across regions';
