-- StarRocks Schema Definitions

CREATE DATABASE IF NOT EXISTS intelligence;

-- Table: client_events
-- Stores every interaction a user has with the platform.
CREATE TABLE IF NOT EXISTS intelligence.client_events
(
    timestamp DateTime64(3) CODEC(Delta, ZSTD(1)), -- Millisecond precision
    user_id UInt64,
    session_id UUID,
    event_type LowCardinality(String),
    url_path String,
    device_id String,
    metadata String, -- JSON blob for extra context
    
    -- Projection for partition pruning
    date Date ALIAS toDate(timestamp)
)
ENGINE = MergeTree
PARTITION BY toStartOfMonth(date)
ORDER BY (user_id, timestamp, event_type)
SETTINGS index_granularity = 8192;

-- Table: market_ticks
-- Stores the reference market data (e.g., S&P 500 index values).
CREATE TABLE IF NOT EXISTS intelligence.market_ticks
(
    timestamp DateTime64(3) CODEC(Delta, ZSTD(1)),
    symbol LowCardinality(String), -- e.g., 'SPX'
    price Float64 CODEC(Gorilla, ZSTD(1)) -- Gorilla codec optimized for floating point series
)
ENGINE = MergeTree
PARTITION BY toStartOfYear(timestamp)
ORDER BY (symbol, timestamp);
