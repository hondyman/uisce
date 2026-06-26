-- Add missing columns to catalog_validation_rules if they don't exist
ALTER TABLE IF EXISTS catalog_validation_rules 
ADD COLUMN IF NOT EXISTS script_content TEXT,
ADD COLUMN IF NOT EXISTS target_entities TEXT[],
ADD COLUMN IF NOT EXISTS created_by TEXT;
