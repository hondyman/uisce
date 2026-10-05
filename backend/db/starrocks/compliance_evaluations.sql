-- StarRocks Warm Tier DDL for Compliance Evaluations & Surveillance
-- Architecture: Primary Key Model on (lineage_id, evaluated_at) for idempotent Stream Load deduplication
-- Partitioning: Monthly on evaluated_at for efficient analytical pruning and historical retention drops

CREATE DATABASE IF NOT EXISTS oms;

CREATE TABLE IF NOT EXISTS oms.compliance_evaluations (
    lineage_id VARCHAR(36) NOT NULL,
    evaluated_at DATETIME NOT NULL,
    tenant_id VARCHAR(36) NOT NULL,
    order_id VARCHAR(36),
    account_id VARCHAR(36),
    security_id VARCHAR(36),
    rule_id VARCHAR(36) NOT NULL,
    rule_version INT NOT NULL,
    action_taken VARCHAR(32) NOT NULL,
    passed BOOLEAN NOT NULL,
    latency_micros BIGINT NOT NULL,
    evaluation_hash VARCHAR(64) NOT NULL,
    input_params JSON,
    metric_snapshots JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
PRIMARY KEY (lineage_id, evaluated_at)
PARTITION BY date_trunc('month', evaluated_at)
DISTRIBUTED BY HASH(lineage_id) BUCKETS 16
PROPERTIES (
    "replication_num" = "1",
    "enable_persistent_index" = "true"
);
