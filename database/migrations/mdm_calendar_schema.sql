-- MDM Calendar Gold Copy Schema
-- Implements multi-tenant, versioned, audited calendar data management
-- with Row-Level Security (RLS) and survivorship tracking

-- ============================================================================
-- 1. GOLDEN RECORD TABLE (Trust Layer)
-- ============================================================================
CREATE TABLE IF NOT EXISTS mdm_calendar_golden (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Semantic Fields
    calendar_date DATE NOT NULL,
    is_business_day BOOLEAN NOT NULL,
    region_code VARCHAR(2) NOT NULL,
    exchange_code VARCHAR(4),
    holiday_name VARCHAR(255),
    
    -- Governance
    source_type VARCHAR(50) NOT NULL,
    confidence_score INT DEFAULT 0 CHECK (confidence_score >= 0 AND confidence_score <= 100),
    version_id INT DEFAULT 1,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    -- Audit Trail
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    -- Unique constraint per tenant/date/region/exchange
    UNIQUE(tenant_id, calendar_date, region_code, exchange_code),
    
    -- Indexes for common queries
    INDEX idx_mdm_golden_tenant_date (tenant_id, calendar_date),
    INDEX idx_mdm_golden_region (tenant_id, region_code),
    INDEX idx_mdm_golden_exchange (tenant_id, exchange_code),
    INDEX idx_mdm_golden_business_day (tenant_id, is_business_day),
    INDEX idx_mdm_golden_created (tenant_id, created_at DESC)
);

-- ============================================================================
-- 2. SOURCE RECORD TABLE (Ingestion Staging)
-- ============================================================================
CREATE TABLE IF NOT EXISTS mdm_calendar_source (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_record_id UUID REFERENCES mdm_calendar_golden(id) ON DELETE SET NULL,
    
    -- Raw Data
    calendar_date DATE NOT NULL,
    is_business_day BOOLEAN NOT NULL,
    region_code VARCHAR(2) NOT NULL,
    exchange_code VARCHAR(4),
    holiday_name VARCHAR(255),
    
    -- Source Attribution
    source_system VARCHAR(50) NOT NULL,  -- e.g., "Bloomberg", "ExchangeFeed", "ManualSteward"
    external_id VARCHAR(255),              -- ID from source system
    source_priority INT DEFAULT 100,      -- Lower number = higher priority
    latency_hours INT,                    -- Data freshness in hours
    is_official BOOLEAN DEFAULT FALSE,    -- Official vs derivative data
    
    -- Ingestion Metadata
    ingested_at TIMESTAMPTZ DEFAULT NOW(),
    ingested_by UUID,
    raw_payload JSONB,                    -- Original source data for audit
    
    INDEX idx_mdm_source_tenant_date (tenant_id, calendar_date),
    INDEX idx_mdm_source_system (tenant_id, source_system),
    INDEX idx_mdm_source_ingested (tenant_id, ingested_at DESC),
    INDEX idx_mdm_source_external_id (tenant_id, external_id)
);

-- ============================================================================
-- 3. LINEAGE TABLE (Audit Trail)
-- ============================================================================
CREATE TABLE IF NOT EXISTS mdm_calendar_lineage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_record_id UUID NOT NULL REFERENCES mdm_calendar_golden(id) ON DELETE CASCADE,
    
    -- Semantic Change Tracking
    semantic_term VARCHAR(50) NOT NULL,  -- e.g., "IsBusinessDay", "HolidayName"
    previous_value TEXT,
    winning_value TEXT,
    winning_source_id UUID REFERENCES mdm_calendar_source(id) ON DELETE SET NULL,
    
    -- Rule Execution Details
    rule_applied VARCHAR(255),           -- e.g., "Priority 1: ExchangeOfficial"
    priority_level INT,                  -- Rule priority (1-N)
    wasm_execution_id UUID,              -- Link to WASM execution trace
    execution_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Alternative Sources (for conflict detection)
    conflicting_sources JSONB,           -- Alternative candidates that lost
    conflict_detected BOOLEAN DEFAULT FALSE,
    
    INDEX idx_mdm_lineage_golden (tenant_id, golden_record_id),
    INDEX idx_mdm_lineage_term (tenant_id, semantic_term),
    INDEX idx_mdm_lineage_executed (tenant_id, execution_timestamp DESC)
);

-- ============================================================================
-- 4. CONFLICT FLAGS TABLE (Stewardship Queue)
-- ============================================================================
CREATE TABLE IF NOT EXISTS mdm_calendar_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_record_id UUID NOT NULL REFERENCES mdm_calendar_golden(id) ON DELETE CASCADE,
    
    -- Conflict Details
    conflict_type VARCHAR(50) NOT NULL,  -- e.g., "HighPriorityDisagreement", "MissingOfficial"
    conflicting_sources JSONB NOT NULL,  -- Array of competing candidates
    severity VARCHAR(20) DEFAULT 'medium', -- low, medium, high, critical
    
    -- Resolution Status
    status VARCHAR(50) DEFAULT 'open',   -- open, in_review, resolved, rejected
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,
    
    -- Escalation
    escalated_to_steward_at TIMESTAMPTZ,
    steward_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_mdm_conflicts_tenant (tenant_id, status),
    INDEX idx_mdm_conflicts_created (tenant_id, created_at DESC),
    INDEX idx_mdm_conflicts_severity (tenant_id, severity)
);

-- ============================================================================
-- 5. VERSIONING/TIME-TRAVEL TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS mdm_calendar_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    golden_record_id UUID NOT NULL REFERENCES mdm_calendar_golden(id) ON DELETE CASCADE,
    
    -- Version Info
    version_id INT NOT NULL,
    version_timestamp TIMESTAMPTZ NOT NULL,
    
    -- Snapshot of Full Record
    calendar_date DATE NOT NULL,
    is_business_day BOOLEAN NOT NULL,
    region_code VARCHAR(2) NOT NULL,
    exchange_code VARCHAR(4),
    holiday_name VARCHAR(255),
    source_type VARCHAR(50),
    confidence_score INT,
    
    -- Change Log
    change_reason VARCHAR(255),
    changed_by UUID,
    change_timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_mdm_versions_golden (tenant_id, golden_record_id),
    INDEX idx_mdm_versions_date (tenant_id, version_timestamp DESC)
);

-- ============================================================================
-- 6. HEALTH METRICS TABLE
-- ============================================================================
CREATE TABLE IF NOT EXISTS mdm_calendar_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Coverage Metrics
    total_calendar_days INT,
    days_with_golden_records INT,
    coverage_percentage NUMERIC(5, 2),
    
    -- Data Quality Metrics
    conflict_count INT DEFAULT 0,
    high_confidence_count INT,
    medium_confidence_count INT,
    low_confidence_count INT,
    
    -- Freshness Metrics
    days_since_last_official_feed INT,
    last_official_feed_timestamp TIMESTAMPTZ,
    
    -- Audit
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_mdm_metrics_tenant (tenant_id, calculated_at DESC)
);

-- ============================================================================
-- 7. ENABLE ROW-LEVEL SECURITY (RLS)
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE mdm_calendar_golden ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_source ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_lineage ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_conflicts ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm_calendar_metrics ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- 8. RLS POLICIES: Tenant Isolation
-- ============================================================================

-- Golden Records: Tenant isolation
CREATE POLICY mdm_golden_tenant_isolation ON mdm_calendar_golden
    FOR ALL
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id));

-- Source Records: Tenant isolation
CREATE POLICY mdm_source_tenant_isolation ON mdm_calendar_source
    FOR ALL
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id));

-- Lineage: Tenant isolation
CREATE POLICY mdm_lineage_tenant_isolation ON mdm_calendar_lineage
    FOR ALL
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id));

-- Conflicts: Tenant isolation
CREATE POLICY mdm_conflicts_tenant_isolation ON mdm_calendar_conflicts
    FOR ALL
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id));

-- Versions: Tenant isolation
CREATE POLICY mdm_versions_tenant_isolation ON mdm_calendar_versions
    FOR ALL
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id));

-- Metrics: Tenant isolation
CREATE POLICY mdm_metrics_tenant_isolation ON mdm_calendar_metrics
    FOR ALL
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id))
    WITH CHECK (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::UUID, tenant_id));

-- ============================================================================
-- 9. AUDIT TRIGGERS
-- ============================================================================

-- Updated_at trigger for golden records
CREATE OR REPLACE FUNCTION mdm_calendar_golden_updated()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE PLPGSQL;

CREATE TRIGGER mdm_calendar_golden_update_trigger
BEFORE UPDATE ON mdm_calendar_golden
FOR EACH ROW
EXECUTE FUNCTION mdm_calendar_golden_updated();

-- ============================================================================
-- 10. HEALTH CHECK VIEWS
-- ============================================================================

-- Coverage dashboard
CREATE OR REPLACE VIEW mdm_calendar_coverage AS
SELECT 
    tenant_id,
    COUNT(DISTINCT calendar_date) as total_records,
    COUNT(DISTINCT calendar_date) FILTER (WHERE is_business_day = true) as business_days,
    COUNT(DISTINCT calendar_date) FILTER (WHERE is_business_day = false) as holidays,
    COUNT(DISTINCT region_code) as distinct_regions,
    MIN(calendar_date) as earliest_date,
    MAX(calendar_date) as latest_date,
    AVG(confidence_score::NUMERIC) FILTER (WHERE confidence_score > 0)::INT as avg_confidence
FROM mdm_calendar_golden
WHERE is_deleted = FALSE
GROUP BY tenant_id;

-- Conflict summary
CREATE OR REPLACE VIEW mdm_calendar_conflicts_summary AS
SELECT 
    tenant_id,
    COUNT(*) FILTER (WHERE status = 'open') as open_conflicts,
    COUNT(*) FILTER (WHERE status = 'in_review') as in_review_conflicts,
    COUNT(*) FILTER (WHERE severity = 'critical') as critical_conflicts,
    MAX(created_at) as most_recent_conflict
FROM mdm_calendar_conflicts
GROUP BY tenant_id;

-- Source contribution stats
CREATE OR REPLACE VIEW mdm_calendar_source_stats AS
SELECT 
    tenant_id,
    source_system,
    COUNT(DISTINCT calendar_date) as records_ingested,
    COUNT(DISTINCT CASE WHEN golden_record_id IS NOT NULL THEN 1 END) as records_won,
    ROUND(100.0 * COUNT(DISTINCT CASE WHEN golden_record_id IS NOT NULL THEN 1 END) 
        / NULLIF(COUNT(DISTINCT calendar_date), 0), 2) as win_percentage
FROM mdm_calendar_source
GROUP BY tenant_id, source_system;

-- ============================================================================
-- 11. INDEXES FOR PERFORMANCE
-- ============================================================================

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_golden_range_query 
    ON mdm_calendar_golden(tenant_id, calendar_date, region_code) 
    WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS idx_golden_by_business_day 
    ON mdm_calendar_golden(tenant_id, region_code, is_business_day) 
    WHERE is_deleted = FALSE;

-- GiST index for date range lookups
CREATE INDEX IF NOT EXISTS idx_golden_date_range 
    ON mdm_calendar_golden USING GIST (tenant_id, daterange(calendar_date, calendar_date, '[]'));

-- Lineage query optimization
CREATE INDEX IF NOT EXISTS idx_lineage_by_term_and_golden 
    ON mdm_calendar_lineage(tenant_id, semantic_term, golden_record_id);
