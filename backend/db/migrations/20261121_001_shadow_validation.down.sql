-- Migration: 20261121_001_shadow_validation.down.sql
-- Description: Rolls back shadow run log table.

DROP TABLE IF EXISTS mdm_eval.shadow_run_log CASCADE;
