-- Register COLUMN_REFERENCES_COLUMN edge type for column-level FK reference edges
-- This edge type is used by the ColumnReferenceScanner to create edges between
-- individual column nodes (one edge per FK column pair), distinct from the
-- AnsiScanner's table-level 'foreign_key' edges.

DO $do$
DECLARE
    _tenant uuid;
    _col_type_id uuid;
BEGIN
    SET search_path TO public;
    SELECT id INTO _tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    SELECT id INTO _col_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'column' LIMIT 1;

    IF _tenant IS NOT NULL AND _col_type_id IS NOT NULL THEN
        INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, source_node_type_id, target_node_type_id, is_directed, is_active, config)
        VALUES (
            'c0c1c2c3-d4e5-f6a7-b8c9-d0e1f2a3b4c5'::uuid,
            _tenant,
            'COLUMN_REFERENCES_COLUMN',
            'A column references another column via FK, code convention, or enum. One edge per column pair.',
            _col_type_id,
            _col_type_id,
            true,   -- is_directed: source_column -> target_column
            true,   -- is_active
            '{
                "reference_kind": {
                    "type": "string",
                    "enum": ["FK_UUID", "FK_CODE", "ENUM", "HIERARCHY_PARENT", "SELF"],
                    "description": "Kind of reference: FK_UUID=physical FK, FK_CODE=varchar code ref, ENUM=check constraint, HIERARCHY_PARENT=self-ref parent, SELF=other self-ref"
                },
                "cardinality": {
                    "type": "string",
                    "enum": ["MANY_TO_ONE", "ONE_TO_ONE", "ONE_TO_MANY", "MANY_TO_MANY"],
                    "description": "Relationship cardinality"
                },
                "is_enforced": {
                    "type": "boolean",
                    "description": "True if backed by a DB constraint (FK or CHECK)"
                },
                "load_order": {
                    "type": "integer",
                    "description": "Suggested load order; lower=earlier (ref tables before mdm)"
                },
                "constraint_name": {
                    "type": "string",
                    "description": "Name of the FK constraint (if is_enforced=true)"
                },
                "description": {
                    "type": "string",
                    "description": "Human-readable description of this reference"
                }
            }'::jsonb
        )
        ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;

        RAISE NOTICE 'Registered COLUMN_REFERENCES_COLUMN edge type for tenant %', _tenant;
    ELSE
        RAISE NOTICE 'Skipped: could not find gold copy tenant (id=%) or column node type (id=%)', _tenant, _col_type_id;
    END IF;
END
$do$;
