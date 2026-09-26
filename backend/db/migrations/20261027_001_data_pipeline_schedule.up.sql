-- Optional run schedule for a data pipeline: {"cron","timezone","enabled"}.
-- The live schedule is a Temporal Schedule; this column is its source of truth.
ALTER TABLE data_pipeline_definitions ADD COLUMN IF NOT EXISTS schedule JSONB;
