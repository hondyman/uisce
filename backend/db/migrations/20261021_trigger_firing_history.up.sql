-- Add trigger_id to data_pipeline_runs to enable firing history per trigger.
-- A pipeline run now records which validation trigger dispatched it (if any).

ALTER TABLE data_pipeline_runs
    ADD COLUMN trigger_id UUID NULL REFERENCES validation_triggers(id);

CREATE INDEX IF NOT EXISTS idx_data_pipeline_runs_trigger_id
    ON data_pipeline_runs(trigger_id)
    WHERE trigger_id IS NOT NULL;

-- Index for last-fired query: join validation_triggers → data_pipeline_runs
-- on (id = trigger_id) ORDER BY start_time DESC LIMIT 1
CREATE INDEX IF NOT EXISTS idx_data_pipeline_runs_trigger_start
    ON data_pipeline_runs(trigger_id, start_time DESC);
