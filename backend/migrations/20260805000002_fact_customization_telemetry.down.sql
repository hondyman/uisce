-- 20260805000002_fact_customization_telemetry.down.sql

BEGIN;

DROP INDEX IF EXISTS idx_fact_telemetry_confidence;
DROP INDEX IF EXISTS idx_fact_telemetry_recommended;
DROP TABLE IF EXISTS fact_customization_telemetry;

COMMIT;
