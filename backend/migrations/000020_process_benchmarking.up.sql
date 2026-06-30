-- Process Benchmarking System Database Schema
-- Supports industry benchmarks, performance scoring, peer comparison, and best practices

-- ============================================================================
-- Industry Benchmarks Table
-- Stores aggregated performance metrics by industry and process type
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_industry_benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    industry VARCHAR(100) NOT NULL,
    process_type VARCHAR(100) NOT NULL,
    
    -- Performance Metrics
    median_duration_minutes DECIMAL(10,2),
    top_quartile_duration_minutes DECIMAL(10,2),
    median_success_rate DECIMAL(5,4),
    top_quartile_success_rate DECIMAL(5,4),
    median_cost_per_process DECIMAL(12,2),
    top_quartile_cost_per_process DECIMAL(12,2),
    median_automation_rate DECIMAL(5,4),
    top_quartile_automation_rate DECIMAL(5,4),
    
    -- Metadata
    sample_size INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data_source VARCHAR(200),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(industry, process_type)
);

CREATE INDEX IF NOT EXISTS idx_bp_industry_benchmarks_industry ON bp_industry_benchmarks(industry);
CREATE INDEX IF NOT EXISTS idx_bp_industry_benchmarks_process ON bp_industry_benchmarks(process_type);

-- ============================================================================
-- Performance Scores Table
-- Stores calculated performance scores for each tenant's processes
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_performance_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workflow_type VARCHAR(100) NOT NULL,
    
    -- Overall Score (0-100)
    overall_score INTEGER NOT NULL CHECK (overall_score >= 0 AND overall_score <= 100),
    grade VARCHAR(3) NOT NULL, -- A+, A, B+, B, C+, C, D, F
    percentile INTEGER CHECK (percentile >= 0 AND percentile <= 100),
    
    -- Dimension Scores (0-100 each)
    efficiency_score INTEGER NOT NULL CHECK (efficiency_score >= 0 AND efficiency_score <= 100),
    quality_score INTEGER NOT NULL CHECK (quality_score >= 0 AND quality_score <= 100),
    speed_score INTEGER NOT NULL CHECK (speed_score >= 0 AND speed_score <= 100),
    automation_score INTEGER NOT NULL CHECK (automation_score >= 0 AND automation_score <= 100),
    compliance_score INTEGER NOT NULL CHECK (compliance_score >= 0 AND compliance_score <= 100),
    
    -- Calculation Metadata
    industry VARCHAR(100),
    sample_size INTEGER DEFAULT 0,
    confidence_level DECIMAL(5,4) DEFAULT 0.95,
    
    -- Timestamps
    calculated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, workflow_type)
);

CREATE INDEX IF NOT EXISTS idx_bp_performance_scores_tenant ON bp_performance_scores(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_performance_scores_workflow ON bp_performance_scores(workflow_type);
CREATE INDEX IF NOT EXISTS idx_bp_performance_scores_industry ON bp_performance_scores(industry);
CREATE INDEX IF NOT EXISTS idx_bp_performance_scores_overall ON bp_performance_scores(overall_score DESC);

-- ============================================================================
-- Best Practices Table
-- Curated library of industry-proven optimization strategies
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_best_practices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    industry VARCHAR(100),
    process_type VARCHAR(100),
    category VARCHAR(50), -- automation, quality, efficiency, compliance, etc.
    
    -- Impact Assessment
    expected_improvement_percent INTEGER CHECK (expected_improvement_percent >= 0 AND expected_improvement_percent <= 100),
    implementation_effort VARCHAR(20) CHECK (implementation_effort IN ('low', 'medium', 'high')),
    implementation_time_weeks INTEGER,
    
    -- Adoption Metrics
    industry_adoption_percent INTEGER CHECK (industry_adoption_percent >= 0 AND industry_adoption_percent <= 100),
    success_rate DECIMAL(5,4),
    
    -- Implementation Guide
    prerequisites TEXT,
    implementation_steps JSONB,
    required_tools TEXT[],
    estimated_cost_range VARCHAR(50),
    
    -- Case Study (optional)
    case_study_company VARCHAR(100),
    case_study_results TEXT,
    case_study_timeline VARCHAR(100),
    
    -- Metadata
    priority VARCHAR(20) CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    tags TEXT[],
    external_resources JSONB, -- URLs, whitepapers, etc.
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bp_best_practices_industry ON bp_best_practices(industry);
CREATE INDEX IF NOT EXISTS idx_bp_best_practices_process ON bp_best_practices(process_type);
CREATE INDEX IF NOT EXISTS idx_bp_best_practices_category ON bp_best_practices(category);
CREATE INDEX IF NOT EXISTS idx_bp_best_practices_priority ON bp_best_practices(priority);

-- ============================================================================
-- Peer Groups Table
-- Defines comparison groups for peer benchmarking
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_peer_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    industry VARCHAR(100) NOT NULL,
    
    -- Group Criteria
    company_size_min INTEGER,
    company_size_max INTEGER,
    geography VARCHAR(100),
    annual_revenue_min DECIMAL(15,2),
    annual_revenue_max DECIMAL(15,2),
    
    -- Metadata
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bp_peer_groups_industry ON bp_peer_groups(industry);
CREATE INDEX IF NOT EXISTS idx_bp_peer_groups_active ON bp_peer_groups(is_active);

-- ============================================================================
-- Peer Group Members Table
-- Maps tenants to their peer groups
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_peer_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    peer_group_id UUID NOT NULL REFERENCES bp_peer_groups(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Membership Info
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Optional Metadata
    company_size INTEGER,
    annual_revenue DECIMAL(15,2),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(peer_group_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_bp_peer_group_members_peer_group ON bp_peer_group_members(peer_group_id);
CREATE INDEX IF NOT EXISTS idx_bp_peer_group_members_tenant ON bp_peer_group_members(tenant_id);

-- ============================================================================
-- Gap Analysis Table
-- Stores identified performance gaps and recommendations
-- ============================================================================

CREATE TABLE IF NOT EXISTS bp_gap_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workflow_type VARCHAR(100) NOT NULL,
    dimension VARCHAR(50) NOT NULL, -- efficiency, quality, speed, automation, compliance
    
    -- Gap Details
    current_score INTEGER NOT NULL CHECK (current_score >= 0 AND current_score <= 100),
    target_score INTEGER NOT NULL CHECK (target_score >= 0 AND target_score <= 100),
    gap_points INTEGER NOT NULL, -- target - current
    priority VARCHAR(20) CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    
    -- Recommendations
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    recommended_action TEXT,
    expected_improvement INTEGER, -- estimated point improvement
    implementation_timeline VARCHAR(100),
    
    -- Related Best Practices
    related_best_practice_ids UUID[],
    
    -- Status
    status VARCHAR(20) DEFAULT 'identified' CHECK (status IN ('identified', 'in_progress', 'completed', 'dismissed')),
    resolution_notes TEXT,
    
    -- Timestamps
    identified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bp_gap_analysis_tenant ON bp_gap_analysis(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_gap_analysis_workflow ON bp_gap_analysis(workflow_type);
CREATE INDEX IF NOT EXISTS idx_bp_gap_analysis_dimension ON bp_gap_analysis(dimension);
CREATE INDEX IF NOT EXISTS idx_bp_gap_analysis_priority ON bp_gap_analysis(priority);
CREATE INDEX IF NOT EXISTS idx_bp_gap_analysis_status ON bp_gap_analysis(status);

-- ============================================================================
-- Trigger to update updated_at timestamps
-- ============================================================================

CREATE OR REPLACE FUNCTION update_bp_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_bp_performance_scores_updated_at ON bp_performance_scores;
CREATE TRIGGER update_bp_performance_scores_updated_at
    BEFORE UPDATE ON bp_performance_scores
    FOR EACH ROW
    EXECUTE FUNCTION update_bp_updated_at();

DROP TRIGGER IF EXISTS update_bp_best_practices_updated_at ON bp_best_practices;
CREATE TRIGGER update_bp_best_practices_updated_at
    BEFORE UPDATE ON bp_best_practices
    FOR EACH ROW
    EXECUTE FUNCTION update_bp_updated_at();

DROP TRIGGER IF EXISTS update_bp_peer_groups_updated_at ON bp_peer_groups;
CREATE TRIGGER update_bp_peer_groups_updated_at
    BEFORE UPDATE ON bp_peer_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_bp_updated_at();

DROP TRIGGER IF EXISTS update_bp_gap_analysis_updated_at ON bp_gap_analysis;
CREATE TRIGGER update_bp_gap_analysis_updated_at
    BEFORE UPDATE ON bp_gap_analysis
    FOR EACH ROW
    EXECUTE FUNCTION update_bp_updated_at();

-- ============================================================================
-- Comments for Documentation
-- ============================================================================

COMMENT ON TABLE bp_industry_benchmarks IS 'Industry-wide performance benchmarks aggregated from market research';
COMMENT ON TABLE bp_performance_scores IS 'Calculated performance scores for tenant processes with dimension breakdowns';
COMMENT ON TABLE bp_best_practices IS 'Curated library of industry-proven optimization strategies';
COMMENT ON TABLE bp_peer_groups IS 'Peer group definitions for anonymous comparison';
COMMENT ON TABLE bp_peer_group_members IS 'Tenant membership in peer groups';
COMMENT ON TABLE bp_gap_analysis IS 'Identified performance gaps with prioritized recommendations';
