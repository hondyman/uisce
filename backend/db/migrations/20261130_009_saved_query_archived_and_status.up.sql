-- Migration: saved_query soft-delete and lifecycle status
-- Date: 20261130

ALTER TABLE data_explorer.saved_query
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

CREATE INDEX IF NOT EXISTS idx_de_saved_query_tenant_status
    ON data_explorer.saved_query(tenant_id, status) WHERE archived_at IS NULL;
