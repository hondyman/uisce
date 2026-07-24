-- Migration: 20260708_add_soft_delete_to_tenants.up.sql
-- Purpose: Add is_deleted and deleted_at columns to support soft delete for tenants.
-- Soft delete allows preserving tenant records for audit/history while hiding them
-- from the standard tenants list query.

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Index for fast filtering of active (non-deleted) tenants
CREATE INDEX IF NOT EXISTS idx_tenants_is_deleted
    ON tenants(is_deleted);

-- Partial index for the most common query: list active tenants
CREATE INDEX IF NOT EXISTS idx_tenants_active
    ON tenants(id)
    WHERE is_deleted = false;
