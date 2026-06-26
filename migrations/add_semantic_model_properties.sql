-- Migration: Add properties to semantic_model node type
-- This adds the property schema definitions to enable the new semantic model properties

UPDATE public.catalog_node_type 
SET properties = '[
  {"name": "technical_name", "title": "Technical Name/ID", "data_type": "string", "order": 1, "required": false, "description": "Internal unique identifier"},
  {"name": "model_type", "title": "Model Type", "data_type": "string", "order": 2, "required": true, "input_type": "select", "options": ["core", "custom"], "default": "core", "description": "Core (read-only) or Custom (user-defined)"},
  {"name": "data_source_description", "title": "Data Source Description", "data_type": "string", "order": 3, "required": false, "description": "Description of underlying data source(s)"},
  {"name": "schema_table_reference", "title": "Schema/Table Reference", "data_type": "string", "order": 4, "required": false, "description": "Database schema.table or file paths"},
  {"name": "extends_model_id", "title": "Extends Model", "data_type": "string", "order": 5, "required": false, "input_type": "lookup", "lookup_type": "semantic_model", "description": "Reference to Core or Custom model this extends"},
  {"name": "linked_semantic_terms", "title": "Linked Semantic Terms", "data_type": "array", "order": 6, "required": false, "description": "List of semantic term IDs used in this model"},
  {"name": "overridden_properties", "title": "Overridden Term Properties", "data_type": "jsonb", "order": 7, "required": false, "description": "Override properties for inherited semantic terms"},
  {"name": "model_calculations", "title": "Model-Specific Calculations", "data_type": "jsonb", "order": 8, "required": false, "description": "Complex calculations combining semantic terms"}
]'::jsonb
WHERE catalog_type_name = 'semantic_model';

-- Verify the update
SELECT 
  catalog_type_name,
  jsonb_array_length(properties) as property_count,
  jsonb_pretty(properties) as properties_schema
FROM public.catalog_node_type 
WHERE catalog_type_name = 'semantic_model';

-- Add new edge types for semantic model relationships
INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, source_node_type_id, target_node_type_id) VALUES
('semantic_model_extends_edge', 'default', 'semantic_model_extends', 'Semantic Model Extends (Inheritance)', 'semantic_model_type', 'semantic_model_type'),
('semantic_model_links_to_edge', 'default', 'semantic_model_links_to', 'Semantic Model Links To Semantic Term', 'semantic_model_type', 'semantic_term_type')
ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;

-- Verify edge types were created
SELECT edge_type_name, description, source_node_type_id, target_node_type_id
FROM public.catalog_edge_types
WHERE edge_type_name IN ('semantic_model_extends', 'semantic_model_links_to');
