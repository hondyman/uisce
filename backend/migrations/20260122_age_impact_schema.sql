-- Migration: Define AGE Graph Schema for Impact Analysis
-- Date: 2026-01-22

-- This migration assumes the AGE extension is already enabled and the graph 'semantic_lineage' exists.
-- It defines the node labels and edge types required for the dynamic impact analysis.

DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'create_graph') THEN
    SET search_path = ag_catalog, "$user", public;

-- Ensure the graph exists (idempotent check)
SELECT create_graph('semantic_lineage')
WHERE NOT EXISTS (SELECT 1 FROM ag_graph WHERE name = 'semantic_lineage');

-- Create Label: BUSINESS_OBJECT
SELECT create_vlabel('semantic_lineage', 'BUSINESS_OBJECT')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'BUSINESS_OBJECT' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: BO_FIELD
SELECT create_vlabel('semantic_lineage', 'BO_FIELD')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'BO_FIELD' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: SEMANTIC_TERM
SELECT create_vlabel('semantic_lineage', 'SEMANTIC_TERM')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'SEMANTIC_TERM' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: DB_COLUMN
SELECT create_vlabel('semantic_lineage', 'DB_COLUMN')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'DB_COLUMN' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: API_ENDPOINT
SELECT create_vlabel('semantic_lineage', 'API_ENDPOINT')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'API_ENDPOINT' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: BI_ARTIFACT
SELECT create_vlabel('semantic_lineage', 'BI_ARTIFACT')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'BI_ARTIFACT' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: AI_ARTIFACT
SELECT create_vlabel('semantic_lineage', 'AI_ARTIFACT')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'AI_ARTIFACT' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: ACCESS_RULE
SELECT create_vlabel('semantic_lineage', 'ACCESS_RULE')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'ACCESS_RULE' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Label: CALCULATION_TERM
SELECT create_vlabel('semantic_lineage', 'CALCULATION_TERM')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'CALCULATION_TERM' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: HAS_FIELD (BO -> BO_FIELD)
SELECT create_elabel('semantic_lineage', 'HAS_FIELD')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'HAS_FIELD' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: BACKED_BY_TERM (BO_FIELD -> SEMANTIC_TERM)
SELECT create_elabel('semantic_lineage', 'BACKED_BY_TERM')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'BACKED_BY_TERM' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: BACKED_BY_CALC (BO_FIELD -> CALCULATION_TERM)
SELECT create_elabel('semantic_lineage', 'BACKED_BY_CALC')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'BACKED_BY_CALC' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: MAPPED_TO_COLUMN (SEMANTIC_TERM -> DB_COLUMN)
SELECT create_elabel('semantic_lineage', 'MAPPED_TO_COLUMN')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'MAPPED_TO_COLUMN' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: EXPOSED_VIA_API (BO -> API_ENDPOINT)
SELECT create_elabel('semantic_lineage', 'EXPOSED_VIA_API')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'EXPOSED_VIA_API' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: USED_IN_BI (BO -> BI_ARTIFACT)
SELECT create_elabel('semantic_lineage', 'USED_IN_BI')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'USED_IN_BI' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: USED_IN_AI (BO -> AI_ARTIFACT)
SELECT create_elabel('semantic_lineage', 'USED_IN_AI')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'USED_IN_AI' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: APPLIES_TO_BO (ACCESS_RULE -> BO)
SELECT create_elabel('semantic_lineage', 'APPLIES_TO_BO')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'APPLIES_TO_BO' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: MASKS_TERM (ACCESS_RULE -> SEMANTIC_TERM)
SELECT create_elabel('semantic_lineage', 'MASKS_TERM')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'MASKS_TERM' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: FILTERS_ON_TERM (ACCESS_RULE -> SEMANTIC_TERM)
SELECT create_elabel('semantic_lineage', 'FILTERS_ON_TERM')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'FILTERS_ON_TERM' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));

-- Create Edge: DEPENDS_ON (CALCULATION_TERM -> SEMANTIC_TERM/CALCULATION_TERM)
SELECT create_elabel('semantic_lineage', 'DEPENDS_ON')
WHERE NOT EXISTS (SELECT 1 FROM ag_label WHERE name = 'DEPENDS_ON' AND graph = (SELECT graphid FROM ag_graph WHERE name = 'semantic_lineage'));
  ELSE
    RAISE NOTICE 'AGE extension not available - skipping AGE graph setup';
  END IF;
END
$do$;
