-- 20261003_001_bof_consolidation.up.sql
-- Phase 1: Add missing columns to business_object_fields to make it the master table
-- consolidating from bo_fields. These columns are needed by the boresolver
-- to read from business_object_fields instead of bo_fields.
--
-- Columns added:
-- - display_name: human-readable label (maps to bo_fields.display_name)
-- - technical_name: physical/technical name (maps to bo_fields.technical_name)
-- - data_type: column data type like text, number, date (maps to bo_fields.type)
-- - is_required: field is required (maps to bo_fields.is_required)
-- - is_system: system-managed field (maps to bo_fields.is_system)
-- - description: field description (maps to bo_fields.description)
-- - reference_entity: reference BO for reference type fields (maps to bo_fields.reference_entity)
-- - display_order: ordering sequence (maps to bo_fields.sequence)
-- - section_name: UI section grouping (maps to bo_fields.section_name)
-- - default_value: default value expression (maps to bo_fields.default_value)
-- - validation_rules: JSON validation rules (maps to bo_fields.validation_rules)
-- - picklist_values: allowed picklist values (maps to bo_fields.picklist_values)

BEGIN;

-- Add missing columns to business_object_fields
ALTER TABLE public.business_object_fields
    ADD COLUMN IF NOT EXISTS display_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS technical_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS data_type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS is_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS reference_entity VARCHAR(255),
    ADD COLUMN IF NOT EXISTS display_order INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS section_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS default_value TEXT,
    ADD COLUMN IF NOT EXISTS validation_rules JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS picklist_values JSONB NOT NULL DEFAULT '{}';

-- Create index for lookups
CREATE INDEX IF NOT EXISTS idx_bof_display_order
    ON public.business_object_fields (tenant_id, bo_id, display_order);

COMMIT;
