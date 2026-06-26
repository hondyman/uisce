-- StarRocks Pre-Aggregation Database Setup for Cube.js
-- This database stores materialized pre-aggregations instead of Cube Store
-- Provides HA and better performance for the semantic layer

-- Create database for Cube.js pre-aggregations
CREATE DATABASE IF NOT EXISTS cube_preaggs;

USE cube_preaggs;

-- Grant permissions for Cube.js refresh workers
-- In production, create a dedicated user with limited permissions
GRANT ALL PRIVILEGES ON cube_preaggs.* TO 'root'@'%';

-- Create metadata table for pre-aggregation tracking
CREATE TABLE IF NOT EXISTS preagg_metadata (
  preagg_id VARCHAR(255) PRIMARY KEY,
  cube_name VARCHAR(255) NOT NULL,
  tenant_id VARCHAR(255) NOT NULL,
  datasource_id VARCHAR(255) NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_refresh DATETIME,
  refresh_status VARCHAR(50),
  row_count BIGINT,
  storage_bytes BIGINT,
  INDEX idx_tenant (tenant_id),
  INDEX idx_cube (cube_name),
  INDEX idx_refresh (last_refresh)
) ENGINE=OLAP
DUPLICATE KEY(preagg_id)
DISTRIBUTED BY HASH(preagg_id) BUCKETS 10;

-- Create resource groups for tenant workload isolation
-- Prevents noisy neighbor problems

CREATE RESOURCE GROUP IF NOT EXISTS tenant_premium
WITH (
  cpu_weight = 10,
  mem_limit = '40%',
  concurrency_limit = 20,
  type = 'normal'
);

CREATE RESOURCE GROUP IF NOT EXISTS tenant_standard
WITH (
  cpu_weight = 5,
  mem_limit = '30%',
  concurrency_limit = 10,
  type = 'normal'
);

CREATE RESOURCE GROUP IF NOT EXISTS tenant_basic
WITH (
  cpu_weight = 2,
  mem_limit = '20%',
  concurrency_limit = 5,
  type = 'normal'
);

-- Create monitoring view for pre-aggregation health
CREATE VIEW IF NOT EXISTS v_preagg_health AS
SELECT 
  cube_name,
  tenant_id,
  COUNT(*) as preagg_count,
  SUM(row_count) as total_rows,
  SUM(storage_bytes) / (1024*1024*1024) as storage_gb,
  MAX(last_refresh) as latest_refresh,
  MIN(last_refresh) as oldest_refresh,
  SUM(CASE WHEN refresh_status = 'error' THEN 1 ELSE 0 END) as error_count
FROM preagg_metadata
GROUP BY cube_name, tenant_id;

-- Example: Query pre-aggregation health
-- SELECT * FROM v_preagg_health WHERE tenant_id = 'your-tenant-id';

SHOW DATABASES;
