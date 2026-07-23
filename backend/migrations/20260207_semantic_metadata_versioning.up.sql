-- Migration: Semantic Layer Metadata Versioning
-- Date: 2026-02-07
-- Description: Adds tables for semantic layer versioning and field aliases.
-- This enables deterministic metadata tracking, audit trails, and safe field renames.

-- Semantic metadata versioning table
-- Tracks all changes to business object metadata with complete before/after state
CREATE TABLE IF NOT EXISTS metadata_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_object_id UUID NOT NULL,
    version INT NOT NULL,
    
    -- Change tracking
    change_type TEXT NOT NULL, -- field_added, field_renamed, field_removed, field_type_changed, physical_mapping_changed, etc.
    change_detail JSONB, -- Additional metadata about the change
    previous_value JSONB, -- Full previous state
    new_value JSONB, -- Full new state
    
    -- Audit trail
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT, -- User who made the change
    
    UNIQUE(tenant_id, business_object_id, version)
);

CREATE INDEX IF NOT EXISTS idx_metadata_versions_bo ON metadata_versions(tenant_id, business_object_id);
CREATE INDEX IF NOT EXISTS idx_metadata_versions_created ON metadata_versions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_metadata_versions_change_type ON metadata_versions(change_type);

-- Field aliases table for backward-compatible field renames
-- When a field is renamed, old name is stored here to maintain compatibility
CREATE TABLE IF NOT EXISTS field_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    field_id UUID NOT NULL,
    
    -- The old name that was replaced
    old_name TEXT NOT NULL,
    
    -- Who renamed it and when
    renamed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    renamed_by TEXT, -- User who renamed the field
    
    -- Status tracking
    is_active BOOLEAN NOT NULL DEFAULT true,
    description TEXT, -- Reason for the rename
    
    UNIQUE(tenant_id, field_id, old_name)
);

CREATE INDEX IF NOT EXISTS idx_field_aliases_field ON field_aliases(tenant_id, field_id);
CREATE INDEX IF NOT EXISTS idx_field_aliases_old_name ON field_aliases(tenant_id, old_name);
CREATE INDEX IF NOT EXISTS idx_field_aliases_active ON field_aliases(is_active) WHERE is_active = true;

-- Add RLS policies for semantic metadata tables
ALTER TABLE metadata_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE field_aliases ENABLE ROW LEVEL SECURITY;

-- RLS Policy for metadata_versions: users can only see their tenant's versions
CREATE POLICY metadata_versions_tenant_isolation ON metadata_versions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- RLS Policy for field_aliases: users can only see their tenant's aliases
CREATE POLICY field_aliases_tenant_isolation ON field_aliases
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Grant permissions
GRANT SELECT, INSERT, UPDATE ON metadata_versions TO authenticated;
GRANT SELECT, INSERT, UPDATE ON field_aliases TO authenticated;
