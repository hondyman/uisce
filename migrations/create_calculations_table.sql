CREATE TABLE IF NOT EXISTS calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id UUID, -- Link to catalog_node if applicable
    name VARCHAR(255) NOT NULL UNIQUE,
    title VARCHAR(255),
    description TEXT,
    formula TEXT,
    engine_type VARCHAR(50), -- postgres, cube, python, excel
    return_type VARCHAR(50),
    arguments JSONB DEFAULT '[]',
    category VARCHAR(100),
    subcategory VARCHAR(100),
    domain_id UUID, -- Link to data_domain
    execution_type VARCHAR(50) DEFAULT 'realtime', -- realtime, batch
    engine VARCHAR(50) DEFAULT 'internal', -- internal, cube, spark
    is_materialized BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_calculations_domain_id ON calculations(domain_id);
