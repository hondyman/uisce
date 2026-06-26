-- Add is_active column to business_objects table
ALTER TABLE business_objects 
ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- Update existing records to be active by default
UPDATE business_objects SET is_active = true WHERE is_active IS NULL;
