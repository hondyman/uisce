-- Migration: 20260904_001_capture_unmanaged_schema_drift.down.sql
-- Description: Revert is_gold_copy column added to tenants

-- Only run if the column was added by this migration and is not referenced
-- by any existing constraint or index (it isn't — it's a plain boolean).
-- The column was confirmed nullable with no NOT NULL constraint in the live DB.

ALTER TABLE public.tenants DROP COLUMN IF EXISTS is_gold_copy;
