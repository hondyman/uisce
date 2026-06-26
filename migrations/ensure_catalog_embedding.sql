-- Add embedding column to catalog_node if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'catalog_node' AND column_name = 'embedding') THEN
        ALTER TABLE catalog_node ADD COLUMN embedding vector(1536);
    END IF;
END $$;
