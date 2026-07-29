-- CBO Telemetry Index Migration
-- Covers the most common CBO query pattern on semantic_query_history_v2:
--   WHERE tenant_id = $1 AND cube_name = $2 AND created_at >= NOW() - interval
-- Existing partial indexes on tenant_id and cube_name separately are insufficient
-- for the CBO's correlated filter pattern.

CREATE INDEX IF NOT EXISTS idx_cbo_telemetry_tenant_cube_time
    ON semantic_query_history_v2 (tenant_id, cube_name, created_at DESC)
    WHERE cube_name IS NOT NULL;

COMMENT ON INDEX idx_cbo_telemetry_tenant_cube_time
    IS 'CBO telemetry router: covers (tenant_id, cube_name, created_at) filter for GetOptimalFlavor';
