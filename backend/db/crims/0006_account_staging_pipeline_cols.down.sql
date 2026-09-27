DROP INDEX IF EXISTS staging.idx_staging_account_load_run;
ALTER TABLE staging.account_data DROP COLUMN IF EXISTS _source_row_num;
ALTER TABLE staging.account_data DROP COLUMN IF EXISTS _load_run_id;
ALTER TABLE staging.account_data ALTER COLUMN source_system DROP DEFAULT;
