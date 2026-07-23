-- ============================================================================
-- StarRocks Schema for Unified Calc Engine
-- Hot Tier: Native OLAP tables (< 90 days, real-time)
-- Cold Tier: External tables on Parquet/Iceberg (> 90 days, historical)
-- ============================================================================

-- =============================================================================
-- HOT DATABASE: Native StarRocks Tables
-- =============================================================================

CREATE DATABASE IF NOT EXISTS semantic_hot;
USE semantic_hot;

-- =============================================================================
-- CALC ENGINE AUDIT LOG - Full Compliance Tracking
-- =============================================================================
-- Records every calculation for audit, debugging, and performance analysis
-- Retention: 90 days in hot, then archived to cold
-- =============================================================================

CREATE TABLE IF NOT EXISTS calc_audit_log (
    -- Identity
    audit_id VARCHAR(64) NOT NULL COMMENT 'Unique audit record ID',
    request_id VARCHAR(64) NOT NULL COMMENT 'Request correlation ID',
    
    -- Tenant context
    tenant_id VARCHAR(64) NOT NULL COMMENT 'Tenant ID',
    datasource_id VARCHAR(64) NOT NULL COMMENT 'Datasource ID',
    user_id VARCHAR(64) COMMENT 'User who initiated the calculation',
    
    -- Calculation details
    calc_type VARCHAR(64) NOT NULL COMMENT 'Calculation type: NAV, XIRR, Returns, VaR, etc.',
    calc_id VARCHAR(64) COMMENT 'Calculation definition ID',
    metric_name VARCHAR(128) COMMENT 'Metric name if applicable',
    
    -- Input/Output
    input_params TEXT COMMENT 'JSON: Input parameters',
    output_value TEXT COMMENT 'JSON: Output value (or hash for large results)',
    output_hash VARCHAR(64) COMMENT 'SHA256 hash of output for large results',
    
    -- Execution details
    data_tier VARCHAR(16) COMMENT 'Data tier used: hot, cold, realtime',
    query_mode VARCHAR(32) COMMENT 'Query mode: realtime, hot, cold, union_safe',
    cache_hit BOOLEAN DEFAULT false COMMENT 'Whether result came from cache',
    
    -- Performance metrics
    start_time DATETIME NOT NULL COMMENT 'Calculation start time',
    end_time DATETIME COMMENT 'Calculation end time',
    duration_ms BIGINT COMMENT 'Duration in milliseconds',
    rows_scanned BIGINT DEFAULT 0 COMMENT 'Rows scanned during query',
    rows_returned BIGINT DEFAULT 0 COMMENT 'Rows returned',
    bytes_scanned BIGINT DEFAULT 0 COMMENT 'Bytes scanned during query',
    
    -- Status
    success BOOLEAN DEFAULT true COMMENT 'Whether calculation succeeded',
    error_message TEXT COMMENT 'Error message if failed',
    error_code VARCHAR(32) COMMENT 'Error code if failed',
    
    -- Query analysis (for slow queries)
    sql_query TEXT COMMENT 'SQL query executed (for debugging slow queries)',
    query_plan TEXT COMMENT 'Query execution plan (for optimization)',
    
    -- Source tracking
    source_ip VARCHAR(45) COMMENT 'Client IP address',
    user_agent VARCHAR(255) COMMENT 'Client user agent',
    api_endpoint VARCHAR(128) COMMENT 'API endpoint called',
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(audit_id)
PARTITION BY RANGE(start_time) (
    START ("2024-01-01") END ("2026-01-01") EVERY (INTERVAL 1 DAY)
)
DISTRIBUTED BY HASH(tenant_id, datasource_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "3",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "DAY",
    "dynamic_partition.start" = "-90",
    "dynamic_partition.end" = "3",
    "dynamic_partition.prefix" = "p"
)
COMMENT 'Full audit trail for calc engine - compliance and debugging';

-- Index for common queries
ALTER TABLE calc_audit_log SET ("bloom_filter_columns" = "tenant_id,datasource_id,calc_type,user_id");

-- =============================================================================
-- CALC ENGINE METRICS - Aggregated Performance Metrics
-- =============================================================================

CREATE TABLE IF NOT EXISTS calc_metrics_hourly (
    -- Dimensions
    hour_bucket DATETIME NOT NULL COMMENT 'Hour bucket for aggregation',
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    calc_type VARCHAR(64) NOT NULL,
    data_tier VARCHAR(16),
    
    -- Counters
    request_count BIGINT DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    error_count BIGINT DEFAULT 0,
    cache_hit_count BIGINT DEFAULT 0,
    
    -- Latency metrics (in milliseconds)
    latency_sum DOUBLE DEFAULT 0 COMMENT 'Sum of all latencies',
    latency_min DOUBLE COMMENT 'Minimum latency',
    latency_max DOUBLE COMMENT 'Maximum latency',
    latency_p50 DOUBLE COMMENT 'Median latency',
    latency_p95 DOUBLE COMMENT '95th percentile latency',
    latency_p99 DOUBLE COMMENT '99th percentile latency',
    
    -- Resource metrics
    rows_scanned_sum BIGINT DEFAULT 0,
    bytes_scanned_sum BIGINT DEFAULT 0,
    
    -- Outlier counts
    outlier_count BIGINT DEFAULT 0 COMMENT 'Calculations > 3 std devs from mean',
    slow_query_count BIGINT DEFAULT 0 COMMENT 'Calculations > 1s',
    
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (hour_bucket, tenant_id, datasource_id, calc_type, data_tier)
)
UNIQUE KEY(hour_bucket, tenant_id, datasource_id, calc_type, data_tier)
DISTRIBUTED BY HASH(tenant_id, datasource_id) BUCKETS 8
PROPERTIES (
    "replication_num" = "3"
)
COMMENT 'Hourly aggregated calc engine metrics for dashboards';

-- =============================================================================
-- CALC ENGINE ALERTS - Performance Alert History
-- =============================================================================

CREATE TABLE IF NOT EXISTS calc_alerts (
    alert_id VARCHAR(64) NOT NULL,
    alert_type VARCHAR(32) NOT NULL COMMENT 'latency_p95, latency_p99, error_rate, outlier',
    tenant_id VARCHAR(64),
    datasource_id VARCHAR(64),
    calc_type VARCHAR(64),
    
    severity VARCHAR(16) NOT NULL COMMENT 'warning, critical',
    message TEXT NOT NULL,
    value DOUBLE COMMENT 'Actual value that triggered alert',
    threshold DOUBLE COMMENT 'Threshold that was exceeded',
    
    metadata JSON COMMENT 'Additional context',
    
    created_at DATETIME NOT NULL,
    acked_at DATETIME COMMENT 'When alert was acknowledged',
    acked_by VARCHAR(64) COMMENT 'Who acknowledged',
    resolved_at DATETIME COMMENT 'When alert was resolved',
    resolved_by VARCHAR(64) COMMENT 'Who resolved',
    
    PRIMARY KEY (alert_id)
)
UNIQUE KEY(alert_id)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 4
PROPERTIES (
    "replication_num" = "3"
)
COMMENT 'Calc engine performance alert history';

-- =============================================================================
-- SLOW QUERY LOG - Detailed Analysis of Slow Queries
-- =============================================================================

CREATE TABLE IF NOT EXISTS calc_slow_queries (
    slow_query_id VARCHAR(64) NOT NULL,
    audit_id VARCHAR(64) NOT NULL COMMENT 'Reference to audit log',
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    calc_type VARCHAR(64) NOT NULL,
    
    duration_ms BIGINT NOT NULL,
    sql_query TEXT NOT NULL,
    query_plan TEXT,
    
    recommendations JSON COMMENT 'Auto-generated optimization recommendations',
    
    -- Analysis metadata
    tables_accessed JSON COMMENT 'Tables accessed by query',
    indexes_used JSON COMMENT 'Indexes used',
    full_scan_detected BOOLEAN DEFAULT false,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    analyzed_at DATETIME,
    
    PRIMARY KEY (slow_query_id)
)
UNIQUE KEY(slow_query_id)
DISTRIBUTED BY HASH(tenant_id, datasource_id) BUCKETS 4
PROPERTIES (
    "replication_num" = "3"
)
COMMENT 'Detailed analysis of slow calculations for optimization';

-- Holdings table (hot tier - real-time positions)
CREATE TABLE IF NOT EXISTS holdings (
CREATE TABLE IF NOT EXISTS holdings (
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    holding_id VARCHAR(64) NOT NULL,
    portfolio_id VARCHAR(64) NOT NULL,
    ticker VARCHAR(32) NOT NULL,
    security_name VARCHAR(255),
    quantity DECIMAL(20, 8) NOT NULL,
    cost_basis DECIMAL(20, 4),
    currency VARCHAR(3) DEFAULT 'USD',
    sector VARCHAR(64),
    asset_class VARCHAR(64),
    as_of_date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(tenant_id, datasource_id, holding_id)
PARTITION BY RANGE(as_of_date) (
    START ("2024-01-01") END ("2026-01-01") EVERY (INTERVAL 1 MONTH)
)
DISTRIBUTED BY HASH(tenant_id, portfolio_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "3",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "MONTH",
    "dynamic_partition.start" = "-3",
    "dynamic_partition.end" = "1",
    "dynamic_partition.prefix" = "p",
    "dynamic_partition.buckets" = "16"
);

-- Portfolio NAV table (hot tier - daily NAV values)
CREATE TABLE IF NOT EXISTS portfolio_nav (
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    portfolio_id VARCHAR(64) NOT NULL,
    as_of_date DATE NOT NULL,
    nav_value DECIMAL(20, 4) NOT NULL,
    cash_balance DECIMAL(20, 4) DEFAULT 0,
    total_market_value DECIMAL(20, 4),
    inflows DECIMAL(20, 4) DEFAULT 0,
    outflows DECIMAL(20, 4) DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(tenant_id, datasource_id, portfolio_id, as_of_date)
PARTITION BY RANGE(as_of_date) (
    START ("2024-01-01") END ("2026-01-01") EVERY (INTERVAL 1 MONTH)
)
DISTRIBUTED BY HASH(tenant_id, portfolio_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "3",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "MONTH",
    "dynamic_partition.start" = "-3",
    "dynamic_partition.end" = "1"
);

-- Prices table (hot tier - current and recent prices)
CREATE TABLE IF NOT EXISTS prices (
    ticker VARCHAR(32) NOT NULL,
    price_date DATE NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    open_price DECIMAL(20, 8),
    high_price DECIMAL(20, 8),
    low_price DECIMAL(20, 8),
    volume BIGINT,
    currency VARCHAR(3) DEFAULT 'USD',
    source VARCHAR(32),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(ticker, price_date)
PARTITION BY RANGE(price_date) (
    START ("2024-01-01") END ("2026-01-01") EVERY (INTERVAL 1 MONTH)
)
DISTRIBUTED BY HASH(ticker) BUCKETS 8
PROPERTIES (
    "replication_num" = "3",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "MONTH",
    "dynamic_partition.start" = "-3",
    "dynamic_partition.end" = "1"
);

-- FX rates table (hot tier)
CREATE TABLE IF NOT EXISTS fx_rates (
    from_currency VARCHAR(3) NOT NULL,
    to_currency VARCHAR(3) NOT NULL,
    rate_date DATE NOT NULL,
    rate DECIMAL(20, 10) NOT NULL,
    source VARCHAR(32),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(from_currency, to_currency, rate_date)
PARTITION BY RANGE(rate_date) (
    START ("2024-01-01") END ("2026-01-01") EVERY (INTERVAL 1 MONTH)
)
DISTRIBUTED BY HASH(from_currency, to_currency) BUCKETS 4
PROPERTIES (
    "replication_num" = "3"
);

-- Transactions table (hot tier)
CREATE TABLE IF NOT EXISTS transactions (
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    transaction_id VARCHAR(64) NOT NULL,
    portfolio_id VARCHAR(64) NOT NULL,
    transaction_date DATE NOT NULL,
    ticker VARCHAR(32),
    transaction_type VARCHAR(32) NOT NULL,
    quantity DECIMAL(20, 8),
    price DECIMAL(20, 8),
    currency VARCHAR(3) DEFAULT 'USD',
    fees DECIMAL(20, 4) DEFAULT 0,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(tenant_id, datasource_id, transaction_id)
PARTITION BY RANGE(transaction_date) (
    START ("2024-01-01") END ("2026-01-01") EVERY (INTERVAL 1 MONTH)
)
DISTRIBUTED BY HASH(tenant_id, portfolio_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "3",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "MONTH",
    "dynamic_partition.start" = "-3",
    "dynamic_partition.end" = "1"
);

-- Portfolio performance (pre-computed metrics)
CREATE TABLE IF NOT EXISTS portfolio_performance (
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    portfolio_id VARCHAR(64) NOT NULL,
    as_of_date DATE NOT NULL,
    return_1d DECIMAL(10, 6),
    return_1w DECIMAL(10, 6),
    return_1m DECIMAL(10, 6),
    return_3m DECIMAL(10, 6),
    return_ytd DECIMAL(10, 6),
    return_1y DECIMAL(10, 6),
    return_itd DECIMAL(10, 6),
    volatility_30d DECIMAL(10, 6),
    sharpe_ratio DECIMAL(10, 6),
    max_drawdown DECIMAL(10, 6),
    var_95 DECIMAL(20, 4),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
UNIQUE KEY(tenant_id, datasource_id, portfolio_id)
DISTRIBUTED BY HASH(tenant_id, portfolio_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "3"
);

-- Calculation definitions (registry for custom calculations)
CREATE TABLE IF NOT EXISTS calculation_definitions (
    id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    formula TEXT NOT NULL,
    input_params JSON,
    output_type VARCHAR(32) DEFAULT 'scalar',
    cacheable BOOLEAN DEFAULT true,
    cache_ttl_seconds INT DEFAULT 300,
    data_source VARCHAR(32) DEFAULT 'hot',
    tags JSON,
    active BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
UNIQUE KEY(id)
DISTRIBUTED BY HASH(id) BUCKETS 4
PROPERTIES (
    "replication_num" = "3"
);

-- =============================================================================
-- COLD DATABASE: External Tables on Parquet/S3
-- =============================================================================

CREATE DATABASE IF NOT EXISTS semantic_cold;
USE semantic_cold;

-- Create external catalog for S3/HDFS Parquet files
-- Note: Configure this based on your object storage setup

-- External holdings table (cold tier - historical)
CREATE EXTERNAL TABLE IF NOT EXISTS holdings (
    tenant_id VARCHAR(64),
    datasource_id VARCHAR(64),
    holding_id VARCHAR(64),
    portfolio_id VARCHAR(64),
    ticker VARCHAR(32),
    security_name VARCHAR(255),
    quantity DECIMAL(20, 8),
    cost_basis DECIMAL(20, 4),
    currency VARCHAR(3),
    sector VARCHAR(64),
    asset_class VARCHAR(64),
    as_of_date DATE,
    created_at DATETIME,
    updated_at DATETIME
)
ENGINE = file
PROPERTIES (
    "path" = "s3://your-bucket/semantic_cold/holdings/",
    "format" = "parquet",
    "aws.s3.access_key" = "${AWS_ACCESS_KEY}",
    "aws.s3.secret_key" = "${AWS_SECRET_KEY}",
    "aws.s3.region" = "us-east-1"
);

-- External portfolio_nav table (cold tier)
CREATE EXTERNAL TABLE IF NOT EXISTS portfolio_nav (
    tenant_id VARCHAR(64),
    datasource_id VARCHAR(64),
    portfolio_id VARCHAR(64),
    as_of_date DATE,
    nav_value DECIMAL(20, 4),
    cash_balance DECIMAL(20, 4),
    total_market_value DECIMAL(20, 4),
    inflows DECIMAL(20, 4),
    outflows DECIMAL(20, 4),
    created_at DATETIME
)
ENGINE = file
PROPERTIES (
    "path" = "s3://your-bucket/semantic_cold/portfolio_nav/",
    "format" = "parquet",
    "aws.s3.access_key" = "${AWS_ACCESS_KEY}",
    "aws.s3.secret_key" = "${AWS_SECRET_KEY}",
    "aws.s3.region" = "us-east-1"
);

-- External prices table (cold tier - historical prices)
CREATE EXTERNAL TABLE IF NOT EXISTS prices (
    ticker VARCHAR(32),
    price_date DATE,
    price DECIMAL(20, 8),
    open_price DECIMAL(20, 8),
    high_price DECIMAL(20, 8),
    low_price DECIMAL(20, 8),
    volume BIGINT,
    currency VARCHAR(3),
    source VARCHAR(32),
    created_at DATETIME
)
ENGINE = file
PROPERTIES (
    "path" = "s3://your-bucket/semantic_cold/prices/",
    "format" = "parquet",
    "aws.s3.access_key" = "${AWS_ACCESS_KEY}",
    "aws.s3.secret_key" = "${AWS_SECRET_KEY}",
    "aws.s3.region" = "us-east-1"
);

-- External transactions table (cold tier)
CREATE EXTERNAL TABLE IF NOT EXISTS transactions (
    tenant_id VARCHAR(64),
    datasource_id VARCHAR(64),
    transaction_id VARCHAR(64),
    portfolio_id VARCHAR(64),
    transaction_date DATE,
    ticker VARCHAR(32),
    transaction_type VARCHAR(32),
    quantity DECIMAL(20, 8),
    price DECIMAL(20, 8),
    currency VARCHAR(3),
    fees DECIMAL(20, 4),
    notes TEXT,
    created_at DATETIME
)
ENGINE = file
PROPERTIES (
    "path" = "s3://your-bucket/semantic_cold/transactions/",
    "format" = "parquet",
    "aws.s3.access_key" = "${AWS_ACCESS_KEY}",
    "aws.s3.secret_key" = "${AWS_SECRET_KEY}",
    "aws.s3.region" = "us-east-1"
);

-- =============================================================================
-- RESOURCE GROUPS FOR QOS
-- =============================================================================

-- Create resource groups for different workload types
USE semantic_hot;

-- =============================================================================
-- TIER WATERMARK TABLE - CRITICAL FOR DATA INTEGRITY
-- =============================================================================
-- This table is the SINGLE SOURCE OF TRUTH for hot/cold tier boundaries
-- All queries MUST respect these boundaries to prevent double-counting
-- =============================================================================

CREATE TABLE IF NOT EXISTS tier_watermarks (
    table_name VARCHAR(64) NOT NULL COMMENT 'Table being tracked (holdings, transactions, etc.)',
    tenant_id VARCHAR(64) NOT NULL COMMENT 'Tenant ID for isolation',
    datasource_id VARCHAR(64) NOT NULL COMMENT 'Datasource ID within tenant',
    
    -- THE AUTHORITATIVE BOUNDARY
    -- Hot tier: as_of_date >= cutoff_date
    -- Cold tier: as_of_date < cutoff_date
    cutoff_date DATE NOT NULL COMMENT 'Authoritative boundary date between hot and cold',
    
    -- Migration state machine
    -- STABLE: Normal operation, safe to query both tiers
    -- MIGRATING: Migration in progress, queries should use HOT ONLY
    -- VALIDATING: Post-migration validation, queries should use HOT ONLY
    state VARCHAR(20) DEFAULT 'STABLE' COMMENT 'STABLE|MIGRATING|VALIDATING',
    
    -- Migration tracking
    migration_started DATETIME COMMENT 'When current/last migration started',
    migration_ended DATETIME COMMENT 'When current/last migration completed',
    
    -- Validation checksums for integrity verification
    hot_row_count BIGINT DEFAULT 0 COMMENT 'Row count in hot tier after last validation',
    cold_row_count BIGINT DEFAULT 0 COMMENT 'Row count in cold tier after last validation',
    total_row_count BIGINT DEFAULT 0 COMMENT 'Total rows (hot + cold) - should never change',
    last_validated_at DATETIME COMMENT 'Last integrity validation timestamp',
    
    -- Metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (table_name, tenant_id, datasource_id)
)
UNIQUE KEY(table_name, tenant_id, datasource_id)
DISTRIBUTED BY HASH(tenant_id, datasource_id) BUCKETS 4
PROPERTIES (
    "replication_num" = "3"
)
COMMENT 'Single source of truth for hot/cold tier boundaries - prevents double counting';

-- Initialize default watermarks for core tables
INSERT INTO tier_watermarks (table_name, tenant_id, datasource_id, cutoff_date, state, total_row_count)
VALUES 
    ('holdings', 'default', 'default', DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY), 'STABLE', 0),
    ('portfolio_nav', 'default', 'default', DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY), 'STABLE', 0),
    ('transactions', 'default', 'default', DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY), 'STABLE', 0),
    ('prices', 'default', 'default', DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY), 'STABLE', 0)
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- =============================================================================
-- MIGRATION AUDIT LOG - Track every migration for compliance
-- =============================================================================

CREATE TABLE IF NOT EXISTS tier_migration_log (
    migration_id VARCHAR(64) NOT NULL,
    table_name VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    datasource_id VARCHAR(64) NOT NULL,
    
    old_cutoff_date DATE NOT NULL,
    new_cutoff_date DATE NOT NULL,
    
    rows_migrated BIGINT DEFAULT 0,
    rows_in_hot_before BIGINT DEFAULT 0,
    rows_in_hot_after BIGINT DEFAULT 0,
    rows_in_cold_before BIGINT DEFAULT 0,
    rows_in_cold_after BIGINT DEFAULT 0,
    
    validation_passed BOOLEAN DEFAULT false,
    validation_errors TEXT,
    
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    duration_seconds INT,
    
    initiated_by VARCHAR(64) COMMENT 'scheduler, manual, or user ID',
    
    PRIMARY KEY (migration_id)
)
UNIQUE KEY(migration_id)
DISTRIBUTED BY HASH(tenant_id, datasource_id) BUCKETS 4
PROPERTIES (
    "replication_num" = "3"
)
COMMENT 'Audit trail for all tier migrations - required for compliance';

-- High priority for real-time app queries
CREATE RESOURCE GROUP IF NOT EXISTS realtime_high
TO (user='app_user')
WITH (
    'cpu_core_limit' = '16',
    'mem_limit' = '80%',
    'concurrency_limit' = '100',
    'type' = 'normal'
);

-- Normal priority for analytics/BI queries
CREATE RESOURCE GROUP IF NOT EXISTS analytics_normal
TO (user='analytics_user')
WITH (
    'cpu_core_limit' = '8',
    'mem_limit' = '40%',
    'concurrency_limit' = '50',
    'type' = 'normal'
);

-- Low priority for batch/ETL jobs
CREATE RESOURCE GROUP IF NOT EXISTS batch_low
TO (user='batch_user')
WITH (
    'cpu_core_limit' = '4',
    'mem_limit' = '20%',
    'concurrency_limit' = '10',
    'type' = 'normal'
);

-- =============================================================================
-- MATERIALIZED VIEWS FOR COMMON AGGREGATIONS
-- =============================================================================

-- Pre-aggregated daily portfolio metrics
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_portfolio_daily_metrics
DISTRIBUTED BY HASH(tenant_id, portfolio_id) BUCKETS 16
REFRESH ASYNC START('00:05:00') EVERY (INTERVAL 1 HOUR)
AS
SELECT
    tenant_id,
    datasource_id,
    portfolio_id,
    as_of_date,
    SUM(quantity * p.price) as total_market_value,
    COUNT(DISTINCT ticker) as position_count,
    SUM(CASE WHEN asset_class = 'EQUITY' THEN quantity * p.price ELSE 0 END) as equity_value,
    SUM(CASE WHEN asset_class = 'FIXED_INCOME' THEN quantity * p.price ELSE 0 END) as fixed_income_value,
    SUM(CASE WHEN asset_class = 'CASH' THEN quantity * p.price ELSE 0 END) as cash_value
FROM holdings h
LEFT JOIN prices p ON h.ticker = p.ticker AND p.price_date = h.as_of_date
GROUP BY tenant_id, datasource_id, portfolio_id, as_of_date;

-- Pre-aggregated sector allocation
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_sector_allocation
DISTRIBUTED BY HASH(tenant_id, portfolio_id) BUCKETS 16
REFRESH ASYNC START('00:10:00') EVERY (INTERVAL 1 HOUR)
AS
SELECT
    tenant_id,
    datasource_id,
    portfolio_id,
    as_of_date,
    sector,
    SUM(quantity * p.price) as sector_value,
    COUNT(*) as position_count
FROM holdings h
LEFT JOIN prices p ON h.ticker = p.ticker AND p.price_date = h.as_of_date
WHERE sector IS NOT NULL
GROUP BY tenant_id, datasource_id, portfolio_id, as_of_date, sector;

-- =============================================================================
-- INDEXES FOR COMMON QUERY PATTERNS
-- =============================================================================

-- Bloom filter indexes for tenant isolation
ALTER TABLE holdings SET ("bloom_filter_columns" = "tenant_id,datasource_id,portfolio_id");
ALTER TABLE portfolio_nav SET ("bloom_filter_columns" = "tenant_id,datasource_id,portfolio_id");
ALTER TABLE transactions SET ("bloom_filter_columns" = "tenant_id,datasource_id,portfolio_id");

-- =============================================================================
-- SAMPLE CALCULATION DEFINITIONS
-- =============================================================================

INSERT INTO calculation_definitions (id, name, description, formula, input_params, output_type, tags)
VALUES
-- NAV Calculation
('nav_total', 'Total NAV', 'Calculate total Net Asset Value for a portfolio',
 'SELECT SUM(h.quantity * p.price * COALESCE(fx.rate, 1.0)) as nav_value
  FROM {{database}}.holdings h
  LEFT JOIN {{database}}.prices p ON h.ticker = p.ticker AND p.price_date = ''{{as_of_date}}''
  LEFT JOIN {{database}}.fx_rates fx ON h.currency = fx.from_currency AND fx.to_currency = ''USD'' AND fx.rate_date = ''{{as_of_date}}''
  WHERE h.tenant_id = ''{{tenant_id}}'' AND h.datasource_id = ''{{datasource_id}}'' AND h.portfolio_id = ''{{portfolio_id}}''',
 '[{"name": "portfolio_id", "type": "string", "required": true}, {"name": "as_of_date", "type": "date", "required": false}]',
 'scalar',
 '{"category": "valuation", "frequency": "realtime"}'),

-- Daily Return
('return_daily', 'Daily Return', 'Calculate daily return for a portfolio',
 'SELECT (current.nav_value - previous.nav_value) / previous.nav_value as daily_return
  FROM {{database}}.portfolio_nav current
  JOIN {{database}}.portfolio_nav previous ON current.portfolio_id = previous.portfolio_id
    AND previous.as_of_date = DATE_SUB(current.as_of_date, INTERVAL 1 DAY)
  WHERE current.tenant_id = ''{{tenant_id}}'' AND current.datasource_id = ''{{datasource_id}}''
    AND current.portfolio_id = ''{{portfolio_id}}'' AND current.as_of_date = ''{{as_of_date}}''',
 '[{"name": "portfolio_id", "type": "string", "required": true}, {"name": "as_of_date", "type": "date", "required": true}]',
 'scalar',
 '{"category": "performance", "frequency": "daily"}'),

-- Position Count
('position_count', 'Position Count', 'Count distinct positions in portfolio',
 'SELECT COUNT(DISTINCT ticker) as position_count
  FROM {{database}}.holdings
  WHERE tenant_id = ''{{tenant_id}}'' AND datasource_id = ''{{datasource_id}}'' AND portfolio_id = ''{{portfolio_id}}''',
 '[{"name": "portfolio_id", "type": "string", "required": true}]',
 'scalar',
 '{"category": "portfolio", "frequency": "realtime"}');
