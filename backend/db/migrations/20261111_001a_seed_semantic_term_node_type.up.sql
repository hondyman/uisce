-- Seed the semantic_term catalog node type where it is missing, ahead of
-- 20261111_002_security_attribute_defs_and_graph and 20261111_004_party_attribute_defs_graph_pipelines.
--
-- Both look up catalog_node_types.catalog_type_name = 'semantic_term' and, for
-- attribute_def rows, insert semantic_survivorship_rules with the resulting term
-- id without a NULL guard. alpha has had this row since 2025-08 (it predates the
-- migrations directory), but backend-gated-tests restores a schema-only snapshot,
-- so there it is absent and both migrations fail with
-- null value in column "semantic_term_id" of relation "semantic_survivorship_rules".
--
-- Row copied verbatim from alpha (id 820b942a-..., description, config; is_active
-- NULL as on alpha), owned by the gold-copy tenant. Inserted only when no
-- semantic_term type exists and a gold-copy tenant does: a no-op on alpha.
DO $st$
DECLARE
    _gold uuid;
BEGIN
    IF EXISTS (SELECT 1 FROM public.catalog_node_types
                WHERE catalog_type_name IN ('semantic_term', 'SEMANTIC_TERM')) THEN
        RETURN;
    END IF;
    SELECT id INTO _gold FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF _gold IS NULL THEN
        RAISE NOTICE 'semantic_term node type not seeded: no gold-copy tenant';
        RETURN;
    END IF;

    INSERT INTO public.catalog_node_types (
        id, tenant_id, catalog_type_name, description, is_active, config, properties
    ) VALUES (
        '820b942a-9c9e-4abc-acdc-84616db33098'::uuid,
        _gold,
        'semantic_term',
        $d$A semantic term is a named, governed business concept with an authoritative definition, data type, lineage, and executable expression that can be reused across datasets, business objects, calculations, and applications.$d$,
        NULL,
        $cfg${
    "sql": {
        "type": "string",
        "required": true,
        "description": "The SQL expression that defines how the dimension value is derived from the underlying data source",
        "placeholder": "${TABLE}.column_name",
        "control_type": "textarea"
    },
    "case": {
        "type": "object",
        "required": false,
        "description": "Optional; allows for conditional logic to map SQL values to labels using CASE statement",
        "control_type": "sql_editor"
    },
    "name": {
        "type": "string",
        "required": true,
        "description": "The unique identifier for the dimension, which must be distinct within the cube",
        "control_type": "text"
    },
    "type": {
        "type": "string",
        "required": true,
        "description": "The data type of the dimension (e.g., string, number, boolean, time, geo)",
        "lookup_type": "dimension_type",
        "control_type": "select"
    },
    "color": "#3B82F6",
    "order": {
        "type": "string",
        "required": false,
        "description": "Optional; sets the default sort order for the dimension (asc or desc)",
        "lookup_type": "sort_order",
        "control_type": "select"
    },
    "title": {
        "type": "string",
        "required": false,
        "description": "Optional; provides a human-readable name for display purposes",
        "control_type": "text"
    },
    "latitude": {
        "type": "object",
        "required": false,
        "properties": {
            "sql": {
                "type": "string",
                "required": true,
                "control_type": "textarea"
            }
        },
        "description": "Required for geo-type dimensions to specify latitude coordinate",
        "control_type": "group"
    },
    "longitude": {
        "type": "object",
        "required": false,
        "properties": {
            "sql": {
                "type": "string",
                "required": true,
                "control_type": "textarea"
            }
        },
        "description": "Required for geo-type dimensions to specify longitude coordinate",
        "control_type": "group"
    },
    "sub_query": {
        "type": "boolean",
        "required": false,
        "description": "Optional; enables referencing a measure from another cube as a dimension",
        "control_type": "toggle"
    },
    "properties": [
        {
            "name": "term_type",
            "label": "Term Type",
            "order": 0,
            "options": [
                "ATTRIBUTE",
                "MEASURE",
                "KEY",
                "TIME",
                "CALCULATION"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "data_type",
            "label": "Data Type",
            "order": 1,
            "options": [
                "STRING",
                "INTEGER",
                "DECIMAL",
                "DATE",
                "BOOLEAN",
                "UUID",
                "COMPOSITE"
            ],
            "nullable": false,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "aggregation_type",
            "label": "Agg Type",
            "order": 2,
            "options": [
                "NONE",
                "SUM",
                "AVG",
                "COUNT",
                "MIN",
                "MAX"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "is_calculated",
            "label": "Is Calculated",
            "order": 3,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "calculation_type",
            "label": "Calc Type",
            "order": 4,
            "options": [
                "SQL",
                "FUNCTION",
                "CONSTANT",
                "DERIVED"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "identity_role",
            "label": "Identity Role",
            "order": 5,
            "options": [
                "NONE",
                "BUSINESS_KEY",
                "SEMANTIC_ID",
                "COMPONENT"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "is_imput_to_calc",
            "label": "Input to Calc",
            "order": 6,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "is_output",
            "label": "Is Calc Output",
            "order": 7,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "format_type",
            "label": "Format",
            "order": 8,
            "options": [
                "CURRENCY",
                "PERCENT",
                "NUMBER",
                "DATE"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "default_grain",
            "label": "Grain",
            "order": 9,
            "options": [
                "ROW",
                "AGGREGATED",
                "TIME_SERIES"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "is_system_generated",
            "label": "System Generated",
            "order": 10,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "is_deterministic",
            "label": "Determinestic",
            "order": 11,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "is_stable",
            "label": "Is Stable",
            "order": 12,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "resolution_strategy",
            "label": "resolution Strategy",
            "order": 13,
            "options": [
                "LOOKUP_OR_CREATE"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "is_primary_identifier",
            "label": "Primary Identifier",
            "order": 14,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        },
        {
            "name": "identity_scope",
            "label": "Identity Scope",
            "order": 15,
            "options": [
                "ENTITY",
                "VERSION",
                "SOURCE"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "surrogate_strategy",
            "label": "Surrogate Strategy",
            "order": 16,
            "options": [
                "UUIDv4",
                "hash-based",
                "UUIDv7"
            ],
            "nullable": true,
            "data_type": "text",
            "input_type": "select"
        },
        {
            "name": "is_exposed",
            "label": "Is Exposed",
            "order": 17,
            "nullable": true,
            "data_type": "boolean",
            "input_type": "checkbox"
        }
    ],
    "description": {
        "type": "string",
        "required": false,
        "description": "Optional; a human-readable description for documentation and tooling",
        "control_type": "textarea"
    },
    "primary_key": {
        "type": "boolean",
        "required": false,
        "description": "Optional; marks the dimension as a primary key for the cube, which affects joins and deduplication",
        "control_type": "toggle"
    },
    "granularities": {
        "type": "array",
        "required": false,
        "description": "For time dimensions, specifies available time granularities (e.g., year, month, day)",
        "lookup_type": "time_granularity",
        "control_type": "multiselect"
    }
}$cfg$::jsonb,
        '{}'::jsonb
    )
    ON CONFLICT (id) DO NOTHING;
END
$st$;
