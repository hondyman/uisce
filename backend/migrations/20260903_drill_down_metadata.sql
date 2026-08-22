-- 20260903_drill_down_metadata.sql
-- Aggregate-to-Detail Drill-Through Metadata and Relationship Mappings

CREATE SCHEMA IF NOT EXISTS semantic_drill;

-- 1. Aggregate Calculation to Granular Fact Mapping Registry
CREATE TABLE IF NOT EXISTS semantic_drill.calculation_drill_paths (
    path_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    aggregated_term_node_id UUID NOT NULL,
    granular_bo_id UUID NOT NULL,                        -- Target Business Object for detail view (e.g., TaxLots)
    drill_sql_template TEXT NOT NULL,                     -- Parameterized query template for detail fetching
    grouping_dimensions JSONB DEFAULT '[]'::jsonb,       -- Dimensions required to slice the aggregate
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_term_drill UNIQUE (tenant_id, aggregated_term_node_id)
);

CREATE INDEX IF NOT EXISTS idx_drill_paths_lookup 
ON semantic_drill.calculation_drill_paths (tenant_id, aggregated_term_node_id);
