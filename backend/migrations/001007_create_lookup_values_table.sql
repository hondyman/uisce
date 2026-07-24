-- Create lookup_values table for dynamic dropdown options
CREATE TABLE IF NOT EXISTS lookup_values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lookup_type VARCHAR(255) NOT NULL,
    value VARCHAR(255) NOT NULL,
    label VARCHAR(255) NOT NULL,
    description TEXT,
    sort_order INTEGER,
    metadata JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(lookup_type, value)
);

CREATE INDEX IF NOT EXISTS idx_lookup_values_type ON lookup_values(lookup_type);
CREATE INDEX IF NOT EXISTS idx_lookup_values_active ON lookup_values(is_active);

-- Seed dimension type values
INSERT INTO lookup_values (lookup_type, value, label, description, sort_order) VALUES
('dimension_type', 'string', 'String', 'Text or character data', 1),
('dimension_type', 'number', 'Number', 'Numeric data (integer or decimal)', 2),
('dimension_type', 'boolean', 'Boolean', 'True/false values', 3),
('dimension_type', 'time', 'Time', 'Date and time values', 4),
('dimension_type', 'geo', 'Geo', 'Geographic coordinates', 5)
ON CONFLICT (lookup_type, value) DO NOTHING;

-- Seed sort order values
INSERT INTO lookup_values (lookup_type, value, label, description, sort_order) VALUES
('sort_order', 'asc', 'Ascending', 'Sort from lowest to highest', 1),
('sort_order', 'desc', 'Descending', 'Sort from highest to lowest', 2)
ON CONFLICT (lookup_type, value) DO NOTHING;

-- Seed time granularity values
INSERT INTO lookup_values (lookup_type, value, label, description, sort_order) VALUES
('time_granularity', 'year', 'Year', 'Annual granularity', 1),
('time_granularity', 'quarter', 'Quarter', 'Quarterly granularity', 2),
('time_granularity', 'month', 'Month', 'Monthly granularity', 3),
('time_granularity', 'week', 'Week', 'Weekly granularity', 4),
('time_granularity', 'day', 'Day', 'Daily granularity', 5),
('time_granularity', 'hour', 'Hour', 'Hourly granularity', 6),
('time_granularity', 'minute', 'Minute', 'Minute granularity', 7),
('time_granularity', 'second', 'Second', 'Second granularity', 8)
ON CONFLICT (lookup_type, value) DO NOTHING;

-- Seed database data types (common types)
INSERT INTO lookup_values (lookup_type, value, label, description, sort_order) VALUES
('db_data_type', 'VARCHAR', 'VARCHAR', 'Variable-length character string', 1),
('db_data_type', 'INTEGER', 'INTEGER', 'Whole number', 2),
('db_data_type', 'BIGINT', 'BIGINT', 'Large whole number', 3),
('db_data_type', 'DECIMAL', 'DECIMAL', 'Fixed-point number', 4),
('db_data_type', 'FLOAT', 'FLOAT', 'Floating-point number', 5),
('db_data_type', 'BOOLEAN', 'BOOLEAN', 'True/false value', 6),
('db_data_type', 'DATE', 'DATE', 'Calendar date', 7),
('db_data_type', 'TIMESTAMP', 'TIMESTAMP', 'Date and time', 8),
('db_data_type', 'TEXT', 'TEXT', 'Long text', 9),
('db_data_type', 'JSON', 'JSON', 'JSON data', 10),
('db_data_type', 'JSONB', 'JSONB', 'Binary JSON data', 11)
ON CONFLICT (lookup_type, value) DO NOTHING;
