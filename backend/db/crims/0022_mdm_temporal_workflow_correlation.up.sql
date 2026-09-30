-- 0022_mdm_temporal_workflow_correlation.up.sql
-- Add temporal_workflow_id correlation columns and indexes to MDM governance tables

\set ON_ERROR_STOP on
BEGIN;

-- 1. mdm.golden_override
ALTER TABLE mdm.golden_override ADD COLUMN IF NOT EXISTS temporal_workflow_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_golden_override_temporal_wf ON mdm.golden_override(temporal_workflow_id);

-- 2. mdm.golden_merge_request
ALTER TABLE mdm.golden_merge_request ADD COLUMN IF NOT EXISTS temporal_workflow_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_golden_merge_temporal_wf ON mdm.golden_merge_request(temporal_workflow_id);

-- 3. mdm.mastering_config_change
ALTER TABLE mdm.mastering_config_change ADD COLUMN IF NOT EXISTS temporal_workflow_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_config_change_temporal_wf ON mdm.mastering_config_change(temporal_workflow_id);

COMMIT;
