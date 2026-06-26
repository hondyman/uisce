-- StarRocks Resource Management Schema

-- Calc Engine Database (ephemeral, real-time calculations)
CREATE DATABASE IF NOT EXISTS calc_engine;

CREATE TABLE IF NOT EXISTS calc_engine.calc_results (
  calc_id STRING NOT NULL COMMENT 'Unique calculation identifier',
  tenant_id STRING NOT NULL COMMENT 'Tenant identifier',
  user_id STRING COMMENT 'User who requested calculation',
  metric_name STRING NOT NULL COMMENT 'Name of calculated metric',
  metric_value DOUBLE NOT NULL COMMENT 'Calculated value',
  timestamp DATETIME NOT NULL COMMENT 'Calculation timestamp',
  context JSON COMMENT 'Additional metadata',
  PRIMARY KEY (calc_id, tenant_id)
)
DUPLICATE KEY(calc_id, tenant_id)
DISTRIBUTED BY HASH(calc_id) BUCKETS 32
PROPERTIES (
  "replication_num" = "3",
  "storage_medium" = "SSD",
  "compression" = "LZ4"
)
COMMENT 'Ephemeral calculation results with short retention';

-- Semantic Layer Database (governed, pre-aggregated rollups)
CREATE DATABASE IF NOT EXISTS semantic_layer;

CREATE TABLE IF NOT EXISTS semantic_layer.rollups (
  rollup_id STRING NOT NULL COMMENT 'Rollup identifier',
  tenant_id STRING NOT NULL COMMENT 'Tenant identifier',
  cube_name STRING NOT NULL COMMENT 'Cube name',
  date DATE NOT NULL COMMENT 'Partition date',
  freshness_minutes INT COMMENT 'Minutes since last refresh',
  status STRING COMMENT 'healthy|stale|failing',
  last_build DATETIME COMMENT 'Last successful build timestamp',
  p50_latency DOUBLE COMMENT 'Median query latency (ms)',
  p95_latency DOUBLE COMMENT 'p95 query latency (ms)',
  cache_hit_ratio DOUBLE COMMENT 'Cache hit ratio (0-1)',
  error_count INT COMMENT 'Number of errors',
  PRIMARY KEY (rollup_id, tenant_id, date)
)
AGGREGATE KEY(rollup_id, tenant_id, date)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 64
PROPERTIES (
  "replication_num" = "3",
  "storage_medium" = "SSD",
  "compression" = "LZ4"
)
COMMENT 'Governed pre-aggregation metadata and metrics';

CREATE TABLE IF NOT EXISTS semantic_layer.audit_log (
  id BIGINT AUTO_INCREMENT COMMENT 'Auto-incrementing ID',
  actor STRING NOT NULL COMMENT 'User or service performing action',
  action STRING NOT NULL COMMENT 'Action type',
  scope STRING NOT NULL COMMENT 'Resource scope (tenant, rollup, etc)',
  timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Event timestamp',
  result STRING NOT NULL COMMENT 'success|failure with details',
  PRIMARY KEY (id)
)
DUPLICATE KEY(id)
DISTRIBUTED BY HASH(id) BUCKETS 16
PROPERTIES (
  "replication_num" = "3",
  "storage_medium" = "SSD"
)
COMMENT 'Audit trail for all admin actions';

-- Resource Groups for workload isolation
CREATE RESOURCE GROUP IF NOT EXISTS semantic_rollups
TO ('semantic_layer')
WITH (
  'cpu_share' = '50',              -- 50% guaranteed CPU
  'mem_limit' = '60%',             -- Max 60% cluster memory
  'concurrency_limit' = '50'       -- Max 50 concurrent queries
)
COMMENT 'Resource group for semantic layer rollup queries and refreshes';

CREATE RESOURCE GROUP IF NOT EXISTS calc_engine
TO ('calc_engine')
WITH (
  'cpu_share' = '30',              -- 30% CPU cap
  'mem_limit' = '30%',             -- Max 30% cluster memory
  'concurrency_limit' = '100'      -- Allow bursts but capped
)
COMMENT 'Resource group for real-time calculation engine';

CREATE RESOURCE GROUP IF NOT EXISTS default_group
TO ('default')
WITH (
  'cpu_share' = '20',              -- 20% CPU for background
  'mem_limit' = '10%',             -- Minimal memory
  'concurrency_limit' = '20'
)
COMMENT 'Default resource group for system queries';
