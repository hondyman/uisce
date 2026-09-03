-- Migration: 20261017_forecast_feedback_loop.down.sql
-- Rollback for Forecast Outcome Feedback Loop & Calibration Engine

DROP TABLE IF EXISTS finops.forecast_calibration_state CASCADE;
DROP TABLE IF EXISTS finops.forecast_feedback CASCADE;
