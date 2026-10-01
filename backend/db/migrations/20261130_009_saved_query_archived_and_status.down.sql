-- Migration: rollback saved_query soft-delete and status
-- Date: 20261130

DROP INDEX IF EXISTS data_explorer.idx_de_saved_query_tenant_status;

ALTER TABLE data_explorer.saved_query
    DROP COLUMN IF EXISTS archived_at,
    DROP COLUMN IF EXISTS status;
