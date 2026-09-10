-- StarRocks Initialization SQL
-- This script sets up the external Iceberg catalog and materialized views

-- ============================================
-- 1. Create External Iceberg Catalog
-- ============================================
CREATE EXTERNAL CATALOG IF NOT EXISTS iceberg_catalog
PROPERTIES (
    "type" = "iceberg",
    "iceberg.catalog.type" = "rest",
    "iceberg.catalog.uri" = "http://nessie:19120/api/v1",
    "iceberg.catalog.warehouse" = "s3://lakehouse/warehouse",
    "aws.s3.endpoint" = "http://minio:9000",
    "aws.s3.access_key" = "minioadmin",
    "aws.s3.secret_key" = "minioadmin",
    "aws.s3.enable_path_style_access" = "true"
);

-- ============================================
-- 2. Resource Groups for Multi-Tenancy
-- ============================================

-- Premium tier: High-priority tenants with dedicated resources
CREATE RESOURCE GROUP IF NOT EXISTS tenant_premium
WITH (
    cpu_weight = 100,
    mem_limit = '40%',
    concurrency_limit = 100,
    type = 'normal'
);

-- Standard tier: Regular tenants with fair-share resources
CREATE RESOURCE GROUP IF NOT EXISTS tenant_standard
WITH (
    cpu_weight = 50,
    mem_limit = '30%',
    concurrency_limit = 50,
    type = 'normal'
);

-- Batch tier: Background jobs and ETL
CREATE RESOURCE GROUP IF NOT EXISTS batch_processing
WITH (
    cpu_weight = 20,
    mem_limit = '20%',
    concurrency_limit = 20,
    type = 'normal'
);

-- ============================================
-- 3. Internal Database for Materialized Views
-- ============================================
CREATE DATABASE IF NOT EXISTS wealth_analytics;
USE wealth_analytics;

-- ============================================
-- 4. Materialized Views for Hot Aggregates
-- ============================================

-- Daily P&L Aggregation (refreshes every 5 minutes)
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_pnl_mv
DISTRIBUTED BY HASH(portfolio_id) BUCKETS 16
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS
SELECT
    DATE_TRUNC('day', event_time) as trade_date,
    tenant_id,
    portfolio_id,
    desk_id,
    currency,
    COUNT(*) as total_trades,
    SUM(ABS(quantity)) as total_volume,
    SUM(notional) as total_notional,
    SUM(CASE WHEN side = 'Buy' THEN notional ELSE 0 END) as buy_notional,
    SUM(CASE WHEN side = 'Sell' THEN notional ELSE 0 END) as sell_notional
FROM iceberg_catalog.wealth.trades
GROUP BY 1, 2, 3, 4, 5;

-- Compliance Statistics (refreshes every 5 minutes)
CREATE MATERIALIZED VIEW IF NOT EXISTS compliance_stats_mv
DISTRIBUTED BY HASH(tenant_id) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 5 MINUTE)
AS
SELECT
    DATE_TRUNC('day', event_time) as date,
    tenant_id,
    rule_id,
    status,
    COUNT(*) as count,
    MAX(event_time) as last_event
FROM iceberg_catalog.wealth.compliance_events
GROUP BY 1, 2, 3, 4;

-- Portfolio Holdings Summary (refreshes every 10 minutes)
CREATE MATERIALIZED VIEW IF NOT EXISTS holdings_summary_mv
DISTRIBUTED BY HASH(portfolio_id) BUCKETS 16
REFRESH ASYNC EVERY (INTERVAL 10 MINUTE)
AS
SELECT
    tenant_id,
    portfolio_id,
    symbol,
    currency,
    SUM(CASE WHEN side = 'Buy' THEN quantity ELSE -quantity END) as net_position,
    SUM(notional) as total_notional,
    COUNT(*) as trade_count,
    MIN(event_time) as first_trade,
    MAX(event_time) as last_trade
FROM iceberg_catalog.wealth.trades
GROUP BY 1, 2, 3, 4;

-- Audit Log Aggregation (for compliance dashboards)
CREATE MATERIALIZED VIEW IF NOT EXISTS audit_summary_mv
DISTRIBUTED BY HASH(tenant_id) BUCKETS 8
REFRESH ASYNC EVERY (INTERVAL 15 MINUTE)
AS
SELECT
    DATE_TRUNC('hour', event_time) as hour,
    tenant_id,
    actor_id,
    action,
    resource_type,
    COUNT(*) as action_count
FROM iceberg_catalog.wealth.audit_log
GROUP BY 1, 2, 3, 4, 5;

-- ============================================
-- 5. Sample User Setup (for development)
-- ============================================

-- Create users for different tenant tiers
-- CREATE USER IF NOT EXISTS 'tenant_001'@'%' IDENTIFIED BY 'secure_password';
-- GRANT SELECT ON iceberg_catalog.wealth.* TO 'tenant_001'@'%';
-- SET PROPERTY FOR 'tenant_001' 'resource_group' = 'tenant_premium';

-- CREATE USER IF NOT EXISTS 'tenant_002'@'%' IDENTIFIED BY 'secure_password';
-- GRANT SELECT ON iceberg_catalog.wealth.* TO 'tenant_002'@'%';
-- SET PROPERTY FOR 'tenant_002' 'resource_group' = 'tenant_standard';

-- ============================================
-- 6. Query Tuning Settings
-- ============================================

-- Enable query profile for debugging
-- SET enable_profile = true;

-- Optimize for Iceberg queries
-- SET enable_scan_block_cache = true;
-- SET pipeline_dop = 0;  -- Auto-detect parallelism

-- ============================================
-- 7. OMS Hot Tier (CDC-fed via Debezium + cmd/stream_loader)
-- ============================================
-- Mirrors of orm.{order,placement,order_allocation,execution,
-- execution_allocation}, kept current by the Debezium Postgres
-- connector (orm-oms-connector, registered against crims/orm's
-- orm_cdc_publication) publishing to Kafka topics orm_oms.orm.*,
-- consumed by one uisce-stream-loader-* service per table
-- (docker-compose.remote.yml). PRIMARY KEY model gives natural
-- upsert-by-id semantics for CDC UPDATE events. Timestamps and IDs
-- are stored as strings since Debezium ships ZonedTimestamp/Uuid
-- logical types as plain strings already; numeric(p,s) columns are
-- decoded from Debezium's base64 Decimal encoding by stream_loader
-- before the row reaches here (see cmd/stream_loader/main.go).
CREATE DATABASE IF NOT EXISTS oms;

CREATE TABLE IF NOT EXISTS oms.orm_order (
    id VARCHAR(36) NOT NULL,
    sec_id VARCHAR(20),
    side VARCHAR(10),
    order_type VARCHAR(20),
    status VARCHAR(20),
    target_qty DECIMAL(18,4),
    executed_qty DECIMAL(18,4),
    leaves_qty DECIMAL(18,4),
    limit_price DECIMAL(18,9),
    avg_price DECIMAL(18,9),
    time_in_force VARCHAR(10),
    trade_date VARCHAR(20),
    manager_id VARCHAR(36),
    trader_id VARCHAR(36),
    created_at VARCHAR(64),
    updated_at VARCHAR(64)
) PRIMARY KEY (id) DISTRIBUTED BY HASH(id) BUCKETS 4 PROPERTIES ('replication_num'='1');

CREATE TABLE IF NOT EXISTS oms.orm_placement (
    id VARCHAR(36) NOT NULL,
    order_id VARCHAR(36),
    broker_id VARCHAR(20),
    venue_id VARCHAR(20),
    routed_qty DECIMAL(18,4),
    executed_qty DECIMAL(18,4),
    leaves_qty DECIMAL(18,4),
    status VARCHAR(20),
    fix_clordid VARCHAR(120),
    created_at VARCHAR(64),
    updated_at VARCHAR(64)
) PRIMARY KEY (id) DISTRIBUTED BY HASH(id) BUCKETS 4 PROPERTIES ('replication_num'='1');

CREATE TABLE IF NOT EXISTS oms.orm_order_allocation (
    id VARCHAR(36) NOT NULL,
    order_id VARCHAR(36),
    account_id VARCHAR(20),
    target_qty DECIMAL(18,4),
    allocated_qty DECIMAL(18,4),
    status VARCHAR(20),
    created_at VARCHAR(64),
    updated_at VARCHAR(64)
) PRIMARY KEY (id) DISTRIBUTED BY HASH(id) BUCKETS 4 PROPERTIES ('replication_num'='1');

CREATE TABLE IF NOT EXISTS oms.orm_execution (
    id VARCHAR(36) NOT NULL,
    placement_id VARCHAR(36),
    order_id VARCHAR(36),
    exec_qty DECIMAL(18,4),
    exec_price DECIMAL(18,9),
    exec_time VARCHAR(64),
    broker_exec_id VARCHAR(120),
    last_capacity VARCHAR(1),
    created_at VARCHAR(64)
) PRIMARY KEY (id) DISTRIBUTED BY HASH(id) BUCKETS 4 PROPERTIES ('replication_num'='1');

CREATE TABLE IF NOT EXISTS oms.orm_execution_allocation (
    id VARCHAR(36) NOT NULL,
    execution_id VARCHAR(36),
    order_allocation_id VARCHAR(36),
    alloc_exec_qty DECIMAL(18,4),
    alloc_exec_price DECIMAL(18,9),
    created_at VARCHAR(64)
) PRIMARY KEY (id) DISTRIBUTED BY HASH(id) BUCKETS 4 PROPERTIES ('replication_num'='1');
