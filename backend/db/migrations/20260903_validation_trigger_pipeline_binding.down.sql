ALTER TABLE validation_triggers
    DROP COLUMN IF EXISTS pipeline_id,
    DROP COLUMN IF EXISTS dispatch_mode;
