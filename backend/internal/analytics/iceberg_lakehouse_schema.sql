-- Iceberg Schema for StarRocks Lakehouse
-- These tables are created via Trino or Spark SQL against the Nessie catalog
-- StarRocks queries them directly via the external Iceberg catalog

-- ============================================
-- Create Iceberg Database (via Trino/Spark)
-- ============================================
-- CREATE SCHEMA IF NOT EXISTS iceberg.wealth;

-- ============================================
-- 1. Trades Table (High-Volume: 200M+ trades/day per tenant)
-- ============================================
CREATE TABLE IF NOT EXISTS iceberg.wealth.trades (
    -- Identifiers
    trade_id VARCHAR NOT NULL,
    tenant_id VARCHAR NOT NULL,
    portfolio_id VARCHAR NOT NULL,
    desk_id VARCHAR,
    
    -- Trade details
    symbol VARCHAR NOT NULL,
    side VARCHAR NOT NULL,  -- 'Buy', 'Sell'
    quantity DECIMAL(18, 4) NOT NULL,
    price DECIMAL(18, 4) NOT NULL,
    notional DECIMAL(18, 4) NOT NULL,
    currency VARCHAR NOT NULL,
    
    -- Classification
    basis_id VARCHAR,  -- 'IBOR', 'ABOR'
    asset_class VARCHAR,
    instrument_type VARCHAR,
    
    -- Timestamps
    event_time TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    trade_date DATE NOT NULL,
    settlement_date DATE,
    
    -- Audit
    created_at TIMESTAMP(6) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    source_system VARCHAR
)
WITH (
    format = 'PARQUET',
    -- Partition by day + tenant bucket for optimal query pruning
    partitioning = ARRAY[
        'day(event_time)',
        'bucket(tenant_id, 32)'
    ],
    -- Sort within partitions for better compression and range scans
    sorted_by = ARRAY['portfolio_id', 'symbol', 'event_time'],
    -- Target 256MB files for optimal scan performance
    'write.target-file-size-bytes' = '268435456',
    -- Enable merge-on-read for efficient updates
    'write.delete.mode' = 'merge-on-read',
    'write.update.mode' = 'merge-on-read'
);

-- ============================================
-- 2. Compliance Events Table
-- ============================================
CREATE TABLE IF NOT EXISTS iceberg.wealth.compliance_events (
    -- Identifiers
    event_id VARCHAR NOT NULL,
    tenant_id VARCHAR NOT NULL,
    
    -- Event details
    rule_id VARCHAR NOT NULL,
    rule_name VARCHAR,
    portfolio_id VARCHAR,
    
    -- Result
    status VARCHAR NOT NULL,  -- 'Pass', 'Fail', 'Warning'
    severity VARCHAR,  -- 'Critical', 'High', 'Medium', 'Low'
    details VARCHAR,  -- JSON payload
    
    -- Timestamps
    event_time TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    
    -- Audit
    evaluated_by VARCHAR,
    workflow_id VARCHAR
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'month(event_time)',
        'bucket(tenant_id, 16)'
    ],
    sorted_by = ARRAY['rule_id', 'event_time']
);

-- ============================================
-- 3. Unified Audit Log (Immutable Evidence)
-- ============================================
CREATE TABLE IF NOT EXISTS iceberg.wealth.audit_log (
    -- Identifiers
    trace_id VARCHAR NOT NULL,
    span_id VARCHAR,
    tenant_id VARCHAR NOT NULL,
    
    -- Actor
    actor_id VARCHAR NOT NULL,
    actor_type VARCHAR,  -- 'user', 'service', 'workflow'
    
    -- Action
    action VARCHAR NOT NULL,
    resource_type VARCHAR NOT NULL,
    resource_id VARCHAR,
    
    -- Payload
    payload VARCHAR,  -- JSON
    
    -- Result
    outcome VARCHAR,  -- 'success', 'failure'
    error_message VARCHAR,
    
    -- Timestamps
    event_time TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    
    -- Context
    ip_address VARCHAR,
    user_agent VARCHAR,
    workflow_id VARCHAR
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'year(event_time)',
        'month(event_time)',
        'bucket(tenant_id, 8)'
    ],
    sorted_by = ARRAY['trace_id', 'event_time']
);

-- ============================================
-- 4. Ledger Entries (Multi-Basis Accounting)
-- ============================================
CREATE TABLE IF NOT EXISTS iceberg.wealth.ledger_entries (
    -- Identifiers
    entry_id VARCHAR NOT NULL,
    tenant_id VARCHAR NOT NULL,
    
    -- Accounting
    basis_id VARCHAR NOT NULL,  -- 'IBOR', 'ABOR', 'NAV'
    account_id VARCHAR NOT NULL,
    asset_id VARCHAR NOT NULL,
    
    -- Amounts
    quantity DECIMAL(18, 4) NOT NULL,
    local_amount DECIMAL(18, 4),
    base_amount DECIMAL(18, 4),
    currency VARCHAR NOT NULL,
    
    -- Bi-temporal timestamps
    event_time TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    valid_from TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    valid_to TIMESTAMP(6) WITH TIME ZONE,
    
    -- Reference
    transaction_ref VARCHAR,
    source_trade_id VARCHAR,
    
    -- Audit
    created_at TIMESTAMP(6) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'month(event_time)',
        'bucket(tenant_id, 16)',
        'basis_id'
    ],
    sorted_by = ARRAY['account_id', 'asset_id', 'event_time']
);

-- ============================================
-- 5. Portfolio Snapshots (Point-in-Time)
-- ============================================
CREATE TABLE IF NOT EXISTS iceberg.wealth.portfolio_snapshots (
    -- Identifiers
    snapshot_id VARCHAR NOT NULL,
    tenant_id VARCHAR NOT NULL,
    portfolio_id VARCHAR NOT NULL,
    
    -- Snapshot timestamp
    as_of_date DATE NOT NULL,
    as_of_time TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    
    -- Holdings (JSON array)
    holdings VARCHAR,  -- JSON: [{symbol, quantity, price, value}, ...]
    
    -- Summary metrics
    total_market_value DECIMAL(18, 4),
    cash_balance DECIMAL(18, 4),
    total_positions INT,
    
    -- Hash for integrity
    content_hash VARCHAR,
    
    -- Audit
    created_at TIMESTAMP(6) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'month(as_of_date)',
        'bucket(tenant_id, 8)'
    ],
    sorted_by = ARRAY['portfolio_id', 'as_of_date']
);

-- ============================================
-- 6. Workflow Artifacts (Temporal Integration)
-- ============================================
CREATE TABLE IF NOT EXISTS iceberg.wealth.workflow_artifacts (
    -- Identifiers
    artifact_id VARCHAR NOT NULL,
    tenant_id VARCHAR NOT NULL,
    workflow_id VARCHAR NOT NULL,
    run_id VARCHAR,
    
    -- Artifact details
    artifact_type VARCHAR NOT NULL,  -- 'prompt', 'response', 'policy_eval', 'snapshot'
    artifact_name VARCHAR,
    
    -- Content
    content VARCHAR,  -- JSON or text
    content_hash VARCHAR NOT NULL,  -- SHA-256 for integrity
    
    -- Iceberg snapshot reference (for replay)
    iceberg_snapshot_id BIGINT,
    
    -- Timestamps
    created_at TIMESTAMP(6) WITH TIME ZONE NOT NULL,
    
    -- Metadata
    metadata VARCHAR  -- JSON
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY[
        'month(created_at)',
        'bucket(tenant_id, 8)'
    ],
    sorted_by = ARRAY['workflow_id', 'created_at']
);

-- ============================================
-- Maintenance Procedures (run via Spark/Trino)
-- ============================================

-- Compact small files (run daily)
-- CALL iceberg.system.rewrite_data_files(
--     table => 'wealth.trades',
--     strategy => 'sort',
--     sort_order => 'portfolio_id ASC, event_time ASC',
--     options => map(
--         'target-file-size-bytes', '268435456',
--         'min-input-files', '5'
--     )
-- );

-- Expire old snapshots (keep 7 days for replay)
-- CALL iceberg.system.expire_snapshots(
--     table => 'wealth.trades',
--     older_than => TIMESTAMP '2024-01-01 00:00:00',
--     retain_last => 168
-- );

-- Remove orphan files
-- CALL iceberg.system.remove_orphan_files(
--     table => 'wealth.trades',
--     older_than => TIMESTAMP '2024-01-01 00:00:00'
-- );
