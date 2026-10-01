-- Migration: 20261122_001_three_tier_trend_storage.down.sql

DROP TABLE IF EXISTS mdm_eval.vendor_scorecard_history CASCADE;

ALTER TABLE mdm_eval.scoring_settings
    DROP COLUMN IF EXISTS watermark_hot_days,
    DROP COLUMN IF EXISTS watermark_warm_days;
