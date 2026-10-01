-- 20261125_001_scoring_settings_quality_zones.down.sql
-- Remove configurable quality zone thresholds from mdm_eval.scoring_settings

ALTER TABLE mdm_eval.scoring_settings
    DROP COLUMN IF EXISTS quality_zone_high_threshold,
    DROP COLUMN IF EXISTS quality_zone_mid_threshold;
