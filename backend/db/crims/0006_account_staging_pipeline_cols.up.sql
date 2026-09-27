-- Make staging.account_data compatible with datapipeline staging_sink
-- (requires _load_run_id + _source_row_num like staging.ff_product).

ALTER TABLE staging.account_data
    ADD COLUMN IF NOT EXISTS _load_run_id uuid,
    ADD COLUMN IF NOT EXISTS _source_row_num integer;

-- Pipeline loads set source_system from staging_sink SourceCd when mapped;
-- keep a safe default so NOT NULL still holds for identity-mapped loads.
ALTER TABLE staging.account_data
    ALTER COLUMN source_system SET DEFAULT 'PIPELINE';

CREATE INDEX IF NOT EXISTS idx_staging_account_load_run
    ON staging.account_data (_load_run_id)
    WHERE _load_run_id IS NOT NULL;
