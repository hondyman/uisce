-- Migration: 20260708_add_soft_delete_to_tenants.down.sql
-- Purpose: Rollback the soft delete columns.

DROP INDEX IF EXISTS idx_tenants_is_deleted;
DROP INDEX IF EXISTS idx_tenants_active;

ALTER TABLE tenants
DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE tenants
DROP COLUMN IF EXISTS is_deleted;
