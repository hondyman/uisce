-- Add pre-aggregation tracking columns to query_telemetry table
-- These enable tracking of which queries hit pre-aggregations for ROI analysis

ALTER TABLE query_telemetry ADD COLUMN IF NOT EXISTS preagg_id VARCHAR(255);
ALTER TABLE query_telemetry ADD COLUMN IF NOT EXISTS preagg_hit BOOLEAN DEFAULT FALSE;

-- CREATE index IF NOT EXISTS for efficient querying of pre-agg usage
CREATE INDEX IF NOT EXISTS idx_query_telemetry_preagg ON query_telemetry(preagg_id) WHERE preagg_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN query_telemetry.preagg_id IS 'UUID of pre-aggregation used for this query, if any';
COMMENT ON COLUMN query_telemetry.preagg_hit IS 'Whether this query was routed to a pre-aggregation';
