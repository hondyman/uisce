-- Migration: 20260820_bo_subtypes_and_taxonomy_grouping.up.sql

-- 1. Extend business_objects to support polymorphic subtyping
ALTER TABLE public.business_objects 
ADD COLUMN IF NOT EXISTS discriminator_field VARCHAR(100) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS subtypes_config JSONB DEFAULT '{}'::jsonb;

-- 2. Extend business_object_fields to support Subtype Scoping and Taxonomy Overrides
ALTER TABLE public.business_object_fields 
ADD COLUMN IF NOT EXISTS subtype_scope VARCHAR(50) DEFAULT 'CORE',
ADD COLUMN IF NOT EXISTS custom_ui_group VARCHAR(100) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS ui_sequence INT DEFAULT 100;

-- Index for high-speed tenant-scoped subtype filtering
CREATE INDEX IF NOT EXISTS idx_bof_bo_subtype_tenant 
ON public.business_object_fields(tenant_id, bo_id, subtype_scope);

-- 3. Seed Discriminator & Subtypes for Security Master Business Object
UPDATE public.business_objects
SET 
    discriminator_field = 'sec_typ_cd',
    subtypes_config = '{
        "EQUITY": { "displayName": "Equity", "color": "#10B981", "icon": "trending-up" },
        "FIXED_INCOME": { "displayName": "Fixed Income", "color": "#3B82F6", "icon": "landmark" },
        "DERIVATIVE": { "displayName": "Derivative", "color": "#8B5CF6", "icon": "zap" }
    }'::jsonb
WHERE key = 'security';
