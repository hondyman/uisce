-- +goose Up
-- +goose StatementBegin

-- 1. Ensure `calculation_term` exists in catalog_node_type
INSERT INTO public.catalog_node_type (type_name, description, is_active)
VALUES ('calculation_term', 'A node representing a mathematical or business calculation with semantic dependencies', true)
ON CONFLICT (type_name) DO NOTHING;

-- 2. Insert the core `ct_nav` (Net Asset Value) calculation term
-- Using a deterministic UUID for easy referencing
INSERT INTO public.catalog_node (
    id, tenant_id, type_id, name, description, config, created_by, updated_by
)
SELECT 
    '00000000-0000-0000-0000-000000000001'::uuid, 
    NULL, -- Core system node
    cnt.id, 
    'ct_nav', 
    'Net Asset Value: Aggregate value of all held positions',
    '{"expression": "SUM(st_position_value)", "engine": "wazero"}',
    'system', 
    'system'
FROM public.catalog_node_type cnt
WHERE cnt.type_name = 'calculation_term'
ON CONFLICT (id) DO NOTHING;

-- 3. Ensure the dependency edge type exists (calc_depends_on_term)
INSERT INTO public.catalog_edge_type (type_name, description, is_active)
VALUES ('calc_depends_on_term', 'Calculation node depends on a semantic term node', true)
ON CONFLICT (type_name) DO NOTHING;

-- 4. Establish the `depends_on` edge between `ct_nav` and `st_position_value`
-- We find the 'st_position_value' node first
INSERT INTO public.catalog_edge (
    tenant_id, source_node_id, target_node_id, edge_type_id, properties, created_by
)
SELECT 
    NULL, -- Core system edge
    '00000000-0000-0000-0000-000000000001'::uuid, -- Source: ct_nav
    n.id, -- Target: st_position_value
    et.id,
    '{"weight": 1.0, "required": true}',
    'system'
FROM public.catalog_node n
CROSS JOIN public.catalog_edge_type et
WHERE n.name = 'st_position_value' 
  AND et.type_name = 'calc_depends_on_term'
ON CONFLICT (source_node_id, target_node_id, edge_type_id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove edge
DELETE FROM public.catalog_edge 
WHERE source_node_id = '00000000-0000-0000-0000-000000000001'::uuid;

-- Remove core calculation term
DELETE FROM public.catalog_node 
WHERE id = '00000000-0000-0000-0000-000000000001'::uuid;

-- Optionally remove types if they were newly created (usually we leave them to be safe)
-- DELETE FROM public.catalog_edge_type WHERE type_name = 'calc_depends_on_term';
-- DELETE FROM public.catalog_node_type WHERE type_name = 'calculation_term';

-- +goose StatementEnd
