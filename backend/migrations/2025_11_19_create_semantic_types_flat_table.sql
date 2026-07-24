-- Create semantic_types flat table (like data_domains)
CREATE TABLE IF NOT EXISTS semantic_types (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    semantic_type text NOT NULL,
    data_type text NOT NULL,
    format text NOT NULL,
    notes text,
    tenant_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_semantic_types_tenant_id ON semantic_types(tenant_id);
CREATE INDEX IF NOT EXISTS idx_semantic_types_lookup ON semantic_types(semantic_type, data_type, format);

-- Add unique constraint to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS ux_semantic_types_combo ON semantic_types(semantic_type, data_type, format, COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- Populate the semantic_types table
INSERT INTO semantic_types (semantic_type, data_type, format, notes, tenant_id) VALUES
-- Dimension types
('Dimension', 'string', 'default', '', NULL),
('Dimension', 'string', 'imageUrl', 'Dimension Format', NULL),
('Dimension', 'string', 'link', 'Dimension Format', NULL),
('Dimension', 'string', 'currency', 'Dimension Format (If underlying type is number and formatted as string in SQL)', NULL),
('Dimension', 'string', 'percent', 'Dimension Format (If underlying type is number and formatted as string in SQL)', NULL),
('Dimension', 'number', 'default', '', NULL),
('Dimension', 'number', 'id', 'Dimension Format', NULL),
('Dimension', 'number', 'currency', 'Dimension Format', NULL),
('Dimension', 'number', 'percent', 'Dimension Format', NULL),
('Dimension', 'boolean', 'default', '', NULL),
('Dimension', 'time', 'default', '', NULL),
('Dimension', 'geo', 'default', '', NULL),
-- Measure types
('Measure', 'string', 'default', 'Measure Type', NULL),
('Measure', 'time', 'default', 'Measure Type', NULL),
('Measure', 'boolean', 'default', 'Measure Type', NULL),
('Measure', 'number', 'default', 'Measure Type', NULL),
('Measure', 'number', 'percent', 'Measure Format', NULL),
('Measure', 'number', 'currency', 'Measure Format', NULL),
('Measure', 'number_agg', 'default', 'Measure Type', NULL),
('Measure', 'number_agg', 'percent', 'Measure Format', NULL),
('Measure', 'number_agg', 'currency', 'Measure Format', NULL),
('Measure', 'count', 'default', 'Measure Type', NULL),
('Measure', 'count_distinct', 'default', 'Measure Type', NULL),
('Measure', 'count_distinct_approx', 'default', 'Measure Type', NULL),
('Measure', 'sum', 'default', 'Measure Type', NULL),
('Measure', 'sum', 'currency', 'Measure Format', NULL),
('Measure', 'avg', 'default', 'Measure Type', NULL),
('Measure', 'min', 'default', 'Measure Type', NULL),
('Measure', 'max', 'default', 'Measure Type', NULL),
-- Time type
('Time', 'time', 'default', 'Dedicated Semantic Time Object', NULL)
ON CONFLICT DO NOTHING;
