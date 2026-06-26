-- ============================================================================
-- Usice MDM: Semantic Master Data Management (EDM Schema)
-- Database Schema Initialization - Within "alpha" database
-- Aligns with Usice Architecture Section 2.4 (Multi-Tenant Data Layer)
-- ============================================================================
-- NOTE: This schema creates tables in the "edm" schema of the "alpha" database
-- All MDM tables are organized within this single consolidated schema
-- ============================================================================

-- 1. Enable Required Extensions (on alpha database)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "jsonb";

-- 2. Create EDM Schema (if it doesn't exist)
CREATE SCHEMA IF NOT EXISTS edm;

-- Set search path so all CREATE TABLE statements target edm schema
SET search_path TO edm, public;

-- 2. Semantic Terms Registry (Usice Architecture Section 3)
-- ============================================================================
CREATE TABLE semantic_terms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    definition TEXT,
    pii BOOLEAN DEFAULT false,
    governance_level VARCHAR(50) DEFAULT 'standard',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE semantic_terms IS 'Registry of semantic terms used in business objects';
COMMENT ON COLUMN semantic_terms.name IS 'Canonical name (e.g., CalendarDate, IsBusinessDay)';

-- Seed Semantic Terms
INSERT INTO semantic_terms (name, data_type, definition, governance_level) VALUES
('CalendarDate', 'DATE', 'The specific day being defined', 'critical'),
('IsBusinessDay', 'BOOLEAN', 'True if markets/offices are open', 'critical'),
('RegionCode', 'VARCHAR', 'ISO-3166 Country Code (e.g., US, GB)', 'standard'),
('ExchangeCode', 'VARCHAR', 'ISO-10383 MIC Code (e.g., XNYS, XLON)', 'standard'),
('HolidayName', 'VARCHAR', 'Human-readable name (e.g., Independence Day)', 'standard'),
('SourceType', 'VARCHAR', 'Origin of data (e.g., Bloomberg, Exchange, Internal)', 'audit'),
('ConfidenceScore', 'INT', '0-100 score based on survivorship rules', 'calculated');

-- 3. Business Objects Registry (Usice Architecture Section 4)
-- ============================================================================
CREATE TABLE business_objects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    version VARCHAR(20) NOT NULL,
    description TEXT,
    tenant_isolated BOOLEAN DEFAULT true,
    schema_definition JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE business_objects IS 'Registry of business object definitions';

-- Seed Business Object: HolidaySchedule
INSERT INTO business_objects (name, version, description, schema_definition) VALUES
(
  'HolidaySchedule',
  '1.0',
  'Calendar schedule with business day designations',
  '{
    "fields": [
      {"name": "date", "term": "CalendarDate", "required": true},
      {"name": "is_business_day", "term": "IsBusinessDay", "required": true},
      {"name": "region", "term": "RegionCode", "required": true},
      {"name": "exchange", "term": "ExchangeCode", "required": false},
      {"name": "holiday_name", "term": "HolidayName", "required": false},
      {"name": "source", "term": "SourceType", "required": true},
      {"name": "confidence", "term": "ConfidenceScore", "required": true}
    ]
  }'
);

-- 4. Source Registry (Dynamic Configuration)
-- ============================================================================
-- This table determines which sources are ACTIVE vs INACTIVE
CREATE TABLE mdm_source_registry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_name VARCHAR(50) UNIQUE NOT NULL,
    source_type VARCHAR(20) NOT NULL, -- API, PYTHON_SERVICE, BATCH, FILE
    endpoint_url TEXT,
    api_key_secret_name VARCHAR(100), -- Reference to Vault (e.g., "vault:///secrets/tradinghours-api-key")
    is_active BOOLEAN DEFAULT false,
    priority_score INT DEFAULT 0, -- Lower = Higher Priority (1 is best)
    confidence_base INT DEFAULT 0, -- Base confidence when this source wins
    retry_policy JSONB DEFAULT '{"max_retries": 3, "backoff_ms": 1000}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON TABLE mdm_source_registry IS 'Registry of all available calendar data sources. Toggle is_active to enable/disable.';

-- Seed Initial Sources (all configured, only free ones active initially)
INSERT INTO mdm_source_registry (source_name, source_type, endpoint_url, is_active, priority_score, confidence_base) VALUES
-- ACTIVE FREE SOURCES
('NagerDate', 'API', 'https://date.nager.at/api/v3', true, 4, 70),
('OpenHolidays', 'API', 'https://openholidaysapi.org', true, 4, 70),
('Workalendar', 'PYTHON_SERVICE', 'http://workalendar-service:8000', true, 3, 65),
('HolidaysPyPI', 'PYTHON_SERVICE', 'http://holidays-service:8001', true, 3, 65),

-- INACTIVE COMMERCIAL SOURCES (ready to activate later)
('TradingHours', 'API', 'https://api.tradinghours.com/v1', false, 1, 95),
('EODHD', 'API', 'https://eodhd.com/api', false, 2, 90),
('Xignite', 'API', 'https://api.xignite.com', false, 2, 90),
('Finnhub', 'API', 'https://finnhub.io/api', false, 2, 85);

-- 5. Golden Record Table (The Trust Layer)
-- ============================================================================
-- THIS IS THE AUTHORITATIVE SOURCE OF TRUTH
CREATE TABLE mdm_calendar_golden (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    calendar_date DATE NOT NULL,
    is_business_day BOOLEAN NOT NULL,
    region_code VARCHAR(2) NOT NULL,
    exchange_code VARCHAR(4),
    holiday_name VARCHAR(255),
    source_system VARCHAR(50),
    confidence_score INT DEFAULT 0,
    version_id INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    
    -- Unique per tenant + date + region + exchange
    UNIQUE(tenant_id, calendar_date, region_code, exchange_code)
);

CREATE INDEX idx_calendar_golden_tenant_date ON mdm_calendar_golden(tenant_id, calendar_date);
CREATE INDEX idx_calendar_golden_region ON mdm_calendar_golden(region_code);
CREATE INDEX idx_calendar_golden_exchange ON mdm_calendar_golden(exchange_code);

COMMENT ON TABLE mdm_calendar_golden IS 'Golden record table - the authoritative source of truth for calendar data';

-- 6. Source Record Table (Ingestion Staging)
-- ============================================================================
-- Stores all RAW input from sources for audit/lineage
CREATE TABLE mdm_calendar_source (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    source_registry_id UUID REFERENCES mdm_source_registry(id),
    golden_record_id UUID REFERENCES mdm_calendar_golden(id) ON DELETE SET NULL,
    
    calendar_date DATE NOT NULL,
    is_business_day BOOLEAN NOT NULL,
    region_code VARCHAR(2) NOT NULL,
    exchange_code VARCHAR(4),
    holiday_name VARCHAR(255),
    
    raw_payload JSONB, -- Original data structure from source
    normalized_payload JSONB, -- After semantic normalization
    
    ingested_at TIMESTAMPTZ DEFAULT NOW(),
    ingested_by VARCHAR(100)
);

CREATE INDEX idx_calendar_source_tenant ON mdm_calendar_source(tenant_id);
CREATE INDEX idx_calendar_source_registry ON mdm_calendar_source(source_registry_id);
CREATE INDEX idx_calendar_source_golden ON mdm_calendar_source(golden_record_id);

COMMENT ON TABLE mdm_calendar_source IS 'Source records - raw data before survivorship processing';

-- 7. Lineage & Audit Trail (Traceability)
-- ============================================================================
-- EVERY value in the Golden Record has a reason (Rule + Source)
CREATE TABLE mdm_calendar_lineage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    golden_record_id UUID NOT NULL REFERENCES mdm_calendar_golden(id),
    
    semantic_term VARCHAR(50) NOT NULL, -- e.g., "IsBusinessDay"
    winning_source_id UUID REFERENCES mdm_calendar_source(id),
    competing_source_ids UUID[] DEFAULT ARRAY[]::UUID[], -- All candidates
    
    rule_applied VARCHAR(255), -- e.g., "CalendarSurvivorship_Priority1"
    wasm_execution_id UUID, -- Traceable to WASM execution
    
    winning_value TEXT, -- The selected value
    competing_values TEXT[], -- All candidate values
    
    confidence_score INT,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_lineage_tenant ON mdm_calendar_lineage(tenant_id);
CREATE INDEX idx_lineage_golden ON mdm_calendar_lineage(golden_record_id);
CREATE INDEX idx_lineage_term ON mdm_calendar_lineage(semantic_term);

COMMENT ON TABLE mdm_calendar_lineage IS 'Audit trail - proves why every value was chosen';

-- 8. Survivorship Policies (Dynamic Rules)
-- ============================================================================
CREATE TABLE mdm_survivorship_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID, -- NULL = Global policy
    business_object_id UUID REFERENCES business_objects(id),
    semantic_term VARCHAR(50),
    
    dsl_definition TEXT NOT NULL, -- The Starlark DSL code
    description TEXT,
    
    is_active BOOLEAN DEFAULT true,
    version_number INT DEFAULT 1,
    
    compiled_wasm_path VARCHAR(255), -- Path to compiled WASM binary
    last_compiled_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(100),
    updated_by VARCHAR(100)
);

COMMENT ON TABLE mdm_survivorship_policies IS 'Dynamic survivorship policies compiled to WASM';

-- 9. Ingestion Audit (Operations Intelligence)
-- ============================================================================
CREATE TABLE mdm_ingestion_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    job_type VARCHAR(50), -- 'SCHEDULED', 'MANUAL', 'WEBHOOK'
    status VARCHAR(20), -- 'IN_PROGRESS', 'SUCCESS', 'FAILED'
    
    regions_processed VARCHAR[] DEFAULT ARRAY[]::VARCHAR[],
    sources_used VARCHAR[] DEFAULT ARRAY[]::VARCHAR[],
    
    records_ingested INT DEFAULT 0,
    records_processed INT DEFAULT 0,
    conflicts_detected INT DEFAULT 0,
    
    error_message TEXT,
    error_stack_trace TEXT,
    
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INT
);

CREATE INDEX idx_jobs_tenant ON mdm_ingestion_jobs(tenant_id);
CREATE INDEX idx_jobs_status ON mdm_ingestion_jobs(status);

COMMENT ON TABLE mdm_ingestion_jobs IS 'Audit trail of all ingestion runs for operational intelligence';

-- 10. Stewardship Queue (Conflict Resolution)
-- ============================================================================
CREATE TABLE mdm_stewardship_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    golden_record_id UUID NOT NULL REFERENCES mdm_calendar_golden(id),
    
    issue_type VARCHAR(50), -- 'CONFLICT', 'LOW_CONFIDENCE', 'DATA_QUALITY'
    description TEXT,
    
    conflicting_sources UUID[] DEFAULT ARRAY[]::UUID[],
    recommended_action VARCHAR(255),
    
    status VARCHAR(20) DEFAULT 'PENDING', -- 'PENDING', 'REVIEWED', 'RESOLVED'
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    resolution_note TEXT,
    
    priority INT DEFAULT 5, -- 1=critical, 10=low
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stewardship_tenant ON mdm_stewardship_queue(tenant_id);
CREATE INDEX idx_stewardship_status ON mdm_stewardship_queue(status);

COMMENT ON TABLE mdm_stewardship_queue IS 'Queue of conflicts/issues requiring human review';

-- 11. Multi-Tenant Isolation (Row-Level Security)
-- ============================================================================
-- Enable RLS on all tenant-scoped tables
ALTER TABLE mdm_calendar_golden ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_source ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_lineage ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_ingestion_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_stewardship_queue ENABLE ROW LEVEL SECURITY;

-- Policy: Tenant Isolation (Standard Users can only see their tenant's data)
CREATE POLICY tenant_isolation_golden ON mdm_calendar_golden
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_source ON mdm_calendar_source
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_lineage ON mdm_calendar_lineage
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_jobs ON mdm_ingestion_jobs
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY tenant_isolation_stewardship ON mdm_stewardship_queue
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Policy: Ops Can Manage Sources (Global Access)
CREATE POLICY ops_manage_sources ON mdm_source_registry
    FOR ALL USING (current_setting('app.user_role') IN ('global_ops', 'admin'));

-- 12. Materialized View for Operational Intelligence
-- ============================================================================
CREATE MATERIALIZED VIEW mdm_calendar_coverage AS
SELECT
    tenant_id,
    region_code,
    COUNT(*) as total_days,
    SUM(CASE WHEN is_business_day = true THEN 1 ELSE 0 END) as business_days,
    SUM(CASE WHEN is_business_day = false THEN 1 ELSE 0 END) as holidays,
    ROUND(100.0 * SUM(CASE WHEN confidence_score >= 90 THEN 1 ELSE 0 END) / COUNT(*), 2) as high_confidence_pct,
    MAX(updated_at) as last_updated,
    MIN(calendar_date) as earliest_date,
    MAX(calendar_date) as latest_date
FROM mdm_calendar_golden
GROUP BY tenant_id, region_code;

CREATE INDEX idx_coverage_tenant ON mdm_calendar_coverage(tenant_id);

-- 12. Create Application Role (for service connections)
-- ============================================================================
DO $$ BEGIN
    CREATE USER usice_app WITH PASSWORD 'change_me_in_production';
EXCEPTION WHEN duplicate_object THEN
    ALTER USER usice_app WITH PASSWORD 'change_me_in_production';
END $$;

GRANT CONNECT ON DATABASE alpha TO usice_app;
GRANT USAGE ON SCHEMA edm TO usice_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA edm TO usice_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA edm TO usice_app;

-- 13. Create OPS Role (for administrative operations)
-- ============================================================================
DO $$ BEGIN
    CREATE USER usice_ops WITH PASSWORD 'change_me_in_production';
EXCEPTION WHEN duplicate_object THEN
    ALTER USER usice_ops WITH PASSWORD 'change_me_in_production';
END $$;

GRANT CONNECT ON DATABASE alpha TO usice_ops;
GRANT ALL ON ALL TABLES IN SCHEMA edm TO usice_ops;
GRANT ALL ON ALL SEQUENCES IN SCHEMA edm TO usice_ops;

-- 14. Set Default Search Path for Future Sessions
-- ============================================================================
ALTER DATABASE alpha SET search_path = edm, public;

-- ============================================================================
-- NOTES FOR DEPLOYMENT
-- ============================================================================
-- 1. Prerequisites:
--    - PostgreSQL database "alpha" already exists (100.84.126.19:5432)
--    - psql client available
--
-- 2. To run this script:
--    psql -h 100.84.126.19 -U postgres -d alpha -f schema/001_mdm_init.sql
--
-- 3. Schema created: "edm" (Enterprise Data Management)
--    - All MDM tables are in the edm schema
--    - All queries should use edm.table_name or set search_path
--
-- 4. Users created:
--    - usice_app (application user) - SELECT, INSERT, UPDATE, DELETE
--    - usice_ops (operations user) - Full administrative access
--
-- 5. To verify installation:
--    SELECT table_name FROM information_schema.tables 
--    WHERE table_schema = 'edm' 
--    ORDER BY table_name;
--
-- 6. Application Configuration:
--    - Update DB connection string to: postgresql://100.84.126.19:5432/alpha
--    - Set search_path in app: SET search_path TO edm, public;
--    - Set app.current_tenant_id during each connection (from JWT)
--    - Set app.user_role during each connection (from JWT claims)
--
-- 7. Production Checklist:
--    - Change default passwords for usice_app and usice_ops
--    - Create API keys in vault for mdm_source_registry secret references
--    - Compile survivorship DSL files to WASM before use
--    - Configure Redpanda topics for event streaming
--    - Set up backups for the alpha database
-- ============================================================================
