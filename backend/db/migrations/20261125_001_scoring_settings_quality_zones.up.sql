-- 20261125_001_scoring_settings_quality_zones.up.sql
-- Add configurable quality zone thresholds to mdm_eval.scoring_settings

ALTER TABLE mdm_eval.scoring_settings
    ADD COLUMN IF NOT EXISTS quality_zone_high_threshold NUMERIC(5,2) NOT NULL DEFAULT 70.0,
    ADD COLUMN IF NOT EXISTS quality_zone_mid_threshold NUMERIC(5,2) NOT NULL DEFAULT 50.0;
