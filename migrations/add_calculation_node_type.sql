-- Add 'Calculation' to catalog_node_type if it doesn't exist
INSERT INTO catalog_node_type (id, name, description, icon)
VALUES (
    '550e8400-e29b-41d4-a716-446655440001', -- Fixed UUID for consistency
    'Calculation',
    'A financial or analytical calculation definition',
    'calculate'
) ON CONFLICT (name) DO NOTHING;

-- Create calculations table to store detailed definitions
CREATE TABLE IF NOT EXISTS calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID REFERENCES catalog_nodes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    formula TEXT NOT NULL,
    engine_type TEXT NOT NULL DEFAULT 'postgres', -- postgres, cube, python, excel
    return_type TEXT,
    arguments JSONB DEFAULT '{}'::jsonb, -- Parameters required by the calculation
    category TEXT,
    subcategory TEXT,
    is_materialized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraint to ensure unique name per tenant/datasource context (handled via node_id usually, but good to have)
    CONSTRAINT unique_calculation_name UNIQUE (name)
);

-- Add index for faster lookups
CREATE INDEX IF NOT EXISTS idx_calculations_node_id ON calculations(node_id);
CREATE INDEX IF NOT EXISTS idx_calculations_category ON calculations(category);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_calculations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_calculations_timestamp ON calculations;
CREATE TRIGGER update_calculations_timestamp
    BEFORE UPDATE ON calculations
    FOR EACH ROW
    EXECUTE FUNCTION update_calculations_updated_at();
