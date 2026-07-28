-- Extend catalog_node to support derived expressions, formatting, descriptions, and entity default filters
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS expression TEXT;
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS format VARCHAR(50);
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE catalog_node ADD COLUMN IF NOT EXISTS default_filters JSONB DEFAULT '[]'::jsonb;

-- Extend catalog_edge to support role-playing dimensions (e.g. order_date vs ship_date)
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS role_name VARCHAR(100);
