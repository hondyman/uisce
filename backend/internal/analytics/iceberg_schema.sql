-- Iceberg Schema for Hybrid Analytics (Cold Store)
-- These tables are typically managed by a catalog like AWS Glue, Nessie, or Hive Metastore.
-- The syntax below is generic SQL for defining Iceberg tables (e.g., via Trino or Spark SQL).

-- 1. Trades History (Long-term Retention)
CREATE TABLE IF NOT EXISTS trades_history (
    trade_id UUID,
    portfolio_id UUID,
    desk_id VARCHAR,
    symbol VARCHAR,
    side VARCHAR,
    quantity DECIMAL(18, 4),
    price DECIMAL(18, 4),
    notional DECIMAL(18, 4),
    currency VARCHAR,
    basis_id VARCHAR,
    event_time TIMESTAMP(6),
    ingestion_time TIMESTAMP(6)
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['day(event_time)', 'portfolio_id']
);

-- 2. Compliance History (Audit Trail)
CREATE TABLE IF NOT EXISTS compliance_history (
    event_time TIMESTAMP(6),
    rule_id VARCHAR,
    portfolio_id UUID,
    status VARCHAR,
    details VARCHAR,
    ingestion_time TIMESTAMP(6)
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['month(event_time)']
);

-- 3. Unified Audit Archive (Immutable Evidence)
CREATE TABLE IF NOT EXISTS audit_archive (
    event_time TIMESTAMP(6),
    trace_id UUID,
    actor_id VARCHAR,
    action VARCHAR,
    resource_type VARCHAR,
    resource_id VARCHAR,
    payload VARCHAR, -- JSON stored as string
    ingestion_time TIMESTAMP(6)
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['year(event_time)', 'month(event_time)']
);
