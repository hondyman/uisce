-- Uisce Conversational Discovery Full-Text Search Indices
-- Rule 2 (Graph-First) & Rule 7 (Security Mandate)

BEGIN;

-- 1. Add Search Vector Column to catalog_node
ALTER TABLE catalog_node 
ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;

-- 2. Populate Search Vector (A = Node Key, B = Node Name, C = Description/Type)
UPDATE catalog_node
SET search_vector = 
    setweight(to_tsvector('english', COALESCE(node_key, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(node_name, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(node_type, '')), 'C');

-- 3. GIN Index for High-Speed Full-Text Search
CREATE INDEX IF NOT EXISTS idx_catalog_node_search_vector ON catalog_node USING GIN(search_vector);

-- 4. Add Search Vector to page_registry for Dynamic UI Discovery
ALTER TABLE platform.page_registry 
ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;

UPDATE platform.page_registry
SET search_vector = 
    setweight(to_tsvector('english', COALESCE(page_key, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(page_name, '')), 'B');

CREATE INDEX IF NOT EXISTS idx_page_registry_search_vector ON platform.page_registry USING GIN(search_vector);

COMMIT;
