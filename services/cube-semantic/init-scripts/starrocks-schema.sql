-- StarRocks Schema for Hot/Cold Analytics Architecture
-- 
-- HOT STORE: Native StarRocks tables for real-time queries
-- COLD STORE: External tables on Parquet files for historical data
-- 
-- Cube.js pre-aggregations are stored in cube_preagg database

-- =============================================================================
-- DATABASE SETUP
-- =============================================================================

-- Hot store for real-time data
CREATE DATABASE IF NOT EXISTS cube_hot;

-- Pre-aggregation storage (Cube.js writes here)
CREATE DATABASE IF NOT EXISTS cube_preagg;

-- Cold store references (external tables pointing to Parquet)
CREATE DATABASE IF NOT EXISTS cube_cold;

-- =============================================================================
-- HOT STORE TABLES (Native StarRocks - fast queries)
-- =============================================================================

USE cube_hot;

-- Transactions hot table (last 90 days)
CREATE TABLE IF NOT EXISTS transactions (
    transaction_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    transaction_date DATE NOT NULL,
    transaction_type VARCHAR(32),
    amount DECIMAL(18, 4),
    currency VARCHAR(3),
    category VARCHAR(64),
    merchant VARCHAR(128),
    status VARCHAR(16),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(transaction_id, tenant_id)
PARTITION BY RANGE(transaction_date) (
    PARTITION p_current VALUES LESS THAN (CURRENT_DATE()),
    PARTITION p_future VALUES LESS THAN MAXVALUE
)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "1",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "DAY",
    "dynamic_partition.start" = "-90",
    "dynamic_partition.end" = "3",
    "dynamic_partition.prefix" = "p",
    "dynamic_partition.buckets" = "16"
);

-- Positions hot table (current state)
CREATE TABLE IF NOT EXISTS positions (
    position_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    portfolio_id VARCHAR(64) NOT NULL,
    security_id VARCHAR(64) NOT NULL,
    quantity DECIMAL(18, 8),
    market_value DECIMAL(18, 4),
    cost_basis DECIMAL(18, 4),
    unrealized_pnl DECIMAL(18, 4),
    currency VARCHAR(3),
    as_of_date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(position_id, tenant_id)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "1"
);

-- Portfolio metrics hot table (aggregated daily)
CREATE TABLE IF NOT EXISTS portfolio_metrics (
    metric_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    portfolio_id VARCHAR(64) NOT NULL,
    metric_date DATE NOT NULL,
    total_value DECIMAL(18, 4),
    daily_return DECIMAL(12, 6),
    mtd_return DECIMAL(12, 6),
    ytd_return DECIMAL(12, 6),
    sharpe_ratio DECIMAL(8, 4),
    volatility DECIMAL(8, 4),
    var_95 DECIMAL(18, 4),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DUPLICATE KEY(metric_id, tenant_id)
PARTITION BY RANGE(metric_date) (
    PARTITION p_current VALUES LESS THAN (CURRENT_DATE()),
    PARTITION p_future VALUES LESS THAN MAXVALUE
)
DISTRIBUTED BY HASH(tenant_id) BUCKETS 8
PROPERTIES (
    "replication_num" = "1",
    "dynamic_partition.enable" = "true",
    "dynamic_partition.time_unit" = "MONTH",
    "dynamic_partition.start" = "-12",
    "dynamic_partition.end" = "1",
    "dynamic_partition.prefix" = "p",
    "dynamic_partition.buckets" = "8"
);

-- =============================================================================
-- COLD STORE TABLES (External tables on Parquet)
-- =============================================================================

USE cube_cold;

-- Historical transactions (Parquet on S3/Azure Blob/local)
-- Files partitioned by: /data/parquet/transactions/tenant_id=XXX/year=YYYY/month=MM/
CREATE EXTERNAL TABLE IF NOT EXISTS transactions_history (
    transaction_id VARCHAR(64),
    tenant_id VARCHAR(64),
    account_id VARCHAR(64),
    transaction_date DATE,
    transaction_type VARCHAR(32),
    amount DECIMAL(18, 4),
    currency VARCHAR(3),
    category VARCHAR(64),
    merchant VARCHAR(128),
    status VARCHAR(16),
    created_at DATETIME
)
ENGINE = file
PROPERTIES (
    "path" = "/data/parquet/transactions/",
    "format" = "parquet",
    "enable_recursive_listing" = "true"
);

-- Historical positions (Parquet snapshots)
CREATE EXTERNAL TABLE IF NOT EXISTS positions_history (
    position_id VARCHAR(64),
    tenant_id VARCHAR(64),
    portfolio_id VARCHAR(64),
    security_id VARCHAR(64),
    quantity DECIMAL(18, 8),
    market_value DECIMAL(18, 4),
    cost_basis DECIMAL(18, 4),
    unrealized_pnl DECIMAL(18, 4),
    currency VARCHAR(3),
    as_of_date DATE,
    snapshot_date DATE
)
ENGINE = file
PROPERTIES (
    "path" = "/data/parquet/positions/",
    "format" = "parquet",
    "enable_recursive_listing" = "true"
);

-- Historical portfolio metrics
CREATE EXTERNAL TABLE IF NOT EXISTS portfolio_metrics_history (
    metric_id VARCHAR(64),
    tenant_id VARCHAR(64),
    portfolio_id VARCHAR(64),
    metric_date DATE,
    total_value DECIMAL(18, 4),
    daily_return DECIMAL(12, 6),
    mtd_return DECIMAL(12, 6),
    ytd_return DECIMAL(12, 6),
    sharpe_ratio DECIMAL(8, 4),
    volatility DECIMAL(8, 4),
    var_95 DECIMAL(18, 4)
)
ENGINE = file
PROPERTIES (
    "path" = "/data/parquet/portfolio_metrics/",
    "format" = "parquet",
    "enable_recursive_listing" = "true"
);

-- =============================================================================
-- UNIFIED VIEWS (Hot + Cold)
-- =============================================================================

USE cube_hot;

-- Unified transactions view (hot + cold)
CREATE VIEW IF NOT EXISTS transactions_all AS
SELECT 
    transaction_id,
    tenant_id,
    account_id,
    transaction_date,
    transaction_type,
    amount,
    currency,
    category,
    merchant,
    status,
    created_at,
    'hot' AS data_tier
FROM cube_hot.transactions
UNION ALL
SELECT 
    transaction_id,
    tenant_id,
    account_id,
    transaction_date,
    transaction_type,
    amount,
    currency,
    category,
    merchant,
    status,
    created_at,
    'cold' AS data_tier
FROM cube_cold.transactions_history
WHERE transaction_date < DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY);

-- =============================================================================
-- PRE-AGGREGATION TABLES (Cube.js uses these)
-- =============================================================================

USE cube_preagg;

-- Cube.js will create tables here automatically based on schema definitions
-- Example of what Cube.js creates:
--
-- CREATE TABLE IF NOT EXISTS transactions_by_tenant_daily (
--     tenant_id VARCHAR(64),
--     transaction_date DATE,
--     transaction_type VARCHAR(32),
--     transaction_count BIGINT,
--     total_amount DECIMAL(18, 4),
--     avg_amount DECIMAL(18, 4)
-- ) ...

-- Grant Cube.js permission to create/modify tables
-- In production, use a dedicated cube_preagg user
GRANT ALL ON cube_preagg.* TO 'root'@'%';

-- =============================================================================
-- MATERIALIZED VIEWS (StarRocks native caching - replaces Redis!)
-- =============================================================================

USE cube_hot;

-- Daily transaction summary (refreshed every 5 minutes)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_transactions_daily_summary
DISTRIBUTED BY HASH(tenant_id) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS
SELECT 
    tenant_id,
    DATE(transaction_date) AS txn_date,
    transaction_type,
    COUNT(*) AS transaction_count,
    SUM(amount) AS total_amount,
    AVG(amount) AS avg_amount,
    MIN(amount) AS min_amount,
    MAX(amount) AS max_amount
FROM transactions
WHERE transaction_date >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY tenant_id, DATE(transaction_date), transaction_type;

-- Portfolio summary (refreshed every minute for near-real-time)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_portfolio_summary
DISTRIBUTED BY HASH(tenant_id) BUCKETS 4
REFRESH ASYNC EVERY (INTERVAL 1 MINUTE)
AS
SELECT 
    tenant_id,
    portfolio_id,
    COUNT(DISTINCT security_id) AS position_count,
    SUM(market_value) AS total_market_value,
    SUM(cost_basis) AS total_cost_basis,
    SUM(unrealized_pnl) AS total_unrealized_pnl,
    SUM(unrealized_pnl) / NULLIF(SUM(cost_basis), 0) AS pnl_percentage
FROM positions
WHERE as_of_date = (SELECT MAX(as_of_date) FROM positions)
GROUP BY tenant_id, portfolio_id;

-- =============================================================================
-- DATA LIFECYCLE: Hot to Cold Migration
-- =============================================================================

-- This routine would run via Temporal to migrate old data to Parquet
-- 
-- 1. Export old transactions to Parquet:
--    SELECT * FROM transactions 
--    WHERE transaction_date < DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY)
--    INTO OUTFILE '/data/parquet/transactions/...'
--    FORMAT AS PARQUET;
--
-- 2. Delete from hot store:
--    DELETE FROM transactions 
--    WHERE transaction_date < DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY);
--
-- 3. Verify cold table picks up new files automatically
