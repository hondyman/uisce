-- Quick schema for testing BP triggers locally
\set ON_ERROR_STOP on

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Reuse migration
\i backend/db/migrations/2025_10_21_create_bp_triggers.sql
