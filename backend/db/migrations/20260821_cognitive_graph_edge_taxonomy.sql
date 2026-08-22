-- Migration: 20260821_cognitive_graph_edge_taxonomy.sql
-- Purpose: Create closed-loop rejection store and seed 4-dimensional cognitive edge taxonomy
-- Date: 2026-08-21

BEGIN;

-- 1. Create Closed-Loop Rejection Store Table
CREATE TABLE IF NOT EXISTS public.catalog_edge_rejection_store (
    rejection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_node_id UUID NOT NULL REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    rejected_target_id UUID NOT NULL REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    edge_type_id UUID NOT NULL,
    rejected_by TEXT NOT NULL DEFAULT 'user',
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT catalog_rejection_unique UNIQUE(tenant_id, source_node_id, rejected_target_id, edge_type_id)
);

CREATE INDEX IF NOT EXISTS idx_rejection_lookup 
ON public.catalog_edge_rejection_store (tenant_id, source_node_id, rejected_target_id);

-- 2. Seed Cognitive Edge Taxonomy in Gold Copy and System Tenant
DO $$
DECLARE
    v_gold_tenant_id UUID;
    v_system_tenant_id UUID := '00000000-0000-0000-0000-000000000000';
BEGIN
    SELECT id INTO v_gold_tenant_id FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant_id IS NULL THEN
        SELECT id INTO v_gold_tenant_id FROM public.tenants ORDER BY created_at LIMIT 1;
    END IF;

    -- Ensure Gold Copy & System Tenant have all 10 Edge Types
    -- 1. Understanding
    INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, is_directed, config, created_at, updated_at)
    VALUES
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'IS_EQUIVALENT_TO', 'Exact functional and conceptual semantic equivalence across schemas', true, false, '{"confidence": "number", "dialect": "string"}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'HAS_SYNONYM', 'Lexical or industry alias denoting the identical underlying business term', true, false, '{"dialect": "string", "source": "string"}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'ALIAS_OF', 'Physical or vendor abbreviation alias of a canonical business term', true, true, '{"source": "string"}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET 
        description = EXCLUDED.description, 
        config = EXCLUDED.config, 
        is_active = EXCLUDED.is_active;

    -- 2. Differentiation
    INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, is_directed, config, created_at, updated_at)
    VALUES
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'DIFFERENTIATED_FROM', 'Links semantically adjacent concepts with explicit operational and lifecycle boundaries', true, false, '{"properties": {"distinction_rationale": {"type": "string"}, "when_to_use": {"type": "string"}, "validation_rules": {"type": "object"}}}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'IS_PEER_IDENTIFIER_OF', 'Links alternate identifier symbologies for equivalent domain entities', true, false, '{"properties": {"symbology_family": {"type": "string"}, "iso_standard": {"type": "string"}, "regex": {"type": "string"}}}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET 
        description = EXCLUDED.description, 
        config = EXCLUDED.config, 
        is_active = EXCLUDED.is_active;

    -- 3. Association
    INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, is_directed, config, created_at, updated_at)
    VALUES
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'IS_SPECIALIZATION_OF', 'Indicates a child subtype specialization of a general parent term', true, true, '{"properties": {"discriminator": {"type": "string"}, "hierarchy_level": {"type": "integer"}}}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'IS_GENERALIZATION_OF', 'Parent or umbrella business term overarching specialized terms', true, true, '{"properties": {"hierarchy_level": {"type": "integer"}}}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'SEE_ALSO', 'Associative contextual cross-reference between related domain terms', true, false, '{"association_type": "string", "strength": "number"}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'RELATES_TO', 'Associative semantic relationship in the business domain', true, false, '{"association_type": "string"}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET 
        description = EXCLUDED.description, 
        config = EXCLUDED.config, 
        is_active = EXCLUDED.is_active;

    -- 4. Suggestions
    INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active, is_directed, config, created_at, updated_at)
    VALUES
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'SUGGESTED_MAPPING_FOR', 'Ephemeral candidate edge proposed by AI or heuristic monitoring', true, true, '{"properties": {"confidence": {"type": "number"}, "suggested_by": {"type": "string"}, "match_reason": {"type": "string"}}}'::jsonb, NOW(), NOW()),
        (COALESCE(v_gold_tenant_id, v_system_tenant_id)::text, 'MAPS_TO', 'Active approved semantic mapping edge from physical column to business term', true, true, '{"properties": {"approved_by": {"type": "string"}}}'::jsonb, NOW(), NOW())
    ON CONFLICT (tenant_id, edge_type_name) DO UPDATE SET 
        description = EXCLUDED.description, 
        config = EXCLUDED.config, 
        is_active = EXCLUDED.is_active;

END $$;

COMMIT;
