-- Create StarRocks table for compliance audit
-- This should be run in StarRocks via MySQL client

CREATE DATABASE IF NOT EXISTS alpha;

USE alpha;

CREATE TABLE IF NOT EXISTS compliance_audit (
    event_id VARCHAR(36),
    trace_id VARCHAR(255),
    event_type VARCHAR(50),
    status VARCHAR(50),
    rule_version VARCHAR(50),
    trade_id VARCHAR(255),
    amount DECIMAL(18,2),
    currency VARCHAR(10),
    order_type VARCHAR(50),
    error_details JSON,
    created_at DATETIME
)
DUPLICATE KEY(event_id)
DISTRIBUTED BY HASH(trace_id) BUCKETS 10
PROPERTIES (
    "replication_num" = "1",
    "storage_medium" = "SSD",
    "storage_cooldown_time" = "2099-01-01 00:00:00"
);

-- Create indexes for common queries
CREATE INDEX idx_trace_id ON compliance_audit(trace_id);
CREATE INDEX idx_created_at ON compliance_audit(created_at);
CREATE INDEX idx_status ON compliance_audit(status);

-- Verify table creation
SHOW CREATE TABLE compliance_audit;
