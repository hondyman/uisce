-- Bind validation_triggers to an optional Data Pipeline Studio pipeline,
-- dispatched sync (inline, blocks the write on pipeline failure) or async
-- (via the outbox, never blocks the write) after the trigger's rule checks
-- pass. Both columns are nullable/optional — existing triggers are
-- unaffected (PipelineID nil means "no pipeline binding", fully backward
-- compatible).
ALTER TABLE validation_triggers
    ADD COLUMN IF NOT EXISTS pipeline_id UUID NULL,
    ADD COLUMN IF NOT EXISTS dispatch_mode TEXT NULL; -- 'sync' | 'async'; ignored when pipeline_id IS NULL

COMMENT ON COLUMN validation_triggers.pipeline_id IS 'Optional data_pipeline_definitions.id to run after this trigger''s rule checks pass.';
COMMENT ON COLUMN validation_triggers.dispatch_mode IS 'How pipeline_id is dispatched: sync (inline, blocking) or async (via outbox, non-blocking). Ignored when pipeline_id is NULL.';
