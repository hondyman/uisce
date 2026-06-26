-- Add region and related columns to tenants table
ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS region VARCHAR(50) DEFAULT 'us-west';

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS allowed_regions JSONB DEFAULT '["us-west"]'::jsonb;

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS description TEXT;

ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS gold_copy BOOLEAN DEFAULT false;
