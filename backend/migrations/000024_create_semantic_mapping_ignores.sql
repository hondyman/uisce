-- Migration: 000024_create_semantic_mapping_ignores.sql
-- Created: 2025-10-05
-- Purpose: CREATE TABLE IF NOT EXISTS to persist ignored semantic mapping suggestions

-- Ensure pgcrypto (for gen_random_uuid) is available. This is idempotent.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS public.semantic_mapping_ignores (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_datasource_id uuid,
    tenant_id uuid,
    database_column_node_id uuid NOT NULL,
    ignored_term text NOT NULL,
    created_at timestamptz DEFAULT now()
);

-- Prevent duplicate ignore entries for the same column + term within a datasource
CREATE UNIQUE INDEX IF NOT EXISTS idx_semantic_mapping_ignores_unique
    ON public.semantic_mapping_ignores (tenant_datasource_id, database_column_node_id, ignored_term);

-- Index for fast lookup by column node id
CREATE INDEX IF NOT EXISTS idx_semantic_mapping_ignores_column
    ON public.semantic_mapping_ignores (database_column_node_id);

-- Add comment for documentation
COMMENT ON TABLE public.semantic_mapping_ignores IS 'Records suggestions ignored by users for specific database columns to avoid repeated suggestions.';
