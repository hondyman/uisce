-- Add new columns to calculations table
ALTER TABLE calculations
ADD COLUMN domain_id UUID REFERENCES data_domain(id),
ADD COLUMN execution_type VARCHAR(50) DEFAULT 'realtime',
ADD COLUMN engine VARCHAR(50) DEFAULT 'internal';

-- Add index for domain_id for faster lookups
CREATE INDEX idx_calculations_domain_id ON calculations(domain_id);
