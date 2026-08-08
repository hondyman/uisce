-- +migrate Up
-- Convert timestamp columns in base tables to timestamptz for UTC storage
-- Only alters actual base tables, not views
-- NOTE: Views were dropped and recreated due to dependencies

ALTER TABLE alerts ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE alerts ALTER COLUMN escalated_at TYPE TIMESTAMPTZ USING escalated_at::timestamptz;
ALTER TABLE alerts ALTER COLUMN resolved_at TYPE TIMESTAMPTZ USING resolved_at::timestamptz;
ALTER TABLE alerts ALTER COLUMN triggered_at TYPE TIMESTAMPTZ USING triggered_at::timestamptz;
ALTER TABLE app_user ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE async_jobs ALTER COLUMN completed_at TYPE TIMESTAMPTZ USING completed_at::timestamptz;
ALTER TABLE async_jobs ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE async_jobs ALTER COLUMN started_at TYPE TIMESTAMPTZ USING started_at::timestamptz;
ALTER TABLE bp_branch_executions ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE compliance_results ALTER COLUMN evaluated_at TYPE TIMESTAMPTZ USING evaluated_at::timestamptz;
ALTER TABLE fund_commitments ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE fund_position_snapshots ALTER COLUMN as_of_timestamp TYPE TIMESTAMPTZ USING as_of_timestamp::timestamptz;
ALTER TABLE fund_position_snapshots ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE job_exports ALTER COLUMN completed_at TYPE TIMESTAMPTZ USING completed_at::timestamptz;
ALTER TABLE job_exports ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE job_exports ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at::timestamptz;
ALTER TABLE pushback_analysis ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE rebalance_audit ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at::timestamptz;
ALTER TABLE reconciliation_records ALTER COLUMN reconciled_at TYPE TIMESTAMPTZ USING reconciled_at::timestamptz;
ALTER TABLE scenario_pnl ALTER COLUMN evaluated_at TYPE TIMESTAMPTZ USING evaluated_at::timestamptz;
ALTER TABLE user_profiles ALTER COLUMN last_activity_at TYPE TIMESTAMPTZ USING last_activity_at::timestamptz;
ALTER TABLE wasm_versions ALTER COLUMN build_time TYPE TIMESTAMPTZ USING build_time::timestamptz;

-- +migrate Down
-- Revert timestamptz columns back to timestamp

ALTER TABLE alerts ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE alerts ALTER COLUMN escalated_at TYPE TIMESTAMP;
ALTER TABLE alerts ALTER COLUMN resolved_at TYPE TIMESTAMP;
ALTER TABLE alerts ALTER COLUMN triggered_at TYPE TIMESTAMP;
ALTER TABLE app_user ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE async_jobs ALTER COLUMN completed_at TYPE TIMESTAMP;
ALTER TABLE async_jobs ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE async_jobs ALTER COLUMN started_at TYPE TIMESTAMP;
ALTER TABLE bp_branch_executions ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE compliance_results ALTER COLUMN evaluated_at TYPE TIMESTAMP;
ALTER TABLE fund_commitments ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE fund_position_snapshots ALTER COLUMN as_of_timestamp TYPE TIMESTAMP;
ALTER TABLE fund_position_snapshots ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE job_exports ALTER COLUMN completed_at TYPE TIMESTAMP;
ALTER TABLE job_exports ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE job_exports ALTER COLUMN expires_at TYPE TIMESTAMP;
ALTER TABLE pushback_analysis ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE rebalance_audit ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE reconciliation_records ALTER COLUMN reconciled_at TYPE TIMESTAMP;
ALTER TABLE scenario_pnl ALTER COLUMN evaluated_at TYPE TIMESTAMP;
ALTER TABLE user_profiles ALTER COLUMN last_activity_at TYPE TIMESTAMP;
ALTER TABLE wasm_versions ALTER COLUMN build_time TYPE TIMESTAMP;
