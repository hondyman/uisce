-- Migration: 001_s-- Speed Layer Schema for High-Frequency Investment Data
-- Description: Creates StarRocks tables for high-frequency investment data and performance analytics.

-- 1. Daily Performance Table (MergeTree)
-- Stores daily return factors for TWR calculation.
CREATE TABLE IF NOT EXISTS daily_performance (
    portfolio_id UInt64,
    date Date,
    return_factor Float64, -- (1 + daily_return)
    market_value Float64,
    cash_flow Float64
) ENGINE = MergeTree()
ORDER BY (portfolio_id, date);

-- 2. Materialized View for Monthly TWR (AggregatingMergeTree)
-- Pre-aggregates daily returns using Log-Sum-Exp logic for fast multi-year queries.
CREATE TABLE IF NOT EXISTS monthly_performance_agg (
    portfolio_id UInt64,
    month Date,
    log_sum_return Float64 -- sum(log(return_factor))
) ENGINE = AggregatingMergeTree()
ORDER BY (portfolio_id, month);

CREATE MATERIALIZED VIEW IF NOT EXISTS monthly_performance_mv 
TO monthly_performance_agg
AS SELECT
    portfolio_id,
    toStartOfMonth(date) as month,
    sum(log(return_factor)) as log_sum_return
FROM daily_performance
GROUP BY portfolio_id, month;

-- 3. Current Positions Table (ReplacingMergeTree)
-- Stores the latest snapshot of positions for drift analysis.
-- Deduplicates based on (portfolio_id, security_id) using the highest version.
CREATE TABLE IF NOT EXISTS current_positions (
    portfolio_id UInt64,
    security_id String,
    quantity Float64,
    market_value Float64,
    weight Float64,
    model_id UInt64,
    updated_at DateTime,
    version UInt64 -- Monotonically increasing version for deduplication
) ENGINE = ReplacingMergeTree(version)
ORDER BY (portfolio_id, security_id);

-- 4. Target Models Dictionary (Mock Definition)
-- In production, this would be a dictionary source pointing to Postgres or a file.
-- CREATE DICTIONARY target_models_dict (
--     model_id UInt64,
--     security_id String,
--     target_weight Float64
-- ) PRIMARY KEY model_id, security_id
-- SOURCE(POSTGRESQL(...));
