-- Migration: 20260159_add_page_to_telemetry.sql
-- Goal: Track telemetry by page for SLO evaluation

ALTER TABLE planner_telemetry ADD COLUMN IF NOT EXISTS page_slug text;
CREATE INDEX IF NOT EXISTS idx_planner_telemetry_page ON planner_telemetry(page_slug) WHERE page_slug IS NOT NULL;
