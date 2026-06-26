-- Create semantic_model_calculations table to link calculations to semantic models
CREATE TABLE IF NOT EXISTS semantic_model_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semantic_model_id UUID NOT NULL, -- Link to semantic_models table (assuming it exists, otherwise just UUID)
    calculation_id UUID NOT NULL REFERENCES calculations(id),
    argument_mapping JSONB NOT NULL DEFAULT '{}', -- Map of calc argument name to model column/measure name
    output_name VARCHAR(255) NOT NULL, -- Name of the resulting measure/dimension
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID,
    updated_by UUID
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_semantic_model_calculations_model_id ON semantic_model_calculations(semantic_model_id);
CREATE INDEX IF NOT EXISTS idx_semantic_model_calculations_calc_id ON semantic_model_calculations(calculation_id);
