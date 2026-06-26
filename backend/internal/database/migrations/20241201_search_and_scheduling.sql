-- Migration: PostgreSQL-native Search and Scheduling
-- Replaces Elasticsearch with pg_tsvector + pg_trgm
-- Uses pg_cron for database-level scheduling

-- =============================================================================
-- Enable Required Extensions
-- =============================================================================

-- Full-text search is built-in, but we need pg_trgm for fuzzy matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- pg_cron for scheduling (must be enabled in Azure PostgreSQL Flexible Server)
-- In Azure Portal: Server Parameters -> shared_preload_libraries -> add 'pg_cron'
-- Then run: CREATE EXTENSION IF NOT EXISTS pg_cron;
-- Note: pg_cron can only be created in the 'postgres' database by default

-- For unaccented search (optional, good for international names)
CREATE EXTENSION IF NOT EXISTS unaccent;

-- =============================================================================
-- Search Configuration
-- =============================================================================

-- Custom text search configuration for semantic layer terminology
CREATE TEXT SEARCH CONFIGURATION IF NOT EXISTS semlayer_search (COPY = english);

-- Add synonym support for common terms
-- ALTER TEXT SEARCH CONFIGURATION semlayer_search
--   ALTER MAPPING FOR asciiword, asciihword, hword_asciipart
--   WITH unaccent, english_stem;

-- =============================================================================
-- Semantic Objects Search
-- =============================================================================

-- Add search vector column to semantic_objects
ALTER TABLE semantic_objects 
ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN index for fast full-text search
CREATE INDEX IF NOT EXISTS idx_semantic_objects_search 
ON semantic_objects USING GIN(search_vector);

-- Create trigram indexes for fuzzy/autocomplete search
CREATE INDEX IF NOT EXISTS idx_semantic_objects_name_trgm 
ON semantic_objects USING GIN(name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_semantic_objects_display_name_trgm 
ON semantic_objects USING GIN(display_name gin_trgm_ops);

-- Function to update search vector
CREATE OR REPLACE FUNCTION semantic_objects_search_vector_update() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector := 
    setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.display_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.object_type, '')), 'C') ||
    setweight(to_tsvector('english', COALESCE(
      (SELECT string_agg(tag, ' ') FROM unnest(NEW.tags) AS tag), ''
    )), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update search vector
DROP TRIGGER IF EXISTS semantic_objects_search_update ON semantic_objects;
CREATE TRIGGER semantic_objects_search_update
  BEFORE INSERT OR UPDATE ON semantic_objects
  FOR EACH ROW
  EXECUTE FUNCTION semantic_objects_search_vector_update();

-- Backfill existing records
UPDATE semantic_objects SET search_vector = 
  setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
  setweight(to_tsvector('english', COALESCE(display_name, '')), 'A') ||
  setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
  setweight(to_tsvector('english', COALESCE(object_type, '')), 'C');

-- =============================================================================
-- Bundles Search
-- =============================================================================

ALTER TABLE bundles 
ADD COLUMN IF NOT EXISTS search_vector tsvector;

CREATE INDEX IF NOT EXISTS idx_bundles_search 
ON bundles USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_bundles_name_trgm 
ON bundles USING GIN(name gin_trgm_ops);

CREATE OR REPLACE FUNCTION bundles_search_vector_update() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector := 
    setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.bundle_type, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS bundles_search_update ON bundles;
CREATE TRIGGER bundles_search_update
  BEFORE INSERT OR UPDATE ON bundles
  FOR EACH ROW
  EXECUTE FUNCTION bundles_search_vector_update();

UPDATE bundles SET search_vector = 
  setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
  setweight(to_tsvector('english', COALESCE(description, '')), 'B');

-- =============================================================================
-- Policies Search
-- =============================================================================

ALTER TABLE policies 
ADD COLUMN IF NOT EXISTS search_vector tsvector;

CREATE INDEX IF NOT EXISTS idx_policies_search 
ON policies USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_policies_name_trgm 
ON policies USING GIN(name gin_trgm_ops);

CREATE OR REPLACE FUNCTION policies_search_vector_update() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector := 
    setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.policy_type, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS policies_search_update ON policies;
CREATE TRIGGER policies_search_update
  BEFORE INSERT OR UPDATE ON policies
  FOR EACH ROW
  EXECUTE FUNCTION policies_search_vector_update();

-- =============================================================================
-- Catalog Search (Tables, Columns, etc.)
-- =============================================================================

ALTER TABLE catalog_tables 
ADD COLUMN IF NOT EXISTS search_vector tsvector;

CREATE INDEX IF NOT EXISTS idx_catalog_tables_search 
ON catalog_tables USING GIN(search_vector);

CREATE INDEX IF NOT EXISTS idx_catalog_tables_name_trgm 
ON catalog_tables USING GIN(table_name gin_trgm_ops);

CREATE OR REPLACE FUNCTION catalog_tables_search_vector_update() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector := 
    setweight(to_tsvector('english', COALESCE(NEW.table_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.schema_name, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.table_type, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS catalog_tables_search_update ON catalog_tables;
CREATE TRIGGER catalog_tables_search_update
  BEFORE INSERT OR UPDATE ON catalog_tables
  FOR EACH ROW
  EXECUTE FUNCTION catalog_tables_search_vector_update();

-- =============================================================================
-- Unified Search Function
-- =============================================================================

CREATE OR REPLACE FUNCTION search_all(
  p_query TEXT,
  p_tenant_id UUID,
  p_datasource_id UUID DEFAULT NULL,
  p_entity_types TEXT[] DEFAULT ARRAY['semantic_object', 'bundle', 'policy', 'table'],
  p_limit INT DEFAULT 50,
  p_offset INT DEFAULT 0
) RETURNS TABLE (
  entity_type TEXT,
  entity_id UUID,
  name TEXT,
  display_name TEXT,
  description TEXT,
  rank REAL,
  highlight TEXT
) AS $$
DECLARE
  v_tsquery tsquery;
  v_like_pattern TEXT;
BEGIN
  -- Parse search query for full-text search
  v_tsquery := websearch_to_tsquery('english', p_query);
  
  -- Pattern for trigram/LIKE search
  v_like_pattern := '%' || p_query || '%';
  
  RETURN QUERY
  WITH combined_results AS (
    -- Semantic Objects
    SELECT 
      'semantic_object'::TEXT as entity_type,
      so.id as entity_id,
      so.name,
      so.display_name,
      so.description,
      ts_rank(so.search_vector, v_tsquery) + 
        COALESCE(similarity(so.name, p_query), 0) * 0.5 as rank,
      ts_headline('english', COALESCE(so.description, so.name), v_tsquery,
        'StartSel=<mark>, StopSel=</mark>, MaxWords=50, MinWords=20') as highlight
    FROM semantic_objects so
    WHERE so.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR so.datasource_id = p_datasource_id)
      AND 'semantic_object' = ANY(p_entity_types)
      AND (
        so.search_vector @@ v_tsquery
        OR so.name ILIKE v_like_pattern
        OR so.display_name ILIKE v_like_pattern
        OR similarity(so.name, p_query) > 0.3
      )
    
    UNION ALL
    
    -- Bundles
    SELECT 
      'bundle'::TEXT,
      b.id,
      b.name,
      b.name as display_name,
      b.description,
      ts_rank(b.search_vector, v_tsquery) + 
        COALESCE(similarity(b.name, p_query), 0) * 0.5,
      ts_headline('english', COALESCE(b.description, b.name), v_tsquery,
        'StartSel=<mark>, StopSel=</mark>, MaxWords=50, MinWords=20')
    FROM bundles b
    WHERE b.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR b.datasource_id = p_datasource_id)
      AND 'bundle' = ANY(p_entity_types)
      AND (
        b.search_vector @@ v_tsquery
        OR b.name ILIKE v_like_pattern
        OR similarity(b.name, p_query) > 0.3
      )
    
    UNION ALL
    
    -- Policies
    SELECT 
      'policy'::TEXT,
      p.id,
      p.name,
      p.name as display_name,
      p.description,
      ts_rank(p.search_vector, v_tsquery) + 
        COALESCE(similarity(p.name, p_query), 0) * 0.5,
      ts_headline('english', COALESCE(p.description, p.name), v_tsquery,
        'StartSel=<mark>, StopSel=</mark>, MaxWords=50, MinWords=20')
    FROM policies p
    WHERE p.tenant_id = p_tenant_id
      AND 'policy' = ANY(p_entity_types)
      AND (
        p.search_vector @@ v_tsquery
        OR p.name ILIKE v_like_pattern
        OR similarity(p.name, p_query) > 0.3
      )
    
    UNION ALL
    
    -- Catalog Tables
    SELECT 
      'table'::TEXT,
      ct.id,
      ct.table_name,
      ct.table_name as display_name,
      ct.description,
      ts_rank(ct.search_vector, v_tsquery) + 
        COALESCE(similarity(ct.table_name, p_query), 0) * 0.5,
      ts_headline('english', COALESCE(ct.description, ct.table_name), v_tsquery,
        'StartSel=<mark>, StopSel=</mark>, MaxWords=50, MinWords=20')
    FROM catalog_tables ct
    WHERE ct.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR ct.datasource_id = p_datasource_id)
      AND 'table' = ANY(p_entity_types)
      AND (
        ct.search_vector @@ v_tsquery
        OR ct.table_name ILIKE v_like_pattern
        OR similarity(ct.table_name, p_query) > 0.3
      )
  )
  SELECT * FROM combined_results
  ORDER BY rank DESC
  LIMIT p_limit
  OFFSET p_offset;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- Autocomplete Function (Fast prefix matching)
-- =============================================================================

CREATE OR REPLACE FUNCTION autocomplete(
  p_prefix TEXT,
  p_tenant_id UUID,
  p_datasource_id UUID DEFAULT NULL,
  p_entity_types TEXT[] DEFAULT ARRAY['semantic_object', 'bundle', 'policy'],
  p_limit INT DEFAULT 10
) RETURNS TABLE (
  entity_type TEXT,
  entity_id UUID,
  name TEXT,
  display_name TEXT,
  similarity_score REAL
) AS $$
BEGIN
  RETURN QUERY
  WITH suggestions AS (
    -- Semantic Objects
    SELECT 
      'semantic_object'::TEXT as entity_type,
      so.id,
      so.name,
      so.display_name,
      similarity(so.name, p_prefix) as sim_score
    FROM semantic_objects so
    WHERE so.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR so.datasource_id = p_datasource_id)
      AND 'semantic_object' = ANY(p_entity_types)
      AND (
        so.name ILIKE p_prefix || '%'
        OR so.display_name ILIKE p_prefix || '%'
        OR similarity(so.name, p_prefix) > 0.2
      )
    
    UNION ALL
    
    -- Bundles
    SELECT 
      'bundle'::TEXT,
      b.id,
      b.name,
      b.name,
      similarity(b.name, p_prefix)
    FROM bundles b
    WHERE b.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR b.datasource_id = p_datasource_id)
      AND 'bundle' = ANY(p_entity_types)
      AND (
        b.name ILIKE p_prefix || '%'
        OR similarity(b.name, p_prefix) > 0.2
      )
    
    UNION ALL
    
    -- Policies
    SELECT 
      'policy'::TEXT,
      p.id,
      p.name,
      p.name,
      similarity(p.name, p_prefix)
    FROM policies p
    WHERE p.tenant_id = p_tenant_id
      AND 'policy' = ANY(p_entity_types)
      AND (
        p.name ILIKE p_prefix || '%'
        OR similarity(p.name, p_prefix) > 0.2
      )
  )
  SELECT s.entity_type, s.id, s.name, s.display_name, s.sim_score
  FROM suggestions s
  ORDER BY 
    CASE WHEN s.name ILIKE p_prefix || '%' THEN 0 ELSE 1 END,
    s.sim_score DESC,
    s.name
  LIMIT p_limit;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- Faceted Search Support
-- =============================================================================

CREATE OR REPLACE FUNCTION search_with_facets(
  p_query TEXT,
  p_tenant_id UUID,
  p_datasource_id UUID DEFAULT NULL,
  p_filters JSONB DEFAULT '{}'::JSONB
) RETURNS TABLE (
  results JSONB,
  facets JSONB,
  total_count BIGINT
) AS $$
DECLARE
  v_tsquery tsquery;
BEGIN
  v_tsquery := websearch_to_tsquery('english', p_query);
  
  RETURN QUERY
  WITH base_results AS (
    SELECT 
      so.id,
      so.name,
      so.display_name,
      so.description,
      so.object_type,
      so.datasource_id,
      ts_rank(so.search_vector, v_tsquery) as rank
    FROM semantic_objects so
    WHERE so.tenant_id = p_tenant_id
      AND (p_datasource_id IS NULL OR so.datasource_id = p_datasource_id)
      AND so.search_vector @@ v_tsquery
      -- Apply filters
      AND (
        p_filters->>'object_type' IS NULL 
        OR so.object_type = p_filters->>'object_type'
      )
  ),
  facet_counts AS (
    SELECT 
      jsonb_build_object(
        'object_types', (
          SELECT jsonb_agg(jsonb_build_object('value', object_type, 'count', cnt))
          FROM (
            SELECT object_type, COUNT(*) as cnt
            FROM base_results
            GROUP BY object_type
            ORDER BY cnt DESC
          ) t
        ),
        'datasources', (
          SELECT jsonb_agg(jsonb_build_object('value', datasource_id, 'count', cnt))
          FROM (
            SELECT datasource_id, COUNT(*) as cnt
            FROM base_results
            GROUP BY datasource_id
            ORDER BY cnt DESC
          ) t
        )
      ) as facets
  )
  SELECT 
    (SELECT jsonb_agg(row_to_json(br.*) ORDER BY br.rank DESC) FROM base_results br LIMIT 50),
    fc.facets,
    (SELECT COUNT(*) FROM base_results)
  FROM facet_counts fc;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- pg_cron Scheduled Jobs (Run in postgres database)
-- =============================================================================

-- Note: These must be created in the 'postgres' database where pg_cron is installed
-- Connect to postgres database first: \c postgres

-- Refresh search vector statistics daily at 2 AM UTC
-- SELECT cron.schedule('refresh-search-stats', '0 2 * * *', 
--   'ANALYZE semantic_objects; ANALYZE bundles; ANALYZE policies; ANALYZE catalog_tables;');

-- Clean up soft-deleted records older than 90 days (weekly on Sunday 3 AM)
-- SELECT cron.schedule('cleanup-deleted-records', '0 3 * * 0', $$
--   DELETE FROM semantic_objects WHERE deleted_at < NOW() - INTERVAL '90 days';
--   DELETE FROM bundles WHERE deleted_at < NOW() - INTERVAL '90 days';
--   DELETE FROM policies WHERE deleted_at < NOW() - INTERVAL '90 days';
-- $$);

-- Archive old audit logs monthly (1st of month at 4 AM)
-- SELECT cron.schedule('archive-audit-logs', '0 4 1 * *', $$
--   INSERT INTO audit_logs_archive 
--   SELECT * FROM audit_logs WHERE created_at < NOW() - INTERVAL '1 year';
--   DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '1 year';
-- $$);

-- Refresh materialized views for dashboards (every 15 minutes)
-- SELECT cron.schedule('refresh-dashboard-views', '*/15 * * * *',
--   'REFRESH MATERIALIZED VIEW CONCURRENTLY mv_tenant_usage_stats;');

-- Partition maintenance (create future partitions, drop old ones) - monthly
-- SELECT cron.schedule('partition-maintenance', '0 5 1 * *', $$
--   -- This would call a partition management function
--   SELECT maintain_partitions();
-- $$);

-- =============================================================================
-- Helper: List all pg_cron jobs
-- =============================================================================

CREATE OR REPLACE FUNCTION list_scheduled_jobs()
RETURNS TABLE (
  jobid BIGINT,
  schedule TEXT,
  command TEXT,
  nodename TEXT,
  nodeport INT,
  database TEXT,
  username TEXT,
  active BOOLEAN
) AS $$
BEGIN
  -- This only works if connected to the postgres database
  RETURN QUERY
  SELECT j.jobid, j.schedule, j.command, j.nodename, j.nodeport, 
         j.database, j.username, j.active
  FROM cron.job j
  ORDER BY j.jobid;
EXCEPTION
  WHEN undefined_table THEN
    -- pg_cron not installed or not in postgres database
    RAISE NOTICE 'pg_cron is not available. Connect to postgres database.';
    RETURN;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Performance Indexes
-- =============================================================================

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_semantic_objects_tenant_type 
ON semantic_objects(tenant_id, object_type);

CREATE INDEX IF NOT EXISTS idx_bundles_tenant_status 
ON bundles(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_policies_tenant_type 
ON policies(tenant_id, policy_type);

-- Partial indexes for active records
CREATE INDEX IF NOT EXISTS idx_semantic_objects_active_search 
ON semantic_objects USING GIN(search_vector) 
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_bundles_active_search 
ON bundles USING GIN(search_vector) 
WHERE deleted_at IS NULL;

-- =============================================================================
-- Search Analytics (Optional)
-- =============================================================================

CREATE TABLE IF NOT EXISTS search_analytics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  user_id UUID,
  query TEXT NOT NULL,
  result_count INT,
  clicked_result_id UUID,
  clicked_result_type TEXT,
  search_duration_ms INT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_analytics_tenant_date 
ON search_analytics(tenant_id, created_at DESC);

-- Function to log searches (call from application)
CREATE OR REPLACE FUNCTION log_search(
  p_tenant_id UUID,
  p_user_id UUID,
  p_query TEXT,
  p_result_count INT,
  p_duration_ms INT
) RETURNS UUID AS $$
DECLARE
  v_id UUID;
BEGIN
  INSERT INTO search_analytics (tenant_id, user_id, query, result_count, search_duration_ms)
  VALUES (p_tenant_id, p_user_id, p_query, p_result_count, p_duration_ms)
  RETURNING id INTO v_id;
  RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- Popular searches view
CREATE OR REPLACE VIEW v_popular_searches AS
SELECT 
  tenant_id,
  query,
  COUNT(*) as search_count,
  AVG(result_count) as avg_results,
  AVG(search_duration_ms) as avg_duration_ms
FROM search_analytics
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY tenant_id, query
HAVING COUNT(*) >= 5
ORDER BY search_count DESC;

COMMENT ON FUNCTION search_all IS 'Unified search across all entity types using PostgreSQL full-text search';
COMMENT ON FUNCTION autocomplete IS 'Fast prefix-based autocomplete using trigram similarity';
COMMENT ON FUNCTION search_with_facets IS 'Search with faceted navigation support';
