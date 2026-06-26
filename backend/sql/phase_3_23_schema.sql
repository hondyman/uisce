-- Discovery feature discovery database schema
-- Phase 3.23-C: Discovery API persistent layer

-- Discovery runs table: tracks each discovery execution
CREATE TABLE IF NOT EXISTS discovery_runs (
    id SERIAL PRIMARY KEY,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL, -- pending, running, success, failed, partial
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    sources_scanned JSONB, -- ["postgres", "trino", "logs", "prometheus"]
    candidates_found INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discovery_runs_run_id ON discovery_runs(run_id);
CREATE INDEX idx_discovery_runs_status ON discovery_runs(status);
CREATE INDEX idx_discovery_runs_started_at ON discovery_runs(started_at DESC);

-- Discovery candidates table: discovered feature candidates
CREATE TABLE IF NOT EXISTS discovery_candidates (
    id SERIAL PRIMARY KEY,
    candidate_id VARCHAR(255) UNIQUE NOT NULL,
    run_id VARCHAR(255) NOT NULL REFERENCES discovery_runs(run_id),
    name VARCHAR(255) NOT NULL,
    source_database VARCHAR(50) NOT NULL, -- postgres, trino, logs, prometheus, derived
    source_schema VARCHAR(255),
    source_table VARCHAR(255),
    source_field VARCHAR(255) NOT NULL,
    data_type VARCHAR(50) NOT NULL, -- float, string, integer, boolean, categorical, timestamp
    description TEXT,
    completeness FLOAT CHECK (completeness >= 0 AND completeness <= 1),
    cardinality BIGINT,
    business_value FLOAT CHECK (business_value >= 0 AND business_value <= 1),
    technical_score FLOAT CHECK (technical_score >= 0 AND technical_score <= 1),
    status VARCHAR(50) NOT NULL DEFAULT 'candidate', -- candidate, approved, rejected
    discovered_at TIMESTAMP NOT NULL,
    approved_by VARCHAR(255),
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discovery_candidates_run_id ON discovery_candidates(run_id);
CREATE INDEX idx_discovery_candidates_status ON discovery_candidates(status);
CREATE INDEX idx_discovery_candidates_business_value ON discovery_candidates(business_value DESC);
CREATE INDEX idx_discovery_candidates_source_db ON discovery_candidates(source_database);
CREATE INDEX idx_discovery_candidates_name ON discovery_candidates(name);
CREATE UNIQUE INDEX idx_discovery_candidates_name_source ON discovery_candidates(name, source_database) WHERE status = 'candidate';

-- Feature catalog integration table: maps approved candidates to features
CREATE TABLE IF NOT EXISTS feature_catalog_mappings (
    id SERIAL PRIMARY KEY,
    candidate_id VARCHAR(255) NOT NULL REFERENCES discovery_candidates(candidate_id),
    feature_name VARCHAR(255) NOT NULL,
    feature_version INT DEFAULT 1,
    catalog_status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, deprecated
    mapped_by VARCHAR(255),
    mapped_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deprecation_reason TEXT,
    deprecated_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feature_catalog_mappings_feature_name ON feature_catalog_mappings(feature_name);
CREATE INDEX idx_feature_catalog_mappings_status ON feature_catalog_mappings(catalog_status);

-- Discovery statistics table: pre-computed stats for dashboard
CREATE TABLE IF NOT EXISTS discovery_statistics (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    total_candidates INT,
    approved_count INT,
    rejected_count INT,
    avg_score FLOAT,
    median_score FLOAT,
    source_distribution JSONB, -- {"postgres": 35, "trino": 12, "logs": 18, "prometheus": 28}
    data_type_distribution JSONB,
    score_distribution JSONB, -- {"0.0-0.2": 7, "0.2-0.4": 35, "0.4-0.6": 45, "0.6-0.8": 30, "0.8-1.0": 18}
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_discovery_statistics_date ON discovery_statistics(date);

-- Discovery logs table: detailed logging of discovery process
CREATE TABLE IF NOT EXISTS discovery_logs (
    id SERIAL PRIMARY KEY,
    run_id VARCHAR(255) NOT NULL REFERENCES discovery_runs(run_id),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(100), -- postgres_scanner, trino_scanner, log_parser, metric_extractor, ranker, generator
    action VARCHAR(100), -- scan_start, scan_complete, parse_start, parse_complete, etc
    details TEXT,
    status VARCHAR(50), -- info, warning, error
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discovery_logs_run_id ON discovery_logs(run_id);
CREATE INDEX idx_discovery_logs_timestamp ON discovery_logs(timestamp DESC);
CREATE INDEX idx_discovery_logs_source ON discovery_logs(source);

-- Audit table: track approvals and rejections
CREATE TABLE IF NOT EXISTS discovery_audit (
    id SERIAL PRIMARY KEY,
    candidate_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255),
    action VARCHAR(50) NOT NULL, -- approved, rejected, marked_as_unused
    reason TEXT,
    previous_status VARCHAR(50),
    new_status VARCHAR(50),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_discovery_audit_candidate_id ON discovery_audit(candidate_id);
CREATE INDEX idx_discovery_audit_timestamp ON discovery_audit(timestamp DESC);
CREATE INDEX idx_discovery_audit_user_id ON discovery_audit(user_id);

-- Feature metadata cache table: for quick lookups
CREATE TABLE IF NOT EXISTS feature_metadata (
    feature_name VARCHAR(255) PRIMARY KEY,
    source_database VARCHAR(50),
    source_field VARCHAR(255),
    data_type VARCHAR(50),
    completeness FLOAT,
    cardinality BIGINT,
    importance_score FLOAT,
    last_computed_at TIMESTAMP,
    last_used_at TIMESTAMP,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feature_metadata_created_at ON feature_metadata(created_at DESC);

-- Insert default discovery run entry for testing
INSERT INTO discovery_runs (run_id, status, sources_scanned, candidates_found)
VALUES ('discovery-2026-02-09-001', 'success', '["postgres", "trino", "logs", "prometheus"]', 127)
ON CONFLICT (run_id) DO NOTHING;

-- Update statistics table
INSERT INTO discovery_statistics (
    date,
    total_candidates,
    approved_count,
    rejected_count,
    avg_score,
    median_score,
    source_distribution,
    data_type_distribution,
    score_distribution
) VALUES (
    CURRENT_DATE,
    127,
    12,
    5,
    0.62,
    0.61,
    '{"postgres": 35, "trino": 12, "logs": 18, "prometheus": 28, "derived": 34}',
    '{"float": 65, "string": 35, "integer": 15, "categorical": 12}',
    '{"0.0-0.2": 7, "0.2-0.4": 35, "0.4-0.6": 45, "0.6-0.8": 30, "0.8-1.0": 18}'
) ON CONFLICT (date) DO NOTHING;

-- Table constraints and additions
ALTER TABLE discovery_runs
  ADD CONSTRAINT check_status_valid 
  CHECK (status IN ('pending', 'running', 'success', 'failed', 'partial'));

ALTER TABLE discovery_candidates
  ADD CONSTRAINT check_candidate_status
  CHECK (status IN ('candidate', 'approved', 'rejected'));

ALTER TABLE feature_catalog_mappings
  ADD CONSTRAINT check_catalog_status
  CHECK (catalog_status IN ('pending', 'active', 'deprecated'));

-- Add comments for documentation
COMMENT ON TABLE discovery_runs IS 'Tracks each feature discovery execution (workflow run)';
COMMENT ON TABLE discovery_candidates IS 'Discovered feature candidates from schema/logs/metrics';
COMMENT ON TABLE feature_catalog_mappings IS 'Links approved candidates to feature catalog entries';
COMMENT ON TABLE discovery_statistics IS 'Pre-computed aggregate statistics for dashboard';
COMMENT ON TABLE discovery_logs IS 'Detailed logs of discovery workflow execution';
COMMENT ON TABLE discovery_audit IS 'Audit trail of approvals/rejections';
COMMENT ON COLUMN discovery_runs.run_id IS 'Unique identifier for discovery run';
COMMENT ON COLUMN discovery_candidates.status IS 'candidate (pending review), approved (in catalog), rejected (dismissed)';
COMMENT ON COLUMN discovery_candidates.business_value IS 'Score 0-1: relevance to business metrics';
