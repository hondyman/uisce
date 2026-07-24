-- Add semantic term dimension properties to catalog_node_type
-- These properties define the schema for semantic term dimensions in Cube.js style

DO $$
DECLARE
    semantic_term_type_id UUID;
BEGIN
    -- Find the semantic_term node type
    SELECT id INTO semantic_term_type_id
    FROM catalog_node_type
    WHERE catalog_type_name = 'semantic_term'
    LIMIT 1;
    
    IF semantic_term_type_id IS NULL THEN
        RAISE NOTICE 'semantic_term node type not found, skipping property addition';
        RETURN;
    END IF;
    
    -- Update the properties JSONB to include semantic term dimension properties
    UPDATE catalog_node_type
    SET properties = jsonb_build_object(
        'name', jsonb_build_object(
            'type', 'string',
            'required', true,
            'control_type', 'text',
            'description', 'The unique identifier for the dimension, which must be distinct within the cube'
        ),
        'sql', jsonb_build_object(
            'type', 'string',
            'required', true,
            'control_type', 'textarea',
            'description', 'The SQL expression that defines how the dimension value is derived from the underlying data source',
            'placeholder', '${TABLE}.column_name'
        ),
        'type', jsonb_build_object(
            'type', 'string',
            'required', true,
            'control_type', 'select',
            'lookup_type', 'dimension_type',
            'description', 'The data type of the dimension (e.g., string, number, boolean, time, geo)'
        ),
        'title', jsonb_build_object(
            'type', 'string',
            'required', false,
            'control_type', 'text',
            'description', 'Optional; provides a human-readable name for display purposes'
        ),
        'description', jsonb_build_object(
            'type', 'string',
            'required', false,
            'control_type', 'textarea',
            'description', 'Optional; a human-readable description for documentation and tooling'
        ),
        'order', jsonb_build_object(
            'type', 'string',
            'required', false,
            'control_type', 'select',
            'lookup_type', 'sort_order',
            'description', 'Optional; sets the default sort order for the dimension (asc or desc)'
        ),
        'primary_key', jsonb_build_object(
            'type', 'boolean',
            'required', false,
            'control_type', 'toggle',
            'description', 'Optional; marks the dimension as a primary key for the cube, which affects joins and deduplication'
        ),
        'case', jsonb_build_object(
            'type', 'object',
            'required', false,
            'control_type', 'sql_editor',
            'description', 'Optional; allows for conditional logic to map SQL values to labels using CASE statement'
        ),
        'granularities', jsonb_build_object(
            'type', 'array',
            'required', false,
            'control_type', 'multiselect',
            'lookup_type', 'time_granularity',
            'description', 'For time dimensions, specifies available time granularities (e.g., year, month, day)'
        ),
        'sub_query', jsonb_build_object(
            'type', 'boolean',
            'required', false,
            'control_type', 'toggle',
            'description', 'Optional; enables referencing a measure from another cube as a dimension'
        ),
        'latitude', jsonb_build_object(
            'type', 'object',
            'required', false,
            'control_type', 'group',
            'description', 'Required for geo-type dimensions to specify latitude coordinate',
            'properties', jsonb_build_object(
                'sql', jsonb_build_object('type', 'string', 'required', true, 'control_type', 'textarea')
            )
        ),
        'longitude', jsonb_build_object(
            'type', 'object',
            'required', false,
            'control_type', 'group',
            'description', 'Required for geo-type dimensions to specify longitude coordinate',
            'properties', jsonb_build_object(
                'sql', jsonb_build_object('type', 'string', 'required', true, 'control_type', 'textarea')
            )
        )
    ),
    updated_at = NOW()
    WHERE id = semantic_term_type_id;
    
    RAISE NOTICE 'Added semantic term dimension properties to node type: %', semantic_term_type_id;
END $$;
