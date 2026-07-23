-- Migration: 000025_create_semantic_mapping_suggestions.sql
-- Created: 2025-10-06
-- Purpose: Create table to store generated semantic mapping suggestions

-- Ensure pgcrypto (for gen_random_uuid) is available. This is idempotent.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Table to store pre-calculated or generated semantic mapping suggestions
CREATE TABLE IF NOT EXISTS public.semantic_mapping_suggestions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_datasource_id uuid NOT NULL,
    database_column_node_id uuid NOT NULL,
    semantic_term_node_id uuid NOT NULL,
    confidence_score numeric(5, 4) NOT NULL,
    source text NOT NULL, -- e.g., 'ml_model_v1', 'heuristic_v2'
    created_at timestamptz DEFAULT now(),
    -- Add foreign key constraints if catalog tables are in the same DB
    -- CONSTRAINT fk_column_node FOREIGN KEY (database_column_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    -- CONSTRAINT fk_term_node FOREIGN KEY (semantic_term_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE,
    CONSTRAINT uq_suggestion UNIQUE (tenant_datasource_id, database_column_node_id, semantic_term_node_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_suggestions_column_id ON public.semantic_mapping_suggestions (database_column_node_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_term_id ON public.semantic_mapping_suggestions (semantic_term_node_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_confidence ON public.semantic_mapping_suggestions (confidence_score DESC);

COMMENT ON TABLE public.semantic_mapping_suggestions IS 'Stores potential semantic mappings between database columns and semantic terms, along with a confidence score.';

-- Helper function to generate some basic suggestions for demonstration.
-- In a real system, this would be part-of a more complex backend process.
CREATE OR REPLACE FUNCTION public.generate_initial_suggestions(p_datasource_id uuid)
RETURNS void AS $$
BEGIN
    -- A simple heuristic: suggest a semantic term if its name is found in the column name.
    -- This is a basic example; real-world logic would be more sophisticated (e.g., using ML).
    INSERT INTO public.semantic_mapping_suggestions (
        tenant_datasource_id,
        database_column_node_id,
        semantic_term_node_id,
        confidence_score,
        source
    )
    SELECT
        c.tenant_datasource_id,
        c.id,
        st.id,
        0.75, -- Assign a default confidence score
        'heuristic_v1_name_matching'
    FROM public.catalog_node c
    JOIN public.catalog_node st
        ON c.tenant_datasource_id = st.tenant_datasource_id
        AND c.node_type_id = 'a64c1011-16e8-4ddf-b447-363bf8e15c9a' -- 'column' type
        AND st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098' -- 'semantic_term' type
        AND c.node_name ILIKE '%' || st.node_name || '%'
    WHERE c.tenant_datasource_id = p_datasource_id
    ON CONFLICT (tenant_datasource_id, database_column_node_id, semantic_term_node_id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;