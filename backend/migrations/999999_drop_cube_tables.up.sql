-- Drop all cube-related tables (part of cube removal)
-- These tables were created by migrations:
--   000058_semantic_layer_platform.up.sql
--   000059_core_vs_custom.up.sql
--   000037_cube_model_builder.up.sql
--   cube_model_catalog_schema.sql

DROP TABLE IF EXISTS semantic_cubes_v2 CASCADE;
DROP TABLE IF EXISTS semantic_dimensions_v2 CASCADE;
DROP TABLE IF EXISTS semantic_measures_v2 CASCADE;
DROP TABLE IF EXISTS semantic_pre_aggregations_v2 CASCADE;
DROP TABLE IF EXISTS semantic_query_cache_v2 CASCADE;
DROP TABLE IF EXISTS semantic_query_history_v2 CASCADE;
DROP TABLE IF EXISTS semantic_cube_cache CASCADE;
DROP TABLE IF EXISTS cube_core_models CASCADE;
DROP TABLE IF EXISTS cube_core_measures CASCADE;
DROP TABLE IF EXISTS cube_core_dimensions CASCADE;
DROP TABLE IF EXISTS cube_custom_models CASCADE;
DROP TABLE IF EXISTS cube_custom_measures CASCADE;
DROP TABLE IF EXISTS cube_custom_dimensions CASCADE;
DROP TABLE IF EXISTS cube_security_policies CASCADE;
DROP TABLE IF EXISTS cube_security_context_cache CASCADE;
DROP TABLE IF EXISTS cube_model_wizard_state CASCADE;
DROP TABLE IF EXISTS cube_model_generation_history CASCADE;
DROP TABLE IF EXISTS cube_preagg_suggestions CASCADE;
