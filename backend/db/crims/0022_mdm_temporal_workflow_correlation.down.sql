-- 0022_mdm_temporal_workflow_correlation.down.sql

\set ON_ERROR_STOP on
BEGIN;

DROP INDEX IF EXISTS mdm.idx_config_change_temporal_wf;
ALTER TABLE mdm.mastering_config_change DROP COLUMN IF EXISTS temporal_workflow_id;

DROP INDEX IF EXISTS mdm.idx_golden_merge_temporal_wf;
ALTER TABLE mdm.golden_merge_request DROP COLUMN IF EXISTS temporal_workflow_id;

DROP INDEX IF EXISTS mdm.idx_golden_override_temporal_wf;
ALTER TABLE mdm.golden_override DROP COLUMN IF EXISTS temporal_workflow_id;

COMMIT;
