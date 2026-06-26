-- Phase 4: Holiday Schema & AI Suggestions Database
-- Production-ready schema for AI-generated holiday management
-- Status: Ready to deploy
-- Migration Strategy: Blue-green (create new tables, migrate data, switch)

BEGIN;

-- ============================================================================
-- 1. Core Holidays Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS holidays (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  region VARCHAR(50) NOT NULL,
  
  -- Holiday metadata
  name VARCHAR(255) NOT NULL,
  description TEXT,
  holiday_type VARCHAR(50) NOT NULL, -- 'national', 'regional', 'company', 'cultural'
  
  -- Date range (supports multi-day holidays)
  date_start DATE NOT NULL,
  date_end DATE NOT NULL,
  
  -- Recurrence pattern
  is_recurring BOOLEAN DEFAULT FALSE,
  recurring_pattern VARCHAR(50), -- 'annual', 'monthly', 'weekly', NULL for one-time
  recurring_end_date DATE, -- NULL means indefinite
  
  -- Approval workflow
  status VARCHAR(50) DEFAULT 'approved', -- 'draft', 'approved', 'archived'
  
  -- Audit trail
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  archived_at TIMESTAMP WITH TIME ZONE,
  
  -- Constraints
  CONSTRAINT valid_date_range CHECK (date_end >= date_start),
  CONSTRAINT valid_recurrence CHECK (
    (is_recurring = FALSE AND recurring_pattern IS NULL) OR
    (is_recurring = TRUE AND recurring_pattern IN ('annual', 'monthly', 'weekly'))
  ),
  CONSTRAINT valid_status CHECK (status IN ('draft', 'approved', 'archived')),
  UNIQUE(tenant_id, region, date_start, name)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_holidays_tenant_region_date 
  ON holidays(tenant_id, region, date_start)
  WHERE status = 'approved';

CREATE INDEX IF NOT EXISTS idx_holidays_recurring 
  ON holidays(tenant_id, recurring_pattern)
  WHERE is_recurring = TRUE AND status = 'approved';

CREATE INDEX IF NOT EXISTS idx_holidays_date_range 
  ON holidays(date_start, date_end)
  WHERE status = 'approved';

-- ============================================================================
-- 2. Pending Holiday Suggestions (AI-Generated, Awaiting Admin Approval)
-- ============================================================================

CREATE TABLE IF NOT EXISTS pending_holiday_suggestions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  region VARCHAR(50) NOT NULL,
  
  -- Link to Temporal workflow
  workflow_id VARCHAR(255) NOT NULL,
  
  -- Suggested holidays (JSON array of proposals)
  -- Format: [{name, date_start, date_end, holiday_type, confidence, reason, conflicts: [...]}]
  suggestions JSONB NOT NULL DEFAULT '[]',
  
  -- Generation metadata
  generation_params JSONB, -- {industry, language, year, exclude_historic}
  ai_model VARCHAR(50) DEFAULT 'gpt-4o-mini',
  
  -- Status tracking
  status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'expired'
  rejection_reason TEXT,
  
  -- Admin review
  reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  
  -- Temporal limits
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  expires_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '24 hours',
  
  -- Audit
  approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
  approved_at TIMESTAMP WITH TIME ZONE,
  
  -- Constraints
  CONSTRAINT valid_status CHECK (status IN ('pending', 'approved', 'rejected', 'expired')),
  CONSTRAINT requires_review_for_rejection CHECK (
    (status != 'rejected') OR (rejection_reason IS NOT NULL)
  )
);

CREATE INDEX IF NOT EXISTS idx_suggestions_tenant_status 
  ON pending_holiday_suggestions(tenant_id, status)
  WHERE status IN ('pending', 'approved');

CREATE INDEX IF NOT EXISTS idx_suggestions_expires 
  ON pending_holiday_suggestions(expires_at DESC)
  WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_suggestions_workflow 
  ON pending_holiday_suggestions(workflow_id);

-- ============================================================================
-- 3. Holiday Conflicts (AI-Detected Issues)
-- ============================================================================

CREATE TABLE IF NOT EXISTS holiday_conflicts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  holiday_id UUID NOT NULL REFERENCES holidays(id) ON DELETE CASCADE,
  
  -- Conflict analysis
  conflict_type VARCHAR(50) NOT NULL,
  -- 'overlap': overlaps with existing booking
  -- 'capacity': insufficient capacity during holiday
  -- 'resource': required resource unavailable
  -- 'provider_conflict': multiple providers affected
  -- 'external': external system conflict
  
  severity VARCHAR(50) NOT NULL, -- 'info', 'low', 'medium', 'high', 'critical'
  description TEXT NOT NULL,
  suggested_resolution TEXT,
  
  -- Resolution status
  status VARCHAR(50) DEFAULT 'open', -- 'open', 'acknowledged', 'resolved', 'wontfix'
  resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
  resolved_at TIMESTAMP WITH TIME ZONE,
  
  -- Audit
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  -- Constraints
  CONSTRAINT valid_conflict_type CHECK (conflict_type IN (
    'overlap', 'capacity', 'resource', 'provider_conflict', 'external'
  )),
  CONSTRAINT valid_severity CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  CONSTRAINT valid_status CHECK (status IN ('open', 'acknowledged', 'resolved', 'wontfix'))
);

CREATE INDEX IF NOT EXISTS idx_conflicts_tenant_severity 
  ON holiday_conflicts(tenant_id, severity)
  WHERE status IN ('open', 'acknowledged');

CREATE INDEX IF NOT EXISTS idx_conflicts_holiday 
  ON holiday_conflicts(holiday_id);

-- ============================================================================
-- 4. AI Interaction Logs (Audit Trail for AI Operations)
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai_interaction_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  
  -- Operation tracking
  operation_type VARCHAR(50) NOT NULL,
  -- 'generate_holidays', 'detect_conflicts', 'validate_capacity', 'sync_regions'
  
  workflow_id VARCHAR(255), -- Temporal workflow ID (nullable for non-workflow ops)
  
  -- Request/Response data (for debugging)
  input_params JSONB,
  ai_response JSONB,
  
  -- Token tracking (for cost estimation)
  tokens_used INT DEFAULT 0,
  estimated_cost_cents DECIMAL(10, 2) DEFAULT 0,
  
  -- Status & error handling
  status VARCHAR(50) NOT NULL, -- 'success', 'error', 'timeout', 'rate_limited'
  error_message TEXT,
  error_code VARCHAR(50),
  
  -- Performance metrics
  execution_time_ms INT,
  
  -- Audit
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  -- Constraints
  CONSTRAINT valid_operation CHECK (operation_type IN (
    'generate_holidays', 'detect_conflicts', 'validate_capacity', 'sync_regions'
  )),
  CONSTRAINT valid_status CHECK (status IN ('success', 'error', 'timeout', 'rate_limited'))
);

CREATE INDEX IF NOT EXISTS idx_ai_logs_tenant_operation 
  ON ai_interaction_logs(tenant_id, operation_type)
  WHERE status != 'success';

CREATE INDEX IF NOT EXISTS idx_ai_logs_workflow 
  ON ai_interaction_logs(workflow_id)
  WHERE workflow_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ai_logs_created 
  ON ai_interaction_logs(created_at DESC);

-- ============================================================================
-- 5. AI Adoption Metrics (Aggregated for Dashboard)
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai_adoption_metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  
  -- Date of measurement
  metric_date DATE NOT NULL,
  
  -- Adoption metrics
  suggestions_generated INT DEFAULT 0,
  suggestions_approved INT DEFAULT 0,
  suggestions_rejected INT DEFAULT 0,
  approval_rate DECIMAL(5, 2), -- percentage
  
  -- Cost tracking
  total_tokens_used INT DEFAULT 0,
  api_calls_made INT DEFAULT 0,
  estimated_cost_cents DECIMAL(10, 2) DEFAULT 0,
  
  -- ROI metrics (Phase 5+)
  time_saved_minutes INT DEFAULT 0,
  conflicts_detected INT DEFAULT 0,
  
  -- Computed at aggregation time
  computed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  UNIQUE(tenant_id, metric_date)
);

CREATE INDEX IF NOT EXISTS idx_metrics_tenant_date 
  ON ai_adoption_metrics(tenant_id, metric_date DESC);

-- ============================================================================
-- 6. Market Calendars (International Foundation - Phase 4+)
-- ============================================================================

CREATE TABLE IF NOT EXISTS market_calendars (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  
  -- Calendar identification
  market_code VARCHAR(10) NOT NULL, -- 'NYSE', 'LSE', 'JSX', etc.
  region VARCHAR(50) NOT NULL,
  exchange_name VARCHAR(255) NOT NULL,
  
  -- Trading calendar
  holidays JSONB NOT NULL DEFAULT '[]', -- Array of annual holidays
  half_days JSONB DEFAULT '[]', -- Array of half trading days
  
  -- Metadata
  timezone VARCHAR(50),
  opening_time TIME,
  closing_time TIME,
  
  -- Status
  is_active BOOLEAN DEFAULT TRUE,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  UNIQUE(market_code, region),
  CONSTRAINT valid_market CHECK (
    market_code IN ('NYSE', 'NASDAQ', 'LSE', 'EURONEXT', 'JSX', 'HKX', 'SSE', 'NSE')
  )
);

CREATE INDEX IF NOT EXISTS idx_market_calendars_region 
  ON market_calendars(region)
  WHERE is_active = TRUE;

-- ============================================================================
-- 7. Profile Market Calendar Assignments (Multi-Region Linking)
-- ============================================================================

CREATE TABLE IF NOT EXISTS profile_market_calendars (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
  market_calendar_id UUID NOT NULL REFERENCES market_calendars(id) ON DELETE CASCADE,
  
  -- Priority (if profile uses multiple markets)
  priority INT DEFAULT 1,
  
  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  UNIQUE(profile_id, market_calendar_id)
);

CREATE INDEX IF NOT EXISTS idx_profile_markets_profile 
  ON profile_market_calendars(profile_id);

CREATE INDEX IF NOT EXISTS idx_profile_markets_tenant 
  ON profile_market_calendars(tenant_id);

-- ============================================================================
-- 8. Row-Level Security (RLS) Policies
-- ============================================================================

ALTER TABLE holidays ENABLE ROW LEVEL SECURITY;
ALTER TABLE pending_holiday_suggestions ENABLE ROW LEVEL SECURITY;
ALTER TABLE holiday_conflicts ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_interaction_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_adoption_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE market_calendars ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_market_calendars ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see holidays for their tenant & authorized regions
CREATE POLICY IF NOT EXISTS holidays_tenant_isolation 
  ON holidays 
  FOR ALL 
  USING (
    tenant_id = current_setting('app.tenant_id')::UUID
  );

CREATE POLICY IF NOT EXISTS suggestions_tenant_isolation 
  ON pending_holiday_suggestions 
  FOR ALL 
  USING (
    tenant_id = current_setting('app.tenant_id')::UUID
  );

CREATE POLICY IF NOT EXISTS conflicts_tenant_isolation 
  ON holiday_conflicts 
  FOR ALL 
  USING (
    tenant_id = (SELECT tenant_id FROM holidays WHERE id = holiday_id)
  );

CREATE POLICY IF NOT EXISTS ai_logs_tenant_isolation 
  ON ai_interaction_logs 
  FOR ALL 
  USING (
    tenant_id = current_setting('app.tenant_id')::UUID
  );

CREATE POLICY IF NOT EXISTS metrics_tenant_isolation 
  ON ai_adoption_metrics 
  FOR ALL 
  USING (
    tenant_id = current_setting('app.tenant_id')::UUID
  );

-- ============================================================================
-- 9. Pre-Migration Verification View
-- ============================================================================

CREATE OR REPLACE VIEW v_holiday_migration_status AS
SELECT
  'holidays' as table_name,
  COUNT(*) as record_count,
  MAX(created_at) as latest_record
FROM holidays
UNION ALL
SELECT
  'pending_holiday_suggestions',
  COUNT(*),
  MAX(created_at)
FROM pending_holiday_suggestions
UNION ALL
SELECT
  'holiday_conflicts',
  COUNT(*),
  MAX(created_at)
FROM holiday_conflicts
UNION ALL
SELECT
  'ai_interaction_logs',
  COUNT(*),
  MAX(created_at)
FROM ai_interaction_logs;

-- ============================================================================
-- 10. Comments for Documentation
-- ============================================================================

COMMENT ON TABLE holidays IS 'Holiday calendar entries, both one-time and recurring';
COMMENT ON TABLE pending_holiday_suggestions IS 'AI-generated holiday suggestions pending admin approval';
COMMENT ON TABLE holiday_conflicts IS 'Conflicts detected by AI analysis during holiday validation';
COMMENT ON TABLE ai_interaction_logs IS 'Audit trail of all AI API calls for debugging and cost tracking';
COMMENT ON TABLE ai_adoption_metrics IS 'Aggregated adoption metrics for dashboard and reporting';
COMMENT ON TABLE market_calendars IS 'Trading/market calendars for multiple global exchanges';
COMMENT ON TABLE profile_market_calendars IS 'Link profiles to relevant market calendars for multi-region support';

COMMENT ON COLUMN holidays.recurring_pattern IS 'Pattern: annual (same date each year), monthly, or weekly';
COMMENT ON COLUMN pending_holiday_suggestions.suggestions IS 'JSON array of {name, date_start, date_end, holiday_type, confidence (0-1), reason}';
COMMENT ON COLUMN ai_interaction_logs.operation_type IS 'Type of AI operation: generate_holidays, detect_conflicts, validate_capacity, sync_regions';

COMMIT;

-- ============================================================================
-- Rollback Instructions (if needed)
-- ============================================================================

-- DROP TABLE IF EXISTS profile_market_calendars CASCADE;
-- DROP TABLE IF EXISTS market_calendars CASCADE;
-- DROP TABLE IF EXISTS ai_adoption_metrics CASCADE;
-- DROP TABLE IF EXISTS ai_interaction_logs CASCADE;
-- DROP TABLE IF EXISTS holiday_conflicts CASCADE;
-- DROP TABLE IF EXISTS pending_holiday_suggestions CASCADE;
-- DROP TABLE IF EXISTS holidays CASCADE;
-- DROP VIEW IF EXISTS v_holiday_migration_status;
