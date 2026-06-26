-- backend/internal/migrations/household_ledger.sql
-- Household Reports Schema (Ledger-Backed, Semantic-First)
-- Created: October 30, 2025
-- Status: Foundation for household aggregation + AI semantic cubes

-- ============================================================================
-- MAIN TABLES
-- ============================================================================

-- Households (top-level grouping, ledger-backed like BD)
CREATE TABLE IF NOT EXISTS households (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  
  -- Household metadata
  head_of_household_name VARCHAR(255),
  household_type VARCHAR(50), -- 'individual', 'family', 'trust', 'entity'
  
  -- Ledger integration
  ledger_id UUID,  -- Link to ledger_accounts for reconciliation
  
  -- Status
  status VARCHAR(20) DEFAULT 'active', -- 'active', 'inactive', 'archived'
  is_published BOOLEAN DEFAULT false,
  
  -- Timestamps
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  
  -- Constraints
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  UNIQUE(tenant_id, name)
);

-- Household members (ALTs, SMAs, advisors, etc.)
CREATE TABLE IF NOT EXISTS household_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  
  -- Member info
  member_type VARCHAR(50) NOT NULL, -- 'alt', 'sma', 'advisor', 'beneficiary'
  member_id UUID NOT NULL,  -- Foreign key to accounts/users
  member_name VARCHAR(255),
  
  -- Ledger entry (for ALTs/SMAs)
  ledger_entity_id UUID,  -- Link to ledger_accounts
  
  -- Flags
  is_primary BOOLEAN DEFAULT false,
  is_active BOOLEAN DEFAULT true,
  
  -- Timestamps
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Household semantic views (like BD "Group By Anything")
-- Each household can aggregate multiple semantic views
CREATE TABLE IF NOT EXISTS household_semantic_mappings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  
  -- Semantic view reference
  semantic_view_id UUID NOT NULL,
  view_name VARCHAR(255),
  
  -- Custom grouping (JSON tags, like BD groups)
  group_by_fields JSONB,  -- e.g. {"liquidity": "liquid|illiquid", "generation": "gen_1|gen_2"}
  
  -- Filters (only aggregate matching entities)
  filter_conditions JSONB,  -- e.g. {"status": "active", "min_value": 100000}
  
  -- Weight/allocation
  allocation_weight NUMERIC(5, 2) DEFAULT 1.0,
  
  -- Status
  is_active BOOLEAN DEFAULT true,
  
  -- Timestamps
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Household reports (paginated, cube-based)
CREATE TABLE IF NOT EXISTS household_reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  household_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  
  -- Report metadata
  report_name VARCHAR(255) NOT NULL,
  report_type VARCHAR(50), -- 'summary', 'detailed', 'performance', 'allocation'
  
  -- Report config (ParameterBuilder schema-driven)
  report_config JSONB,  -- Stores selected parameters from schema
  
  -- Semantic cube reference (AI-generated)
  semantic_cube_id UUID,
  semantic_cube_data JSONB,  -- Cached cube structure
  
  -- PDF storage
  pdf_data BYTEA,
  pdf_generated_at TIMESTAMPTZ,
  pdf_file_name VARCHAR(255),
  
  -- Drill-down paths
  drill_paths JSONB,  -- e.g. {"alts": [alt_id_1, alt_id_2], "smas": [...]}
  
  -- Report pages/sections
  page_count INT,
  section_count INT,
  
  -- Status
  status VARCHAR(20) DEFAULT 'draft', -- 'draft', 'generated', 'error'
  generation_error TEXT,
  
  -- Timestamps
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  generated_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ, -- Auto-delete old reports
  
  FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  UNIQUE(household_id, report_name)
);

-- Report execution log (audit trail)
CREATE TABLE IF NOT EXISTS household_report_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id UUID NOT NULL,
  household_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  
  -- Execution details
  action VARCHAR(50), -- 'created', 'generated', 'viewed', 'downloaded', 'deleted'
  user_id UUID,
  user_email VARCHAR(255),
  
  -- Performance metrics
  generation_time_ms INT,
  pdf_size_bytes INT,
  
  -- Metadata
  metadata JSONB,
  
  -- Timestamp
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (report_id) REFERENCES household_reports(id) ON DELETE CASCADE,
  FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Household lookups
CREATE INDEX idx_households_tenant ON households(tenant_id);
CREATE INDEX idx_households_status ON households(status);
CREATE INDEX idx_households_tenant_status ON households(tenant_id, status);

-- Member lookups
CREATE INDEX idx_household_members_household ON household_members(household_id);
CREATE INDEX idx_household_members_tenant ON household_members(tenant_id);
CREATE INDEX idx_household_members_type ON household_members(member_type);

-- Semantic mapping lookups
CREATE INDEX idx_household_semantic_mappings_household ON household_semantic_mappings(household_id);
CREATE INDEX idx_household_semantic_mappings_view ON household_semantic_mappings(semantic_view_id);
CREATE INDEX idx_household_semantic_mappings_active ON household_semantic_mappings(is_active);

-- Report lookups
CREATE INDEX idx_household_reports_household ON household_reports(household_id);
CREATE INDEX idx_household_reports_tenant ON household_reports(tenant_id);
CREATE INDEX idx_household_reports_status ON household_reports(status);
CREATE INDEX idx_household_reports_created ON household_reports(created_at DESC);
CREATE INDEX idx_household_reports_tenant_household ON household_reports(tenant_id, household_id);

-- Report log lookups
CREATE INDEX idx_household_report_logs_report ON household_report_logs(report_id);
CREATE INDEX idx_household_report_logs_household ON household_report_logs(household_id);
CREATE INDEX idx_household_report_logs_tenant ON household_report_logs(tenant_id);
CREATE INDEX idx_household_report_logs_action ON household_report_logs(action);

-- ============================================================================
-- VIEWS
-- ============================================================================

-- Recent reports by household
CREATE VIEW IF NOT EXISTS household_reports_recent AS
SELECT 
  hr.id,
  hr.household_id,
  hr.tenant_id,
  hr.report_name,
  hr.status,
  hr.created_at,
  hr.generated_at,
  COUNT(DISTINCT hrl.id) as view_count,
  MAX(hrl.created_at) as last_viewed_at
FROM household_reports hr
LEFT JOIN household_report_logs hrl ON hr.id = hrl.report_id AND hrl.action = 'viewed'
GROUP BY hr.id, hr.household_id, hr.tenant_id, hr.report_name, hr.status, hr.created_at, hr.generated_at
ORDER BY hr.created_at DESC;

-- Household summary stats
CREATE VIEW IF NOT EXISTS household_stats AS
SELECT
  h.id,
  h.tenant_id,
  h.name,
  COUNT(DISTINCT hm.id) as member_count,
  COUNT(DISTINCT hsm.id) as semantic_view_count,
  COUNT(DISTINCT hr.id) as report_count,
  MAX(hr.created_at) as latest_report_date
FROM households h
LEFT JOIN household_members hm ON h.id = hm.household_id
LEFT JOIN household_semantic_mappings hsm ON h.id = hsm.household_id
LEFT JOIN household_reports hr ON h.id = hr.household_id
GROUP BY h.id, h.tenant_id, h.name;

-- ============================================================================
-- STORED PROCEDURES (Optional: cleanup old reports)
-- ============================================================================

-- Auto-delete reports older than 90 days
CREATE OR REPLACE FUNCTION cleanup_old_household_reports()
RETURNS void AS $$
BEGIN
  DELETE FROM household_reports
  WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '90 days'
  AND status IN ('archived');
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SEED DATA (Optional: test household)
-- ============================================================================

-- Example household (commented out, uncomment for testing)
-- INSERT INTO households (tenant_id, name, household_type, status)
-- VALUES (
--   (SELECT id FROM tenants LIMIT 1),
--   'Test Household - Smith Family',
--   'family',
--   'active'
-- );
