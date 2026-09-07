-- ============================================================================
-- Migration 001: Phase 3.21 Initial Schema
-- ============================================================================
-- Initial creation of feature catalog, materialization, drift detection,
-- quality checks, importance scoring, and governance tables.
--
-- Migration metadata:
-- Version: 3.21.0
-- Created: 2026-02-09
-- Author: Phase 3.21 Implementation
-- Description: Complete feature engineering schema with 10 core tables
-- ============================================================================

-- Migration status tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    description TEXT,
    applied_at TIMESTAMPTZ DEFAULT NOW(),
    duration_ms INT
);

-- Record this migration
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('3.21.0', 'Initial feature engineering schema', NOW())
ON CONFLICT DO NOTHING;

-- ============================================================================
-- MIGRATION 001: Apply phase_3_21_schema.sql
-- ============================================================================
-- The main schema is defined in phase_3_21_schema.sql
-- This migration file is a reference; use init_schema.sh to apply all tables.
-- ============================================================================

-- Verify migration success
SELECT version, description, applied_at
FROM schema_migrations
WHERE version = '3.21.0';
