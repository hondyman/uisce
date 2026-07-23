-- =====================================================================
-- Iceberg Audit & Snapshot Plane - DDL Definitions
-- Production-ready schema for multi-tenant audit fabric
-- =====================================================================

-- =====================================================================
-- 1. SCHEDULER JOB RUNS
-- Tracks every job execution with full semantic and compliance context
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.scheduler_job_runs (
    run_id              VARCHAR,
    job_id              VARCHAR,
    dag_id              VARCHAR,
    tenant_id           VARCHAR NOT NULL,
    start_ts            TIMESTAMP NOT NULL,
    end_ts              TIMESTAMP,
    status              VARCHAR NOT NULL,
    error_message       VARCHAR,
    
    -- JSON context fields
    semantic_context    VARCHAR,  -- JSON stored as string in Iceberg
    compliance_context  VARCHAR,  -- JSON stored as string in Iceberg
    slo_context         VARCHAR,  -- JSON stored as string in Iceberg
    ai_narrative        VARCHAR,  -- JSON stored as string in Iceberg
    metadata            VARCHAR,  -- JSON stored as string in Iceberg
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['tenant_id', 'day(start_ts)'],
    location = 's3a://audit/scheduler_job_runs/'
);

-- =====================================================================
-- 2. SCHEDULER DAG RUNS
-- Tracks DAG-level orchestration
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.scheduler_dag_runs (
    dag_run_id          VARCHAR,
    dag_id              VARCHAR NOT NULL,
    tenant_id           VARCHAR NOT NULL,
    start_ts            TIMESTAMP NOT NULL,
    end_ts              TIMESTAMP,
    status              VARCHAR NOT NULL,
    
    -- JSON context fields
    critical_path       VARCHAR,  -- JSON stored as string
    ai_root_cause       VARCHAR,  -- JSON stored as string
    metadata            VARCHAR,  -- JSON stored as string
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['tenant_id', 'day(start_ts)'],
    location = 's3a://audit/scheduler_dag_runs/'
);

-- =====================================================================
-- 3. GOVERNANCE CHANGESETS
-- Every ChangeSet is immutable and auditable
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.governance_changesets (
    changeset_id        VARCHAR NOT NULL,
    type                VARCHAR NOT NULL,
    actor               VARCHAR NOT NULL,
    tenant_id           VARCHAR,  -- nullable for global changes
    created_at          TIMESTAMP NOT NULL,
    
    -- Change payload
    payload_old         VARCHAR,  -- JSON stored as string
    payload_new         VARCHAR,  -- JSON stored as string
    
    -- Impact analysis
    semantic_impact     VARCHAR,  -- JSON stored as string
    compliance_impact   VARCHAR,  -- JSON stored as string
    tenant_impact       VARCHAR,  -- JSON stored as string (multi-tenant)
    
    -- AI insights
    ai_summary          VARCHAR,  -- JSON stored as string
    ai_risk             VARCHAR,  -- JSON stored as string
    
    -- Approval
    approvers           ARRAY(VARCHAR),
    status              VARCHAR NOT NULL,
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['day(created_at)'],
    location = 's3a://audit/governance_changesets/'
);

-- =====================================================================
-- 4. SEMANTIC SNAPSHOTS
-- Full semantic graph snapshots for time-travel lineage
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.semantic_snapshots (
    snapshot_id         VARCHAR NOT NULL,
    semantic_term_id    VARCHAR NOT NULL,
    version             INT NOT NULL,
    timestamp           TIMESTAMP NOT NULL,
    definition          VARCHAR,
    business_term_id    VARCHAR,
    tenant_id           VARCHAR,  -- nullable for global terms
    
    -- Context
    compliance          VARCHAR,  -- JSON stored as string
    lineage             VARCHAR,  -- JSON stored as string
    metadata            VARCHAR,  -- JSON stored as string
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['semantic_term_id'],
    location = 's3a://audit/semantic_snapshots/'
);

-- =====================================================================
-- 5. ORCHESTRATION EVENTS
-- Every Temporal workflow event
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.orchestration_events (
    event_id            VARCHAR NOT NULL,
    workflow_id         VARCHAR NOT NULL,
    event_type          VARCHAR NOT NULL,
    tenant_id           VARCHAR,  -- nullable for global workflows
    timestamp           TIMESTAMP NOT NULL,
    
    -- Context
    payload             VARCHAR,  -- JSON stored as string
    compliance_context  VARCHAR,  -- JSON stored as string
    semantic_context    VARCHAR,  -- JSON stored as string
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['day(timestamp)'],
    location = 's3a://audit/orchestration_events/'
);

-- =====================================================================
-- 6. AI AUDIT SUGGESTIONS
-- AI-generated narratives and insights
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.ai_suggestions (
    suggestion_id       VARCHAR NOT NULL,
    audit_record_id     VARCHAR NOT NULL,
    record_type         VARCHAR NOT NULL,
    tenant_id           VARCHAR NOT NULL,
    timestamp           TIMESTAMP NOT NULL,
    
    -- AI content
    narrative           VARCHAR NOT NULL,
    root_cause          VARCHAR,
    blast_radius        VARCHAR,
    recommended_fix     VARCHAR,
    suggested_actions   VARCHAR,  -- JSON stored as string
    confidence          DOUBLE,
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['tenant_id', 'day(timestamp)'],
    location = 's3a://audit/ai_suggestions/'
);

-- =====================================================================
-- 7. COMPLIANCE VIOLATIONS
-- Tracks violations for regulator reporting
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.audit.compliance_violations (
    violation_id        VARCHAR NOT NULL,
    tenant_id           VARCHAR NOT NULL,
    job_run_id          VARCHAR,
    violated_at         TIMESTAMP NOT NULL,
    remediated_at       TIMESTAMP,
    violation_type      VARCHAR NOT NULL,
    severity            VARCHAR NOT NULL,
    
    -- Details
    pii_exposed         BOOLEAN NOT NULL,
    affected_records    BIGINT,
    compliance_refs     ARRAY(VARCHAR),
    narrative           VARCHAR,
    metadata            VARCHAR,  -- JSON stored as string
    
    -- Internal metadata
    _ingest_ts          TIMESTAMP NOT NULL,
    _source_service     VARCHAR NOT NULL,
    _schema_version     INT NOT NULL
)
WITH (
    format = 'PARQUET',
    partitioning = ARRAY['tenant_id', 'day(violated_at)'],
    location = 's3a://audit/compliance_violations/'
);

-- =====================================================================
-- 8. TENANT RETENTION POLICIES (Control Plane)
-- Per-tenant retention configuration
-- =====================================================================
CREATE TABLE IF NOT EXISTS iceberg.platform.tenant_retention_policies (
    tenant_id               VARCHAR PRIMARY KEY,
    audit_retention_days    INT NOT NULL DEFAULT 2555,  -- 7 years
    pii_retention_days      INT NOT NULL DEFAULT 2555,
    snapshot_retention_days INT NOT NULL DEFAULT 1825,  -- 5 years
    updated_at              TIMESTAMP NOT NULL,
    updated_by              VARCHAR NOT NULL
)
WITH (
    format = 'PARQUET',
    location = 's3a://platform/tenant_retention_policies/'
);

-- =====================================================================
-- COMMENTS for documentation
-- =====================================================================
COMMENT ON TABLE iceberg.audit.scheduler_job_runs IS 'Immutable audit log of all scheduler job executions with semantic and compliance context';
COMMENT ON TABLE iceberg.audit.scheduler_dag_runs IS 'Immutable audit log of DAG-level orchestration';
COMMENT ON TABLE iceberg.audit.governance_changesets IS 'Immutable audit log of all governance changesets with impact analysis';
COMMENT ON TABLE iceberg.audit.semantic_snapshots IS 'Time-travel snapshots of semantic graph for lineage queries';
COMMENT ON TABLE iceberg.audit.orchestration_events IS 'Immutable audit log of Temporal workflow events';
COMMENT ON TABLE iceberg.audit.ai_suggestions IS 'AI-generated audit narratives and incident analysis';
COMMENT ON TABLE iceberg.audit.compliance_violations IS 'Compliance violation tracking for regulator reporting';
COMMENT ON TABLE iceberg.platform.tenant_retention_policies IS 'Per-tenant audit retention policies';
