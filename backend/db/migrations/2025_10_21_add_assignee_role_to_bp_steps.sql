-- Migration: Add assignee_role to bp_steps
-- Up: add column if not exists and backfill from config JSONB where possible
-- Down: drop the column

-- Up
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='bp_steps' AND column_name='assignee_role'
    ) THEN
        ALTER TABLE bp_steps ADD COLUMN assignee_role TEXT;
    END IF;
END$$;

-- Attempt a best-effort backfill: if config JSONB contains common keys, populate assignee_role
-- This uses COALESCE to pick the first non-null candidate
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='bp_steps' AND column_name='config'
    ) THEN
        UPDATE bp_steps
        SET assignee_role = COALESCE(
            (config->>'assigneeRole'),
            (config->>'assignee_role'),
            (config->>'assignee')
        )
        WHERE assignee_role IS NULL AND config IS NOT NULL;
    END IF;
END$$;

-- Down
-- To rollback, run the following (manual down-step):
-- ALTER TABLE bp_steps DROP COLUMN IF EXISTS assignee_role;
