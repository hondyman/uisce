-- Migration to align business_objects with Workday model (Driving Primary Table)

ALTER TABLE business_objects 
ADD COLUMN IF NOT EXISTS driver_table_id uuid DEFAULT NULL,
ADD COLUMN IF NOT EXISTS driver_table_name text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS technical_name text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS parent_id uuid DEFAULT NULL REFERENCES business_objects(id),
ADD COLUMN IF NOT EXISTS created_by text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS last_modified_by text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS clones_from text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS clone_parent_key text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS clone_parent_display_name text DEFAULT NULL,
ADD COLUMN IF NOT EXISTS category text DEFAULT NULL;

-- Create index for driver table lookups if it doesn't exist
CREATE INDEX IF NOT EXISTS idx_bo_driver_table ON business_objects(driver_table_id);
CREATE INDEX IF NOT EXISTS idx_bo_parent_id ON business_objects(parent_id);
