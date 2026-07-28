-- Migration: Polyglot CRUD and Bi-Temporal Mapping Layer Schema
-- Date: 2026-07-30
-- Purpose: Extend Business Object bindings for dual OLTP CRUD (writeable) and Bi-Temporal OLAP (datalake analytical) modes.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'binding_mode_enum') THEN
        CREATE TYPE binding_mode_enum AS ENUM ('OLTP_CRUD', 'OLAP_READONLY', 'BI_TEMPORAL_OLAP');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS public.business_object_bindings (
    binding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    bo_id VARCHAR(128) NOT NULL,
    binding_name VARCHAR(100) NOT NULL,
    binding_mode binding_mode_enum NOT NULL DEFAULT 'OLTP_CRUD',
    datasource_id VARCHAR(128) NOT NULL,
    physical_table_name VARCHAR(255) NOT NULL,
    
    -- Bi-Temporal Column Mappings (Null if binding_mode == OLTP_CRUD)
    valid_time_start_col VARCHAR(100),       -- e.g., 'effective_from'
    valid_time_end_col VARCHAR(100),         -- e.g., 'effective_to'
    transaction_time_start_col VARCHAR(100), -- e.g., 'sys_start_time'
    transaction_time_end_col VARCHAR(100),   -- e.g., 'sys_end_time'

    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    
    CONSTRAINT uk_bo_binding_name UNIQUE (tenant_id, bo_id, binding_name)
);

CREATE INDEX IF NOT EXISTS idx_bo_bindings_lookup ON public.business_object_bindings(tenant_id, bo_id, binding_mode);
